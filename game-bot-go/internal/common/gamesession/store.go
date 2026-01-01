package gamesession

import (
	"context"
	"log/slog"
	"time"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
)

// KeyFunc: 세션 ID로 Redis 키를 생성하는 함수 타입입니다.
type KeyFunc func(sessionID string) string

// Store: 게임 세션 상태를 Redis에 JSON으로 직렬화하여 저장하고 관리하는 제네릭 저장소입니다.
// 게임별로 다른 키 프리픽스/TTL/데이터 타입만 주입하여 동일한 저장 로직을 재사용합니다.
type Store[T any] struct {
	client  valkey.Client
	logger  *slog.Logger
	keyFunc KeyFunc
	ttl     time.Duration
}

// Config: 세션 저장소 생성에 필요한 설정 정보입니다.
type Config struct {
	KeyFunc KeyFunc
	TTL     time.Duration
}

// NewStore: 새로운 제네릭 세션 저장소 인스턴스를 생성합니다.
func NewStore[T any](client valkey.Client, logger *slog.Logger, cfg Config) *Store[T] {
	if logger == nil {
		logger = slog.Default()
	}
	return &Store[T]{
		client:  client,
		logger:  logger,
		keyFunc: cfg.KeyFunc,
		ttl:     cfg.TTL,
	}
}

// Save: 세션 데이터를 JSON으로 직렬화하여 Redis에 저장합니다. (TTL 설정됨)
func (s *Store[T]) Save(ctx context.Context, sessionID string, data T) error {
	key := s.keyFunc(sessionID)

	payload, err := json.Marshal(data)
	if err != nil {
		return cerrors.RedisError{Operation: "session_marshal", Err: err}
	}

	if err := valkeyx.SetStringEX(ctx, s.client, key, string(payload), s.ttl); err != nil {
		return cerrors.RedisError{Operation: "session_save", Err: err}
	}

	s.logger.Debug("session_saved", "session_id", sessionID)
	return nil
}

// Load: Redis에 저장된 JSON 데이터를 조회하여 역직렬화합니다.
// 데이터가 없거나 만료된 경우 nil을 반환합니다.
func (s *Store[T]) Load(ctx context.Context, sessionID string) (*T, error) {
	key := s.keyFunc(sessionID)

	raw, ok, err := valkeyx.GetBytes(ctx, s.client, key)
	if err != nil {
		return nil, cerrors.RedisError{Operation: "session_load", Err: err}
	}
	if !ok {
		return nil, nil
	}

	var data T
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, cerrors.RedisError{Operation: "session_unmarshal", Err: err}
	}
	return &data, nil
}

// Delete: 세션 데이터를 Redis에서 삭제합니다. (게임 종료 시)
func (s *Store[T]) Delete(ctx context.Context, sessionID string) error {
	key := s.keyFunc(sessionID)

	if err := valkeyx.DeleteKeys(ctx, s.client, key); err != nil {
		return cerrors.RedisError{Operation: "session_delete", Err: err}
	}
	s.logger.Debug("session_deleted", "session_id", sessionID)
	return nil
}

// Exists: 세션이 존재하는지 확인합니다.
func (s *Store[T]) Exists(ctx context.Context, sessionID string) (bool, error) {
	key := s.keyFunc(sessionID)

	cmd := s.client.B().Exists().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, cerrors.RedisError{Operation: "session_exists", Err: err}
	}
	return n > 0, nil
}

// RefreshTTL: 세션의 TTL을 연장합니다.
func (s *Store[T]) RefreshTTL(ctx context.Context, sessionID string) (bool, error) {
	key := s.keyFunc(sessionID)

	ttlSeconds := int64(s.ttl.Seconds())
	cmd := s.client.B().Expire().Key(key).Seconds(ttlSeconds).Build()
	ok, err := s.client.Do(ctx, cmd).AsBool()
	if err != nil {
		return false, cerrors.RedisError{Operation: "session_refresh_ttl", Err: err}
	}
	return ok, nil
}

// Client: 내부 Valkey 클라이언트를 반환합니다.
// 게임별 확장 기능 구현 시 사용됩니다.
func (s *Store[T]) Client() valkey.Client {
	return s.client
}

// Logger: 내부 로거를 반환합니다.
func (s *Store[T]) Logger() *slog.Logger {
	return s.logger
}

// TTL: 설정된 TTL을 반환합니다.
func (s *Store[T]) TTL() time.Duration {
	return s.ttl
}
