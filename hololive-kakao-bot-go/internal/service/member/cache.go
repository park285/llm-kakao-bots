package member

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sourcegraph/conc/pool"
	"go.uber.org/zap"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/internal/service/cache"
)

const (
	memberChannelKeyPrefix = "member:channel:"
	memberNameKeyPrefix    = "member:name:"
	memberAliasKeyPrefix   = "member:alias:"
	memberCachePattern     = "member:*"
	allChannelIDsKey       = "channel_ids"
)

// Cache 는 타입이다.
type Cache struct {
	repo   *Repository
	cache  *cache.Service
	logger *zap.Logger

	// In-memory caches
	byChannelID sync.Map // map[string]*domain.Member
	byName      sync.Map // map[string]*domain.Member
	allMembers  sync.Map // []string (channel IDs)

	// Cache configuration
	cacheTTL time.Duration
	warmup   bool

	// Warm-up configuration
	warmUpChunkSize     int
	warmUpMaxGoroutines int
}

// CacheConfig 는 타입이다.
type CacheConfig struct {
	ValkeyTTL           time.Duration
	WarmUp              bool // Load all members into memory on startup
	WarmUpChunkSize     int
	WarmUpMaxGoroutines int
}

// NewMemberCache 는 동작을 수행한다.
func NewMemberCache(ctx context.Context, repo *Repository, cacheService *cache.Service, logger *zap.Logger, cfg CacheConfig) (*Cache, error) {
	if cfg.ValkeyTTL == 0 {
		cfg.ValkeyTTL = constants.MemberCacheDefaults.ValkeyTTL
	}
	if cfg.WarmUpChunkSize == 0 {
		cfg.WarmUpChunkSize = constants.MemberCacheDefaults.WarmUpChunkSize
	}
	if cfg.WarmUpMaxGoroutines == 0 {
		cfg.WarmUpMaxGoroutines = constants.MemberCacheDefaults.WarmUpMaxGoroutines
	}

	mc := &Cache{
		repo:     repo,
		cache:    cacheService,
		logger:   logger,
		cacheTTL: cfg.ValkeyTTL,
		warmup:   cfg.WarmUp,

		warmUpChunkSize:     cfg.WarmUpChunkSize,
		warmUpMaxGoroutines: cfg.WarmUpMaxGoroutines,
	}

	// Warm up cache if enabled
	if cfg.WarmUp {
		if err := mc.WarmUpCache(ctx); err != nil {
			logger.Warn("Failed to warm up member cache", zap.Error(err))
		}
	}

	return mc, nil
}

func (c *Cache) cacheEnabled() bool {
	return c != nil && c.cache != nil
}

// WarmUpCache 는 동작을 수행한다.
func (c *Cache) WarmUpCache(ctx context.Context) error {
	members, err := c.repo.GetAllMembers(ctx)
	if err != nil {
		return fmt.Errorf("failed to load all members: %w", err)
	}

	// 병렬 처리를 위한 청크 분할
	chunkSize := c.warmUpChunkSize
	chunks := chunkMembers(members, chunkSize)

	// Worker pool로 병렬 캐싱
	p := pool.New().WithMaxGoroutines(c.warmUpMaxGoroutines)
	for _, chunk := range chunks {
		chunk := chunk
		p.Go(func() {
			c.cacheChunk(ctx, chunk)
		})
	}
	p.Wait()

	// 인메모리 캐시 업데이트
	for _, member := range members {
		if member.ChannelID != "" {
			c.byChannelID.Store(member.ChannelID, member)
		}
		c.byName.Store(member.Name, member)
	}

	c.logger.Info("Member cache warmed up",
		zap.Int("total_members", len(members)),
		zap.Int("chunks", len(chunks)),
	)

	return nil
}

// 청크 단위로 Valkey에 파이프라이닝 저장
func (c *Cache) cacheChunk(ctx context.Context, members []*domain.Member) {
	if len(members) == 0 {
		return
	}
	if !c.cacheEnabled() {
		return
	}

	// 배치 저장을 위한 맵 준비
	pairs := make(map[string]interface{}, len(members)*2)

	for _, member := range members {
		if member.ChannelID != "" {
			channelKey := memberChannelKeyPrefix + member.ChannelID
			pairs[channelKey] = member
		}

		nameKey := memberNameKeyPrefix + member.Name
		pairs[nameKey] = member
	}

	// MSet으로 배치 저장
	if err := c.cache.MSet(ctx, pairs, c.cacheTTL); err != nil {
		c.logger.Warn("Failed to batch cache members",
			zap.Int("count", len(members)),
			zap.Error(err))
	}
}

// GetByChannelID 는 동작을 수행한다.
func (c *Cache) GetByChannelID(ctx context.Context, channelID string) (*domain.Member, error) {
	// 인메모리 캐시 먼저 확인
	if val, ok := c.byChannelID.Load(channelID); ok {
		return val.(*domain.Member), nil
	}

	if c.cacheEnabled() {
		// Valkey 캐시 확인
		cacheKey := memberChannelKeyPrefix + channelID
		var member domain.Member
		if err := c.cache.Get(ctx, cacheKey, &member); err == nil && member.Name != "" {
			// 인메모리 캐시에 저장
			c.byChannelID.Store(channelID, &member)
			return &member, nil
		}
	}

	// DB 조회
	dbMember, err := c.repo.FindByChannelID(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if dbMember == nil {
		return nil, nil
	}

	// 캐시 저장
	c.cacheMember(ctx, dbMember)

	return dbMember, nil
}

// GetByName 는 동작을 수행한다.
func (c *Cache) GetByName(ctx context.Context, name string) (*domain.Member, error) {
	// 인메모리 캐시 먼저 확인
	if val, ok := c.byName.Load(name); ok {
		return val.(*domain.Member), nil
	}

	if c.cacheEnabled() {
		// Valkey 캐시 확인
		cacheKey := memberNameKeyPrefix + name
		var member domain.Member
		if err := c.cache.Get(ctx, cacheKey, &member); err == nil && member.Name != "" {
			c.byName.Store(name, &member)
			return &member, nil
		}
	}

	// DB 조회
	dbMember, err := c.repo.FindByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if dbMember == nil {
		return nil, nil
	}

	c.cacheMember(ctx, dbMember)
	return dbMember, nil
}

// FindByAlias 는 동작을 수행한다.
func (c *Cache) FindByAlias(ctx context.Context, alias string) (*domain.Member, error) {
	if c.cacheEnabled() {
		// Valkey 캐시 확인
		cacheKey := memberAliasKeyPrefix + alias
		var member domain.Member
		if err := c.cache.Get(ctx, cacheKey, &member); err == nil && member.Name != "" {
			// 인메모리 캐시에도 저장
			if member.ChannelID != "" {
				c.byChannelID.Store(member.ChannelID, &member)
			}
			c.byName.Store(member.Name, &member)
			return &member, nil
		}
	}

	// DB 조회
	dbMember, err := c.repo.FindByAlias(ctx, alias)
	if err != nil {
		return nil, err
	}
	if dbMember == nil {
		return nil, nil
	}

	// 캐시 저장
	c.cacheMember(ctx, dbMember)

	if c.cacheEnabled() {
		// Alias 키도 캐싱
		cacheKey := memberAliasKeyPrefix + alias
		_ = c.cache.Set(ctx, cacheKey, dbMember, c.cacheTTL)
	}

	return dbMember, nil
}

// GetAllChannelIDs 는 동작을 수행한다.
func (c *Cache) GetAllChannelIDs(ctx context.Context) ([]string, error) {
	if val, ok := c.allMembers.Load(allChannelIDsKey); ok {
		return val.([]string), nil
	}

	channelIDs, err := c.repo.GetAllChannelIDs(ctx)
	if err != nil {
		return nil, err
	}

	c.allMembers.Store(allChannelIDsKey, channelIDs)

	return channelIDs, nil
}

func (c *Cache) cacheMember(ctx context.Context, member *domain.Member) {
	if member.ChannelID != "" {
		c.byChannelID.Store(member.ChannelID, member)
	}
	c.byName.Store(member.Name, member)

	if !c.cacheEnabled() {
		return
	}

	// Valkey에도 저장
	if member.ChannelID != "" {
		channelKey := memberChannelKeyPrefix + member.ChannelID
		if err := c.cache.Set(ctx, channelKey, member, c.cacheTTL); err != nil {
			c.logger.Warn("Failed to cache member by channel ID",
				zap.String("channel_id", member.ChannelID),
				zap.Error(err),
			)
		}
	}

	nameKey := memberNameKeyPrefix + member.Name
	if err := c.cache.Set(ctx, nameKey, member, c.cacheTTL); err != nil {
		c.logger.Warn("Failed to cache member by name",
			zap.String("member", member.Name),
			zap.Error(err),
		)
	}
}

// InvalidateAll 는 동작을 수행한다.
func (c *Cache) InvalidateAll(ctx context.Context) error {
	// 인메모리 캐시 클리어
	c.byChannelID = sync.Map{}
	c.byName = sync.Map{}
	c.allMembers = sync.Map{}

	if !c.cacheEnabled() {
		c.logger.Info("Member cache invalidated", zap.Int("keys_deleted", 0))
		return nil
	}

	// Valkey 캐시 클리어
	keys, err := c.cache.Keys(ctx, memberCachePattern)
	if err != nil {
		return fmt.Errorf("failed to get keys for invalidation: %w", err)
	}
	if len(keys) > 0 {
		if _, err := c.cache.DelMany(ctx, keys); err != nil {
			return fmt.Errorf("failed to invalidate cache store: %w", err)
		}
	}

	c.logger.Info("Member cache invalidated", zap.Int("keys_deleted", len(keys)))
	return nil
}

// Refresh 는 동작을 수행한다.
func (c *Cache) Refresh(ctx context.Context) error {
	if err := c.InvalidateAll(ctx); err != nil {
		return err
	}
	return c.WarmUpCache(ctx)
}

// InvalidateAliasCache invalidates Valkey cache for specific alias
func (c *Cache) InvalidateAliasCache(ctx context.Context, alias string) error {
	if !c.cacheEnabled() {
		c.logger.Info("Alias cache invalidated", zap.String("alias", alias))
		return nil
	}

	aliasKey := memberAliasKeyPrefix + alias
	if err := c.cache.Del(ctx, aliasKey); err != nil {
		c.logger.Warn("Failed to invalidate alias cache",
			zap.String("alias", alias),
			zap.Error(err),
		)
		return fmt.Errorf("failed to invalidate alias cache: %w", err)
	}

	c.logger.Info("Alias cache invalidated", zap.String("alias", alias))
	return nil
}

// 청크 분할 헬퍼
func chunkMembers(members []*domain.Member, chunkSize int) [][]*domain.Member {
	var chunks [][]*domain.Member
	for i := 0; i < len(members); i += chunkSize {
		end := i + chunkSize
		if end > len(members) {
			end = len(members)
		}
		chunks = append(chunks, members[i:end])
	}
	return chunks
}
