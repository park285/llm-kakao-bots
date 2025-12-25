package session

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/config"
	"github.com/park285/llm-kakao-bots/mcp-llm-server-go/internal/llm"
)

var (
	// ErrSessionNotFound 는 세션 미존재 오류다.
	ErrSessionNotFound = errors.New("session not found")
	// ErrStoreDisabled 는 저장소 비활성 오류다.
	ErrStoreDisabled = errors.New("session store disabled")
)

type storeBackend int

const (
	storeBackendMemory storeBackend = iota
	storeBackendValkey
)

// Meta 는 세션 메타데이터다.
type Meta struct {
	ID           string    `json:"id"`
	SystemPrompt string    `json:"system_prompt,omitempty"`
	Model        string    `json:"model,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
}

// Store 는 Valkey 기반 세션 저장소다.
type Store struct {
	client  valkey.Client
	cfg     *config.Config
	enabled bool
	backend storeBackend

	mu              sync.RWMutex
	meta            map[string]Meta
	history         map[string][]llm.HistoryEntry
	metaExpiresAt   map[string]time.Time
	historyExpireAt map[string]time.Time
}

// NewStore 는 세션 저장소를 생성한다.
func NewStore(cfg *config.Config) (*Store, error) {
	if cfg == nil {
		return nil, errors.New("config is nil")
	}

	if !cfg.SessionStore.Enabled {
		if cfg.SessionStore.Required {
			return nil, errors.New("session store required but disabled")
		}
		return newMemoryStore(cfg), nil
	}

	conn, err := parseStoreURL(cfg.SessionStore.URL)
	if err != nil {
		return nil, fmt.Errorf("parse session store url: %w", err)
	}

	var tlsConfig *tls.Config
	if conn.useTLS {
		host, _, splitErr := net.SplitHostPort(conn.addr)
		if splitErr != nil {
			return nil, fmt.Errorf("parse session store addr: %w", splitErr)
		}
		tlsConfig = &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host}
	}

	client, err := valkey.NewClient(valkey.ClientOption{
		TLSConfig:    tlsConfig,
		Username:     conn.username,
		Password:     conn.password,
		InitAddress:  []string{conn.addr},
		SelectDB:     conn.selectDB,
		DisableCache: cfg.SessionStore.DisableCache,
	})
	if err != nil {
		return nil, fmt.Errorf("connect to valkey: %w", err)
	}

	return &Store{
		client:  client,
		cfg:     cfg,
		enabled: true,
		backend: storeBackendValkey,
	}, nil
}

func newMemoryStore(cfg *config.Config) *Store {
	return &Store{
		cfg:             cfg,
		enabled:         true,
		backend:         storeBackendMemory,
		meta:            make(map[string]Meta),
		history:         make(map[string][]llm.HistoryEntry),
		metaExpiresAt:   make(map[string]time.Time),
		historyExpireAt: make(map[string]time.Time),
	}
}

// IsEnabled 는 저장소 활성화 여부를 반환한다.
func (s *Store) IsEnabled() bool {
	return s.enabled
}

// Close 는 Valkey 연결을 종료한다.
func (s *Store) Close() {
	if s == nil {
		return
	}
	if s.backend == storeBackendValkey && s.client != nil {
		s.client.Close()
	}
}

// metaKey 세션 메타데이터 키
func (s *Store) metaKey(sessionID string) string {
	return fmt.Sprintf("session:%s:meta", sessionID)
}

// historyKey 세션 히스토리 키
func (s *Store) historyKey(sessionID string) string {
	return fmt.Sprintf("session:%s:history", sessionID)
}

// ttl 세션 TTL
func (s *Store) ttl() time.Duration {
	return time.Duration(s.cfg.Session.SessionTTLMinutes) * time.Minute
}

// CreateSession 세션 생성
func (s *Store) CreateSession(ctx context.Context, meta Meta) error {
	if !s.enabled {
		return ErrStoreDisabled
	}
	if s.backend == storeBackendMemory {
		return s.createSessionMemory(meta)
	}

	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal session meta: %w", err)
	}

	cmd := s.client.B().Set().Key(s.metaKey(meta.ID)).Value(string(data)).Ex(s.ttl()).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

// GetSession 세션 메타데이터 조회
func (s *Store) GetSession(ctx context.Context, sessionID string) (*Meta, error) {
	if !s.enabled {
		return nil, ErrStoreDisabled
	}
	if s.backend == storeBackendMemory {
		return s.getSessionMemory(sessionID)
	}

	cmd := s.client.B().Get().Key(s.metaKey(sessionID)).Build()
	result, err := s.client.Do(ctx, cmd).ToString()
	if err != nil {
		if valkey.IsValkeyNil(err) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}

	var m Meta
	if err := json.Unmarshal([]byte(result), &m); err != nil {
		return nil, fmt.Errorf("unmarshal session meta: %w", err)
	}

	return &m, nil
}

// UpdateSession 세션 메타데이터 업데이트
func (s *Store) UpdateSession(ctx context.Context, meta Meta) error {
	if !s.enabled {
		return ErrStoreDisabled
	}
	if s.backend == storeBackendMemory {
		return s.updateSessionMemory(meta)
	}

	meta.UpdatedAt = time.Now()
	data, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal session meta: %w", err)
	}

	cmd := s.client.B().Set().Key(s.metaKey(meta.ID)).Value(string(data)).Ex(s.ttl()).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	return nil
}

// DeleteSession 세션 삭제
// DoMulti로 배치 처리하여 2 RTT → 1 RTT로 최적화
func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	if !s.enabled {
		return ErrStoreDisabled
	}
	if s.backend == storeBackendMemory {
		return s.deleteSessionMemory(sessionID)
	}

	metaCmd := s.client.B().Del().Key(s.metaKey(sessionID)).Build()
	historyCmd := s.client.B().Del().Key(s.historyKey(sessionID)).Build()

	results := s.client.DoMulti(ctx, metaCmd, historyCmd)
	for i, result := range results {
		if err := result.Error(); err != nil && !valkey.IsValkeyNil(err) {
			if i == 0 {
				return fmt.Errorf("delete session meta: %w", err)
			}
			return fmt.Errorf("delete session history: %w", err)
		}
	}
	return nil
}

// GetHistory 세션 히스토리 조회
func (s *Store) GetHistory(ctx context.Context, sessionID string) ([]llm.HistoryEntry, error) {
	if !s.enabled {
		return nil, ErrStoreDisabled
	}
	if s.backend == storeBackendMemory {
		return s.getHistoryMemory(sessionID), nil
	}

	cmd := s.client.B().Lrange().Key(s.historyKey(sessionID)).Start(0).Stop(-1).Build()
	results, err := s.client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("get history: %w", err)
	}

	history := make([]llm.HistoryEntry, 0, len(results))
	for _, item := range results {
		var entry llm.HistoryEntry
		if err := json.Unmarshal([]byte(item), &entry); err != nil {
			continue // skip invalid entries
		}
		history = append(history, entry)
	}

	return history, nil
}

// AppendHistory 히스토리에 메시지 추가
// DoMulti로 배치 처리하여 N+2 RTT → 1 RTT로 최적화
func (s *Store) AppendHistory(ctx context.Context, sessionID string, entries ...llm.HistoryEntry) error {
	if !s.enabled {
		return ErrStoreDisabled
	}
	if len(entries) == 0 {
		return nil
	}
	if s.backend == storeBackendMemory {
		return s.appendHistoryMemory(sessionID, entries...)
	}

	historyKey := s.historyKey(sessionID)

	// 모든 entry를 미리 직렬화
	elements := make([]string, 0, len(entries))
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("marshal history entry: %w", err)
		}
		elements = append(elements, string(data))
	}

	// 명령어 배치 구성: RPUSH + EXPIRE + (optional) LTRIM
	cmds := make([]valkey.Completed, 0, 3)

	// 단일 RPUSH로 모든 요소 추가
	rpushCmd := s.client.B().Rpush().Key(historyKey).Element(elements...).Build()
	cmds = append(cmds, rpushCmd)

	// TTL 갱신
	expireCmd := s.client.B().Expire().Key(historyKey).Seconds(int64(s.ttl().Seconds())).Build()
	cmds = append(cmds, expireCmd)

	// 히스토리 크기 제한
	maxPairs := s.cfg.Session.HistoryMaxPairs
	if maxPairs > 0 {
		trimCmd := s.client.B().Ltrim().Key(historyKey).Start(int64(-maxPairs * 2)).Stop(-1).Build()
		cmds = append(cmds, trimCmd)
	}

	// 모든 명령을 단일 RTT로 실행
	results := s.client.DoMulti(ctx, cmds...)
	if err := results[0].Error(); err != nil {
		return fmt.Errorf("append history: %w", err)
	}

	return nil
}

// SessionCount 현재 세션 수 (근사치)
// SCAN 기반으로 구현하여 O(N) 블로킹 KEYS 명령 대신 논블로킹 처리
func (s *Store) SessionCount(ctx context.Context) (int, error) {
	if !s.enabled {
		return 0, nil
	}
	if s.backend == storeBackendMemory {
		return s.sessionCountMemory(), nil
	}

	var count int
	var cursor uint64
	for {
		cmd := s.client.B().Scan().Cursor(cursor).Match("session:*:meta").Count(100).Build()
		result, err := s.client.Do(ctx, cmd).AsScanEntry()
		if err != nil {
			return 0, fmt.Errorf("scan sessions: %w", err)
		}
		count += len(result.Elements)
		cursor = result.Cursor
		if cursor == 0 {
			break
		}
	}
	return count, nil
}

// Ping Valkey 연결 확인
func (s *Store) Ping(ctx context.Context) error {
	if !s.enabled {
		return ErrStoreDisabled
	}
	if s.backend == storeBackendMemory {
		return nil
	}

	cmd := s.client.B().Ping().Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return fmt.Errorf("ping valkey: %w", err)
	}
	return nil
}
