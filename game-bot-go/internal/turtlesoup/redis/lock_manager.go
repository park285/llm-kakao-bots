package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	tsassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/assets"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
)

// LockManager: Redis와 Lua 스크립트를 사용하여 분산 락(Distributed Lock)을 구현한 관리자
// Reentrancy(재진입)를 지원하며, Context 스코프 기반의 자동 락 관리를 제공한다.
type LockManager struct {
	client valkey.Client
	logger *slog.Logger

	releaseSHA       string
	scriptMu         sync.Mutex
	redisCallTimeout time.Duration
}

// NewLockManager: 새로운 LockManager 인스턴스를 생성한다.
func NewLockManager(client valkey.Client, logger *slog.Logger) *LockManager {
	return &LockManager{
		client:           client,
		logger:           logger,
		redisCallTimeout: 5 * time.Second,
	}
}

func (m *LockManager) loadReleaseScript(ctx context.Context) (string, error) {
	m.scriptMu.Lock()
	defer m.scriptMu.Unlock()

	if m.releaseSHA != "" {
		return m.releaseSHA, nil
	}

	cmd := m.client.B().ScriptLoad().Script(tsassets.LockReleaseLua).Build()
	sha, err := m.client.Do(ctx, cmd).ToString()
	if err != nil {
		return "", fmt.Errorf("load release script: %w", err)
	}
	m.releaseSHA = sha
	return sha, nil
}

func (m *LockManager) clearScriptCache() {
	m.scriptMu.Lock()
	defer m.scriptMu.Unlock()
	m.releaseSHA = ""
}

// TryAcquireSharedLock: 여러 사용자가 동시에 접근 가능한 '공유 락(Shared Lock)' 획득을 시도한다. (주로 읽기 작업용)
// 이미 락이 존재하더라도 획득 성공으로 간주하거나, 별도의 키를 사용하여 동시성을 제어한다.
func (m *LockManager) TryAcquireSharedLock(ctx context.Context, lockKey string, ttlSeconds int64) (bool, error) {
	lockKey = strings.TrimSpace(lockKey)
	if lockKey == "" {
		return false, fmt.Errorf("lock key is empty")
	}
	if ttlSeconds <= 0 {
		return false, fmt.Errorf("invalid ttlSeconds: %d", ttlSeconds)
	}

	cmd := m.client.B().Set().Key(lockKey).Value("1").Nx().Ex(time.Duration(ttlSeconds) * time.Second).Build()
	err := m.client.Do(ctx, cmd).Error()
	if err != nil {
		if valkeyx.IsNil(err) {
			return false, nil
		}
		return false, cerrors.RedisError{Operation: "shared_lock_acquire", Err: err}
	}
	return true, nil
}

// ReleaseSharedLock: 공유 락을 해제한다. (DEL)
func (m *LockManager) ReleaseSharedLock(ctx context.Context, lockKey string) error {
	lockKey = strings.TrimSpace(lockKey)
	if lockKey == "" {
		return fmt.Errorf("lock key is empty")
	}
	cmd := m.client.B().Del().Key(lockKey).Build()
	if err := m.client.Do(ctx, cmd).Error(); err != nil {
		return cerrors.RedisError{Operation: "shared_lock_release", Err: err}
	}
	return nil
}

// WithLock: 배타적 락(Write Lock)을 획득한 상태에서 주어진 함수(block)를 실행한다.
// 락 획득 실패 시 에러를 반환하며, 실행 완료 후 자동으로 락을 해제한다. 재진입(Reentry)을 지원한다.
func (m *LockManager) WithLock(ctx context.Context, sessionID string, holderName *string, block func(ctx context.Context) error) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session id is empty")
	}

	scope, ok := lockScopeFromContext(ctx)
	if !ok {
		scope = newLockScope()
		ctx = withLockScope(ctx, scope)
	}

	key := lockKey(sessionID)
	if scope.incrementIfHeld(key) {
		defer scope.decrement(key)
		return block(ctx)
	}

	timeoutSeconds := int64(tsconfig.RedisLockTimeoutSeconds)
	lockTTLSeconds := int64(tsconfig.RedisLockTTLSeconds)

	acquireCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	defer cancel()

	token, err := newToken()
	if err != nil {
		return fmt.Errorf("generate lock token failed: %w", err)
	}

	holderValue := buildHolderValue(token, holderName)

	m.logger.Debug("lock_attempting", "session_id", sessionID, "timeout_seconds", timeoutSeconds)

	acquired, acquireErr := m.acquire(acquireCtx, sessionID, token, holderValue, time.Duration(lockTTLSeconds)*time.Second)
	if acquireErr != nil {
		return acquireErr
	}
	if !acquired {
		currentHolder, holderErr := m.getHolder(ctx, sessionID)
		if holderErr != nil {
			m.logger.Warn("lock_holder_read_failed", "err", holderErr, "session_id", sessionID)
		}
		return cerrors.LockError{
			SessionID:   sessionID,
			HolderName:  currentHolder,
			Description: "failed to acquire lock",
		}
	}

	scope.set(key, token)
	defer func() {
		releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), m.redisCallTimeout)
		defer releaseCancel()

		releaseToken, shouldRelease := scope.releaseIfLast(key)
		if !shouldRelease {
			return
		}
		if err := m.release(releaseCtx, sessionID, releaseToken); err != nil {
			m.logger.Warn("lock_release_failed", "err", err, "session_id", sessionID)
		} else {
			m.logger.Debug("lock_released", "session_id", sessionID)
		}
	}()

	m.logger.Debug("lock_acquired", "session_id", sessionID)
	return block(ctx)
}

// 락 획득 재시도 설정
const (
	lockRetryInitialDelay  = 50 * time.Millisecond
	lockRetryMaxDelay      = 500 * time.Millisecond
	lockRetryDelayMultiply = 2
)

// acquire: Redis의 SET NX 명령과 고유 토큰을 사용하여 락 획득을 시도한다.
// 실패 시 지수 백오프(Exponential Backoff) 전략으로 재시도한다.
func (m *LockManager) acquire(ctx context.Context, sessionID string, token string, holderValue string, ttl time.Duration) (bool, error) {
	key := lockKey(sessionID)
	holderKey := lockHolderKey(sessionID)

	delay := lockRetryInitialDelay

	for {
		cmd := m.client.B().Set().Key(key).Value(token).Nx().Ex(ttl).Build()
		err := m.client.Do(ctx, cmd).Error()
		if err != nil {
			if valkeyx.IsNil(err) {
				// SET NX failed (key exists)
				if ctx.Err() != nil {
					return false, nil
				}

				// Exponential backoff: 50ms → 100ms → 200ms → ... 最大 500ms
				timer := time.NewTimer(delay)
				select {
				case <-timer.C:
					delay = min(delay*lockRetryDelayMultiply, lockRetryMaxDelay)
					continue
				case <-ctx.Done():
					timer.Stop()
					return false, nil
				}
			}
			if ctx.Err() != nil {
				return false, nil
			}
			return false, cerrors.RedisError{Operation: "lock_acquire", Err: err}
		}

		// Lock acquired
		break
	}

	holderCmd := m.client.B().Set().Key(holderKey).Value(holderValue).Ex(ttl).Build()
	if err := m.client.Do(ctx, holderCmd).Error(); err != nil {
		_ = m.release(context.Background(), sessionID, token)
		return false, cerrors.RedisError{Operation: "lock_set_holder", Err: err}
	}

	return true, nil
}

func (m *LockManager) release(ctx context.Context, sessionID string, token string) error {
	sha, err := m.loadReleaseScript(ctx)
	if err != nil {
		return err
	}

	key := lockKey(sessionID)
	holderKey := lockHolderKey(sessionID)

	cmd := m.client.B().Evalsha().Sha1(sha).Numkeys(2).Key(key, holderKey).Arg(token).Build()
	if err := m.client.Do(ctx, cmd).Error(); err != nil {
		if valkeyx.IsNoScript(err) {
			m.clearScriptCache()
			return m.release(ctx, sessionID, token)
		}
		return cerrors.RedisError{Operation: "lock_release", Err: err}
	}
	return nil
}

func (m *LockManager) getHolder(ctx context.Context, sessionID string) (*string, error) {
	holderKey := lockHolderKey(sessionID)
	cmd := m.client.B().Get().Key(holderKey).Build()
	value, err := m.client.Do(ctx, cmd).ToString()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, cerrors.RedisError{Operation: "lock_get_holder", Err: err}
	}

	_, name := parseHolderValue(value)
	if strings.TrimSpace(name) == "" {
		return nil, nil
	}
	return &name, nil
}

func newToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("rand read failed: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func buildHolderValue(token string, holderName *string) string {
	name := "다른 사용자"
	if holderName != nil && strings.TrimSpace(*holderName) != "" {
		name = strings.TrimSpace(*holderName)
	}
	return token + "|" + name
}

func parseHolderValue(raw string) (token string, name string) {
	raw = strings.TrimSpace(raw)
	delim := strings.Index(raw, "|")
	if delim <= 0 {
		return "", raw
	}
	return raw[:delim], raw[delim+1:]
}

type lockScopeKey struct{}

type lockScope struct {
	mu   sync.Mutex
	held map[string]*heldLock
}

type heldLock struct {
	token string
	count int
}

func newLockScope() *lockScope {
	return &lockScope{held: make(map[string]*heldLock)}
}

func withLockScope(ctx context.Context, scope *lockScope) context.Context {
	return context.WithValue(ctx, lockScopeKey{}, scope)
}

func lockScopeFromContext(ctx context.Context) (*lockScope, bool) {
	scope, ok := ctx.Value(lockScopeKey{}).(*lockScope)
	return scope, ok && scope != nil
}

func (s *lockScope) incrementIfHeld(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.held[key]
	if !ok {
		return false
	}
	entry.count++
	return true
}

func (s *lockScope) set(key string, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.held[key] = &heldLock{token: token, count: 1}
}

func (s *lockScope) decrement(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.held[key]
	if !ok {
		return
	}
	entry.count--
	if entry.count <= 0 {
		delete(s.held, key)
	}
}

func (s *lockScope) releaseIfLast(key string) (token string, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.held[key]
	if !ok {
		return "", false
	}

	entry.count--
	if entry.count > 0 {
		return "", false
	}

	delete(s.held, key)
	return entry.token, true
}
