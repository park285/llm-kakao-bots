package redis

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qassets "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/assets"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

type lockMode int

const (
	lockModeWrite lockMode = iota
	lockModeRead
)

func (m lockMode) String() string {
	switch m {
	case lockModeWrite:
		return "WRITE"
	case lockModeRead:
		return "READ"
	default:
		return "UNKNOWN"
	}
}

// LockManager: Redis Lua 스크립트를 활용하여 읽기/쓰기 분산 락을 관리하는 컴포넌트
type LockManager struct {
	client valkey.Client
	logger *slog.Logger

	acquireReadSHA  string
	acquireWriteSHA string
	releaseReadSHA  string
	releaseWriteSHA string
	renewReadSHA    string
	renewWriteSHA   string

	scriptMu         sync.Mutex
	redisCallTimeout time.Duration
}

// NewLockManager: 새로운 LockManager 인스턴스를 생성하고 Redis 클라이언트를 설정한다.
func NewLockManager(client valkey.Client, logger *slog.Logger) *LockManager {
	return &LockManager{
		client:           client,
		logger:           logger,
		redisCallTimeout: 5 * time.Second,
	}
}

func (m *LockManager) loadScripts(ctx context.Context) error {
	m.scriptMu.Lock()
	defer m.scriptMu.Unlock()

	if m.acquireReadSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockAcquireReadLua)
		if err != nil {
			return fmt.Errorf("load acquire_read script: %w", err)
		}
		m.acquireReadSHA = sha
	}
	if m.acquireWriteSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockAcquireWriteLua)
		if err != nil {
			return fmt.Errorf("load acquire_write script: %w", err)
		}
		m.acquireWriteSHA = sha
	}
	if m.releaseReadSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockReleaseReadLua)
		if err != nil {
			return fmt.Errorf("load release_read script: %w", err)
		}
		m.releaseReadSHA = sha
	}
	if m.releaseWriteSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockReleaseLua)
		if err != nil {
			return fmt.Errorf("load release_write script: %w", err)
		}
		m.releaseWriteSHA = sha
	}
	if m.renewReadSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockRenewReadLua)
		if err != nil {
			return fmt.Errorf("load renew_read script: %w", err)
		}
		m.renewReadSHA = sha
	}
	if m.renewWriteSHA == "" {
		sha, err := m.loadScript(ctx, qassets.LockRenewWriteLua)
		if err != nil {
			return fmt.Errorf("load renew_write script: %w", err)
		}
		m.renewWriteSHA = sha
	}
	return nil
}

func (m *LockManager) loadScript(ctx context.Context, script string) (string, error) {
	cmd := m.client.B().ScriptLoad().Script(script).Build()
	sha, err := m.client.Do(ctx, cmd).ToString()
	if err != nil {
		return "", fmt.Errorf("script load failed: %w", err)
	}
	return sha, nil
}

func (m *LockManager) clearScriptCache() {
	m.scriptMu.Lock()
	defer m.scriptMu.Unlock()
	m.acquireReadSHA = ""
	m.acquireWriteSHA = ""
	m.releaseReadSHA = ""
	m.releaseWriteSHA = ""
	m.renewReadSHA = ""
	m.renewWriteSHA = ""
}

// TryAcquireSharedLock: 공유 락(읽기 락 성격) 획득을 시도한다. (1회 시도, SET NX 사용)
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

// ReleaseSharedLock: 공유 락을 해제한다. (DEL 사용)
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

// WithLock: 배타적 락(Write Lock)을 획득한 상태에서 작업을 수행한다. (재진입 가능, Context Scope)
func (m *LockManager) WithLock(ctx context.Context, chatID string, holderName *string, block func(ctx context.Context) error) error {
	return m.withRWLock(ctx, chatID, holderName, lockModeWrite, block)
}

// WithReadLock: 공유 락(Read Lock)을 획득한 상태에서 작업을 수행한다. (재진입 가능, Context Scope)
func (m *LockManager) WithReadLock(ctx context.Context, chatID string, holderName *string, block func(ctx context.Context) error) error {
	return m.withRWLock(ctx, chatID, holderName, lockModeRead, block)
}

func (m *LockManager) withRWLock(ctx context.Context, chatID string, holderName *string, mode lockMode, block func(ctx context.Context) error) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return fmt.Errorf("chat id is empty")
	}

	scope, ok := lockScopeFromContext(ctx)
	if !ok {
		scope = newLockScope()
		ctx = withLockScope(ctx, scope)
	}

	handled, err := m.handleReentry(ctx, scope, chatID, holderName, mode, block)
	if handled {
		return err
	}

	token, err := newToken()
	if err != nil {
		return fmt.Errorf("generate lock token failed: %w", err)
	}

	ttlMillis := int64(qconfig.RedisLockTTLSeconds) * 1000

	// 락 획득 재시도 (exponential backoff)
	acquired, acquireErr := m.acquireWithRetry(ctx, chatID, mode, token, ttlMillis)
	if acquireErr != nil {
		return acquireErr
	}
	if !acquired {
		return cerrors.LockError{
			SessionID:   chatID,
			HolderName:  holderName,
			Description: "failed to acquire lock after retries",
		}
	}

	key, err := scopeKey(chatID, mode)
	if err != nil {
		return err
	}

	renewCancel := m.startRenewWatchdog(ctx, chatID, mode, token, ttlMillis)
	defer m.releaseIfLast(ctx, scope, key, chatID, ttlMillis)

	scope.set(key, heldLock{
		mode:      mode,
		token:     token,
		count:     1,
		stopRenew: renewCancel,
	})

	m.logger.Debug("lock_acquired", "chat_id", chatID, "mode", mode.String())
	return block(ctx)
}

func (m *LockManager) handleReentry(
	ctx context.Context,
	scope *lockScope,
	chatID string,
	holderName *string,
	mode lockMode,
	block func(ctx context.Context) error,
) (handled bool, err error) {
	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)

	switch mode {
	case lockModeWrite:
		if scope.incrementIfHeld(writeKey) {
			defer scope.decrement(writeKey)
			return true, block(ctx)
		}
		if scope.isHeld(readKey) {
			return true, cerrors.LockError{
				SessionID:   chatID,
				HolderName:  holderName,
				Description: "write lock requested while read lock held",
			}
		}
		return false, nil
	case lockModeRead:
		if scope.incrementIfHeld(writeKey) {
			defer scope.decrement(writeKey)
			return true, block(ctx)
		}
		if scope.incrementIfHeld(readKey) {
			defer scope.decrement(readKey)
			return true, block(ctx)
		}
		return false, nil
	default:
		return true, fmt.Errorf("unknown lock mode: %d", mode)
	}
}

func (m *LockManager) releaseIfLast(ctx context.Context, scope *lockScope, key string, chatID string, ttlMillis int64) {
	held, shouldRelease := scope.releaseIfLast(key)
	if !shouldRelease {
		return
	}

	if held.stopRenew != nil {
		held.stopRenew()
	}

	releaseCtx, releaseCancel := context.WithTimeout(context.WithoutCancel(ctx), m.redisCallTimeout)
	defer releaseCancel()

	if err := m.release(releaseCtx, chatID, held.mode, held.token, ttlMillis); err != nil {
		m.logger.Warn("lock_release_failed", "err", err, "chat_id", chatID)
		return
	}
	m.logger.Debug("lock_released", "chat_id", chatID)
}

func scopeKey(chatID string, mode lockMode) (string, error) {
	switch mode {
	case lockModeWrite:
		return lockKey(chatID), nil
	case lockModeRead:
		return readLockKey(chatID), nil
	default:
		return "", fmt.Errorf("unknown lock mode: %d", mode)
	}
}

func (m *LockManager) acquire(ctx context.Context, chatID string, mode lockMode, token string, ttlMillis int64) (bool, error) {
	if err := m.loadScripts(ctx); err != nil {
		return false, err
	}

	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	switch mode {
	case lockModeWrite:
		cmd := m.client.B().Evalsha().Sha1(m.acquireWriteSHA).Numkeys(2).Key(writeKey, readKey).Arg(token, ttlArg).Build()
		n, err := m.client.Do(ctx, cmd).AsInt64()
		if err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.acquire(ctx, chatID, mode, token, ttlMillis)
			}
			return false, cerrors.RedisError{Operation: "lock_acquire_write", Err: err}
		}
		return n == 1, nil
	case lockModeRead:
		cmd := m.client.B().Evalsha().Sha1(m.acquireReadSHA).Numkeys(2).Key(writeKey, readKey).Arg(token, ttlArg).Build()
		n, err := m.client.Do(ctx, cmd).AsInt64()
		if err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.acquire(ctx, chatID, mode, token, ttlMillis)
			}
			return false, cerrors.RedisError{Operation: "lock_acquire_read", Err: err}
		}
		return n == 1, nil
	default:
		return false, fmt.Errorf("unknown lock mode: %d", mode)
	}
}

// 락 획득 재시도 설정
const (
	lockRetryMaxAttempts   = 3
	lockRetryInitialDelay  = 50 * time.Millisecond
	lockRetryMaxDelay      = 500 * time.Millisecond
	lockRetryDelayMultiply = 2
)

// acquireWithRetry 락 획득을 exponential backoff로 재시도.
// 경합 상황에서 즉시 실패 대신 짧은 간격으로 재시도하여 성공률 향상.
func (m *LockManager) acquireWithRetry(
	ctx context.Context,
	chatID string,
	mode lockMode,
	token string,
	ttlMillis int64,
) (bool, error) {
	delay := lockRetryInitialDelay

	for attempt := 0; attempt < lockRetryMaxAttempts; attempt++ {
		acquired, err := m.acquire(ctx, chatID, mode, token, ttlMillis)
		if err != nil {
			return false, err
		}
		if acquired {
			if attempt > 0 {
				m.logger.Debug("lock_acquired_after_retry", "chat_id", chatID, "mode", mode.String(), "attempt", attempt+1)
			}
			return true, nil
		}

		// 마지막 시도면 재시도 없이 종료
		if attempt == lockRetryMaxAttempts-1 {
			break
		}

		// Context 취소 확인 후 대기
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("lock acquire canceled: %w", ctx.Err())
		case <-time.After(delay):
			// 다음 재시도를 위해 delay 증가 (exponential backoff)
			delay = min(delay*lockRetryDelayMultiply, lockRetryMaxDelay)
		}
	}

	m.logger.Debug("lock_acquire_failed_after_retries", "chat_id", chatID, "mode", mode.String(), "attempts", lockRetryMaxAttempts)
	return false, nil
}

func (m *LockManager) renew(ctx context.Context, chatID string, mode lockMode, token string, ttlMillis int64) (bool, error) {
	if err := m.loadScripts(ctx); err != nil {
		return false, err
	}

	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	switch mode {
	case lockModeWrite:
		cmd := m.client.B().Evalsha().Sha1(m.renewWriteSHA).Numkeys(1).Key(writeKey).Arg(token, ttlArg).Build()
		n, err := m.client.Do(ctx, cmd).AsInt64()
		if err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.renew(ctx, chatID, mode, token, ttlMillis)
			}
			return false, cerrors.RedisError{Operation: "lock_renew_write", Err: err}
		}
		return n == 1, nil
	case lockModeRead:
		cmd := m.client.B().Evalsha().Sha1(m.renewReadSHA).Numkeys(1).Key(readKey).Arg(token, ttlArg).Build()
		n, err := m.client.Do(ctx, cmd).AsInt64()
		if err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.renew(ctx, chatID, mode, token, ttlMillis)
			}
			return false, cerrors.RedisError{Operation: "lock_renew_read", Err: err}
		}
		return n == 1, nil
	default:
		return false, fmt.Errorf("unknown lock mode: %d", mode)
	}
}

func (m *LockManager) release(ctx context.Context, chatID string, mode lockMode, token string, ttlMillis int64) error {
	if err := m.loadScripts(ctx); err != nil {
		return err
	}

	writeKey := lockKey(chatID)
	readKey := readLockKey(chatID)
	ttlArg := strconv.FormatInt(ttlMillis, 10)

	switch mode {
	case lockModeWrite:
		cmd := m.client.B().Evalsha().Sha1(m.releaseWriteSHA).Numkeys(1).Key(writeKey).Arg(token).Build()
		if err := m.client.Do(ctx, cmd).Error(); err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.release(ctx, chatID, mode, token, ttlMillis)
			}
			return cerrors.RedisError{Operation: "lock_release_write", Err: err}
		}
		return nil
	case lockModeRead:
		cmd := m.client.B().Evalsha().Sha1(m.releaseReadSHA).Numkeys(1).Key(readKey).Arg(token, ttlArg).Build()
		if err := m.client.Do(ctx, cmd).Error(); err != nil {
			if valkeyx.IsNoScript(err) {
				m.clearScriptCache()
				return m.release(ctx, chatID, mode, token, ttlMillis)
			}
			return cerrors.RedisError{Operation: "lock_release_read", Err: err}
		}
		return nil
	default:
		return fmt.Errorf("unknown lock mode: %d", mode)
	}
}

func (m *LockManager) startRenewWatchdog(ctx context.Context, chatID string, mode lockMode, token string, ttlMillis int64) context.CancelFunc {
	const (
		minIntervalSeconds = 1
		divisor            = 3
	)

	intervalMillis := ttlMillis / divisor
	interval := time.Duration(intervalMillis) * time.Millisecond
	if interval < time.Duration(minIntervalSeconds)*time.Second {
		interval = time.Duration(minIntervalSeconds) * time.Second
	}

	renewCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-renewCtx.Done():
				return
			case <-ticker.C:
				callCtx, callCancel := context.WithTimeout(context.WithoutCancel(ctx), m.redisCallTimeout)
				renewed, err := m.renew(callCtx, chatID, mode, token, ttlMillis)
				callCancel()
				if err != nil {
					m.logger.Warn("lock_renew_failed", "chat_id", chatID, "mode", mode.String(), "err", err)
					return
				}
				if !renewed {
					m.logger.Warn("lock_renew_rejected", "chat_id", chatID, "mode", mode.String())
					return
				}
			}
		}
	}()

	return cancel
}

func newToken() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("rand read failed: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

type lockScopeKey struct{}

type lockScope struct {
	mu   sync.Mutex
	held map[string]*heldLock
}

type heldLock struct {
	mode      lockMode
	token     string
	count     int
	stopRenew context.CancelFunc
}

func newLockScope() *lockScope {
	return &lockScope{held: make(map[string]*heldLock)}
}

func withLockScope(ctx context.Context, scope *lockScope) context.Context {
	return context.WithValue(ctx, lockScopeKey{}, scope)
}

func lockScopeFromContext(ctx context.Context) (*lockScope, bool) {
	scope, ok := ctx.Value(lockScopeKey{}).(*lockScope)
	return scope, ok
}

func (s *lockScope) incrementIfHeld(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.held[key]
	if !ok {
		return false
	}

	lock.count++
	return true
}

func (s *lockScope) decrement(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.held[key]
	if !ok {
		return
	}

	lock.count--
	if lock.count <= 0 {
		delete(s.held, key)
	}
}

func (s *lockScope) set(key string, lock heldLock) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.held[key] = &lock
}

func (s *lockScope) isHeld(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.held[key]
	return ok
}

func (s *lockScope) releaseIfLast(key string) (held heldLock, shouldRelease bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	lock, ok := s.held[key]
	if !ok {
		return heldLock{}, false
	}

	lock.count--
	if lock.count > 0 {
		return heldLock{}, false
	}

	delete(s.held, key)
	return *lock, true
}
