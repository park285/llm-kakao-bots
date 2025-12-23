package session

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
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

type storeConnInfo struct {
	addr     string
	username string
	password string
	selectDB int
	useTLS   bool
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

func (s *Store) createSessionMemory(meta Meta) error {
	now := time.Now()
	expiresAt := s.computeExpiry(now)

	s.mu.Lock()
	s.pruneExpiredLocked(now)
	s.meta[meta.ID] = meta
	if !expiresAt.IsZero() {
		s.metaExpiresAt[meta.ID] = expiresAt
	} else {
		delete(s.metaExpiresAt, meta.ID)
	}
	s.mu.Unlock()
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

	var meta Meta
	if err := json.Unmarshal([]byte(result), &meta); err != nil {
		return nil, fmt.Errorf("unmarshal session meta: %w", err)
	}

	return &meta, nil
}

func (s *Store) getSessionMemory(sessionID string) (*Meta, error) {
	if strings.TrimSpace(sessionID) == "" {
		return nil, ErrSessionNotFound
	}

	now := time.Now()
	s.mu.Lock()
	s.pruneExpiredLocked(now)
	expiresAt, ok := s.metaExpiresAt[sessionID]
	if ok && !expiresAt.IsZero() && now.After(expiresAt) {
		delete(s.metaExpiresAt, sessionID)
		delete(s.meta, sessionID)
		s.mu.Unlock()
		return nil, ErrSessionNotFound
	}

	meta, ok := s.meta[sessionID]
	if !ok {
		s.mu.Unlock()
		return nil, ErrSessionNotFound
	}
	s.mu.Unlock()

	copied := meta
	return &copied, nil
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

func (s *Store) updateSessionMemory(meta Meta) error {
	now := time.Now()
	meta.UpdatedAt = now
	expiresAt := s.computeExpiry(now)

	s.mu.Lock()
	s.pruneExpiredLocked(now)
	s.meta[meta.ID] = meta
	if !expiresAt.IsZero() {
		s.metaExpiresAt[meta.ID] = expiresAt
	} else {
		delete(s.metaExpiresAt, meta.ID)
	}
	s.mu.Unlock()
	return nil
}

// DeleteSession 세션 삭제
func (s *Store) DeleteSession(ctx context.Context, sessionID string) error {
	if !s.enabled {
		return ErrStoreDisabled
	}
	if s.backend == storeBackendMemory {
		return s.deleteSessionMemory(sessionID)
	}

	metaCmd := s.client.B().Del().Key(s.metaKey(sessionID)).Build()
	historyCmd := s.client.B().Del().Key(s.historyKey(sessionID)).Build()

	if err := s.client.Do(ctx, metaCmd).Error(); err != nil {
		return fmt.Errorf("delete session meta: %w", err)
	}
	if err := s.client.Do(ctx, historyCmd).Error(); err != nil {
		return fmt.Errorf("delete session history: %w", err)
	}

	return nil
}

func (s *Store) deleteSessionMemory(sessionID string) error {
	now := time.Now()
	s.mu.Lock()
	s.pruneExpiredLocked(now)
	delete(s.meta, sessionID)
	delete(s.history, sessionID)
	delete(s.metaExpiresAt, sessionID)
	delete(s.historyExpireAt, sessionID)
	s.mu.Unlock()
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

func (s *Store) getHistoryMemory(sessionID string) []llm.HistoryEntry {
	now := time.Now()
	s.mu.Lock()
	s.pruneExpiredLocked(now)
	expiresAt, ok := s.historyExpireAt[sessionID]
	if ok && !expiresAt.IsZero() && now.After(expiresAt) {
		delete(s.historyExpireAt, sessionID)
		delete(s.history, sessionID)
		s.mu.Unlock()
		return nil
	}

	history := s.history[sessionID]
	if len(history) == 0 {
		s.mu.Unlock()
		return nil
	}
	copied := append([]llm.HistoryEntry(nil), history...)
	s.mu.Unlock()
	return copied
}

// AppendHistory 히스토리에 메시지 추가
func (s *Store) AppendHistory(ctx context.Context, sessionID string, entries ...llm.HistoryEntry) error {
	if !s.enabled {
		return ErrStoreDisabled
	}
	if s.backend == storeBackendMemory {
		return s.appendHistoryMemory(sessionID, entries...)
	}

	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("marshal history entry: %w", err)
		}

		cmd := s.client.B().Rpush().Key(s.historyKey(sessionID)).Element(string(data)).Build()
		if err := s.client.Do(ctx, cmd).Error(); err != nil {
			return fmt.Errorf("append history: %w", err)
		}
	}

	// TTL 갱신
	expireCmd := s.client.B().Expire().Key(s.historyKey(sessionID)).Seconds(int64(s.ttl().Seconds())).Build()
	_ = s.client.Do(ctx, expireCmd)

	// 히스토리 크기 제한
	maxPairs := s.cfg.Session.HistoryMaxPairs
	if maxPairs > 0 {
		trimCmd := s.client.B().Ltrim().Key(s.historyKey(sessionID)).Start(int64(-maxPairs * 2)).Stop(-1).Build()
		_ = s.client.Do(ctx, trimCmd)
	}

	return nil
}

func (s *Store) appendHistoryMemory(sessionID string, entries ...llm.HistoryEntry) error {
	now := time.Now()
	expiresAt := s.computeExpiry(now)

	s.mu.Lock()
	s.pruneExpiredLocked(now)
	existing := s.history[sessionID]
	existing = append(existing, entries...)

	maxPairs := 0
	if s.cfg != nil {
		maxPairs = s.cfg.Session.HistoryMaxPairs
	}
	if maxPairs > 0 {
		maxEntries := maxPairs * 2
		if len(existing) > maxEntries {
			existing = existing[len(existing)-maxEntries:]
		}
	}

	s.history[sessionID] = existing
	if !expiresAt.IsZero() {
		s.historyExpireAt[sessionID] = expiresAt
	} else {
		delete(s.historyExpireAt, sessionID)
	}
	s.mu.Unlock()
	return nil
}

// SessionCount 현재 세션 수 (근사치)
func (s *Store) SessionCount(ctx context.Context) (int, error) {
	if !s.enabled {
		return 0, nil
	}
	if s.backend == storeBackendMemory {
		return s.sessionCountMemory(), nil
	}

	cmd := s.client.B().Keys().Pattern("session:*:meta").Build()
	results, err := s.client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		return 0, fmt.Errorf("count sessions: %w", err)
	}

	return len(results), nil
}

func (s *Store) sessionCountMemory() int {
	now := time.Now()
	s.mu.Lock()
	s.pruneExpiredLocked(now)
	count := len(s.meta)
	s.mu.Unlock()
	return count
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

func (s *Store) computeExpiry(now time.Time) time.Time {
	ttl := time.Duration(0)
	if s != nil {
		ttl = s.ttl()
	}
	if ttl <= 0 {
		return time.Time{}
	}
	return now.Add(ttl)
}

func (s *Store) pruneExpiredLocked(now time.Time) {
	for sessionID, expiresAt := range s.metaExpiresAt {
		if expiresAt.IsZero() || now.Before(expiresAt) {
			continue
		}
		delete(s.metaExpiresAt, sessionID)
		delete(s.meta, sessionID)
	}

	for sessionID, expiresAt := range s.historyExpireAt {
		if expiresAt.IsZero() || now.Before(expiresAt) {
			continue
		}
		delete(s.historyExpireAt, sessionID)
		delete(s.history, sessionID)
	}
}

func parseStoreURL(raw string) (storeConnInfo, error) {
	if strings.TrimSpace(raw) == "" {
		return storeConnInfo{}, errors.New("session store url is empty")
	}

	if !strings.Contains(raw, "://") {
		return parseStoreAddr(raw)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return storeConnInfo{}, fmt.Errorf("parse url: %w", err)
	}

	host := parsed.Hostname()
	if host == "" {
		return storeConnInfo{}, errors.New("session store host missing")
	}

	port := parsed.Port()
	if port == "" {
		port = "6379"
	}

	selectDB := 0
	if strings.TrimSpace(parsed.Path) != "" && parsed.Path != "/" {
		path := strings.TrimPrefix(parsed.Path, "/")
		db, err := strconv.Atoi(path)
		if err != nil {
			return storeConnInfo{}, fmt.Errorf("invalid session store db: %w", err)
		}
		if db < 0 {
			return storeConnInfo{}, errors.New("invalid session store db")
		}
		selectDB = db
	}

	username := ""
	password := ""
	if parsed.User != nil {
		username = parsed.User.Username()
		pw, _ := parsed.User.Password()
		password = pw
	}

	useTLS := strings.EqualFold(parsed.Scheme, "rediss")

	return storeConnInfo{
		addr:     net.JoinHostPort(host, port),
		username: username,
		password: password,
		selectDB: selectDB,
		useTLS:   useTLS,
	}, nil
}

func parseStoreAddr(addr string) (storeConnInfo, error) {
	trimmed := strings.TrimSpace(addr)
	if trimmed == "" {
		return storeConnInfo{}, errors.New("session store address is empty")
	}

	host, port, err := net.SplitHostPort(trimmed)
	if err != nil {
		var addrErr *net.AddrError
		if !errors.As(err, &addrErr) {
			return storeConnInfo{}, fmt.Errorf("invalid session store address: %w", err)
		}
		switch addrErr.Err {
		case "missing port in address":
			host = strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
			port = "6379"
		case "too many colons in address":
			host = trimmed
			port = "6379"
		default:
			return storeConnInfo{}, fmt.Errorf("invalid session store address: %w", err)
		}
	}

	if strings.TrimSpace(host) == "" {
		return storeConnInfo{}, errors.New("session store host missing")
	}

	return storeConnInfo{
		addr:     net.JoinHostPort(host, port),
		selectDB: 0,
		useTLS:   false,
	}, nil
}
