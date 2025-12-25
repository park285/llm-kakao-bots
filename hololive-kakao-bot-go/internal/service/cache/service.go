package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/valkey-io/valkey-go"

	"log/slog"

	"github.com/kapu/hololive-kakao-bot-go/internal/constants"
	"github.com/kapu/hololive-kakao-bot-go/internal/domain"
	"github.com/kapu/hololive-kakao-bot-go/pkg/errors"
)

// Service: Valkey(Redis) 클라이언트를 래핑하여 캐싱 기능을 제공하는 서비스
// 기본 Key-Value 외에도 Set, Hash 등 다양한 자료구조 연산을 지원한다.
type Service struct {
	client    valkey.Client
	logger    *slog.Logger
	closeOnce sync.Once
}

const memberHashKey = "hololive:members"

// Config: Valkey 연결 설정을 담는 구조체
type Config struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// NewCacheService: 새로운 Valkey 캐시 서비스 인스턴스를 생성하고 연결을 수립한다.
func NewCacheService(cfg Config, logger *slog.Logger) (*Service, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress:       []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Password:          cfg.Password,
		SelectDB:          cfg.DB,
		ConnWriteTimeout:  constants.MQConfig.ConnWriteTimeout,
		BlockingPoolSize:  constants.ValkeyConfig.BlockingPoolSize,
		PipelineMultiplex: constants.ValkeyConfig.PipelineMultiplex,
		Dialer:            net.Dialer{Timeout: constants.MQConfig.DialTimeout},
	})
	if err != nil {
		return nil, errors.NewCacheError("failed to create cache client", "init", "", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), constants.ValkeyConfig.ReadyTimeout)
	defer cancel()

	// Ping 테스트
	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		return nil, errors.NewCacheError("failed to connect to cache store", "ping", "", err)
	}

	logger.Info("Cache store connected",
		slog.String("addr", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)),
		slog.Int("db", cfg.DB),
		slog.Int("pool_size", constants.ValkeyConfig.BlockingPoolSize),
	)

	return &Service{
		client: client,
		logger: logger,
	}, nil
}

// Get: 키에 해당하는 값을 조회하고, 결과를 dest 인터페이스에 언마샬링한다.
func (c *Service) Get(ctx context.Context, key string, dest any) error {
	resp := c.client.Do(ctx, c.client.B().Get().Key(key).Build())
	if valkey.IsValkeyNil(resp.Error()) {
		return nil // Key doesn't exist - not an error
	}
	if resp.Error() != nil {
		c.logger.Error("Cache get operation failed", slog.String("key", key), slog.Any("error", resp.Error()))
		return errors.NewCacheError("get failed", "get", key, resp.Error())
	}

	value, err := resp.ToString()
	if err != nil {
		c.logger.Error("Cache value conversion failed", slog.String("key", key), slog.Any("error", err))
		return errors.NewCacheError("conversion failed", "get", key, err)
	}

	if dest != nil {
		if err := json.Unmarshal([]byte(value), dest); err != nil {
			c.logger.Error("Cache value unmarshal failed", slog.String("key", key), slog.Any("error", err))
			return errors.NewCacheError("unmarshal failed", "get", key, err)
		}
	}

	return nil
}

// MGet 배치 조회 (파이프라이닝 활용)
func (c *Service) MGet(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	resp := c.client.Do(ctx, c.client.B().Mget().Key(keys...).Build())
	if resp.Error() != nil {
		c.logger.Error("Cache mget failed", slog.Int("keys", len(keys)), slog.Any("error", resp.Error()))
		return nil, errors.NewCacheError("mget failed", "mget", fmt.Sprintf("%d keys", len(keys)), resp.Error())
	}

	values, err := resp.AsStrSlice()
	if err != nil {
		return nil, errors.NewCacheError("mget conversion failed", "mget", "", err)
	}

	result := make(map[string]string, len(keys))
	for i, key := range keys {
		if i < len(values) && values[i] != "" {
			result[key] = values[i]
		}
	}

	return result, nil
}

// Set: 값을 JSON으로 마샬링하여 키에 저장한다. (TTL 지정 가능)
func (c *Service) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return errors.NewCacheError("marshal failed", "set", key, err)
	}

	var cmd valkey.Completed
	if ttl > 0 {
		cmd = c.client.B().Set().Key(key).Value(string(jsonData)).ExSeconds(int64(ttl.Seconds())).Build()
	} else {
		cmd = c.client.B().Set().Key(key).Value(string(jsonData)).Build()
	}

	if err := c.client.Do(ctx, cmd).Error(); err != nil {
		c.logger.Error("Cache set failed", slog.String("key", key), slog.Any("error", err))
		return errors.NewCacheError("set failed", "set", key, err)
	}

	return nil
}

// MSet 배치 저장 (파이프라이닝 활용)
func (c *Service) MSet(ctx context.Context, pairs map[string]interface{}, ttl time.Duration) error {
	if len(pairs) == 0 {
		return nil
	}

	// 파이프라인 사용
	cmds := make([]valkey.Completed, 0, len(pairs))
	for key, value := range pairs {
		jsonData, err := json.Marshal(value)
		if err != nil {
			c.logger.Warn("Failed to marshal value for MSet", slog.String("key", key), slog.Any("error", err))
			continue
		}

		var cmd valkey.Completed
		if ttl > 0 {
			cmd = c.client.B().Set().Key(key).Value(string(jsonData)).ExSeconds(int64(ttl.Seconds())).Build()
		} else {
			cmd = c.client.B().Set().Key(key).Value(string(jsonData)).Build()
		}
		cmds = append(cmds, cmd)
	}

	// 배치 실행
	for _, resp := range c.client.DoMulti(ctx, cmds...) {
		if resp.Error() != nil {
			c.logger.Error("MSet command failed", slog.Any("error", resp.Error()))
			return errors.NewCacheError("mset failed", "mset", "", resp.Error())
		}
	}

	return nil
}

// Del: 지정된 키를 삭제한다.
func (c *Service) Del(ctx context.Context, key string) error {
	if err := c.client.Do(ctx, c.client.B().Del().Key(key).Build()).Error(); err != nil {
		c.logger.Error("Cache delete failed", slog.String("key", key), slog.Any("error", err))
		return errors.NewCacheError("delete failed", "del", key, err)
	}
	return nil
}

// DelMany: 여러 키를 한 번에 삭제한다.
func (c *Service) DelMany(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	resp := c.client.Do(ctx, c.client.B().Del().Key(keys...).Build())
	if resp.Error() != nil {
		c.logger.Error("Cache delete many failed", slog.Int("count", len(keys)), slog.Any("error", resp.Error()))
		return 0, errors.NewCacheError("delete many failed", "del", fmt.Sprintf("%d keys", len(keys)), resp.Error())
	}

	deleted, err := resp.AsInt64()
	if err != nil {
		return 0, errors.NewCacheError("delete many conversion failed", "del", "", err)
	}

	return deleted, nil
}

// Keys: 주어진 패턴과 일치하는 모든 키를 찾아서 반환한다. (주의: 대량 검색 시 부하 발생 가능)
func (c *Service) Keys(ctx context.Context, pattern string) ([]string, error) {
	resp := c.client.Do(ctx, c.client.B().Keys().Pattern(pattern).Build())
	if resp.Error() != nil {
		c.logger.Error("Cache keys search failed", slog.String("pattern", pattern), slog.Any("error", resp.Error()))
		return []string{}, errors.NewCacheError("keys search failed", "keys", pattern, resp.Error())
	}

	keys, err := resp.AsStrSlice()
	if err != nil {
		return []string{}, errors.NewCacheError("keys conversion failed", "keys", pattern, err)
	}

	return keys, nil
}

// SAdd: Set 자료구조에 멤버들을 추가한다.
func (c *Service) SAdd(ctx context.Context, key string, members []string) (int64, error) {
	if len(members) == 0 {
		return 0, nil
	}

	resp := c.client.Do(ctx, c.client.B().Sadd().Key(key).Member(members...).Build())
	if resp.Error() != nil {
		c.logger.Error("Cache sadd failed", slog.String("key", key), slog.Any("error", resp.Error()))
		return 0, errors.NewCacheError("sadd failed", "sadd", key, resp.Error())
	}

	added, err := resp.AsInt64()
	if err != nil {
		return 0, errors.NewCacheError("sadd conversion failed", "sadd", key, err)
	}

	return added, nil
}

// SRem: Set 자료구조에서 멤버들을 제거한다.
func (c *Service) SRem(ctx context.Context, key string, members []string) (int64, error) {
	if len(members) == 0 {
		return 0, nil
	}

	resp := c.client.Do(ctx, c.client.B().Srem().Key(key).Member(members...).Build())
	if resp.Error() != nil {
		c.logger.Error("Cache srem failed", slog.String("key", key), slog.Any("error", resp.Error()))
		return 0, errors.NewCacheError("srem failed", "srem", key, resp.Error())
	}

	removed, err := resp.AsInt64()
	if err != nil {
		return 0, errors.NewCacheError("srem conversion failed", "srem", key, err)
	}

	return removed, nil
}

// SMembers: Set의 모든 멤버를 조회한다.
func (c *Service) SMembers(ctx context.Context, key string) ([]string, error) {
	resp := c.client.Do(ctx, c.client.B().Smembers().Key(key).Build())
	if resp.Error() != nil {
		c.logger.Error("Cache smembers failed", slog.String("key", key), slog.Any("error", resp.Error()))
		return []string{}, errors.NewCacheError("smembers failed", "smembers", key, resp.Error())
	}

	members, err := resp.AsStrSlice()
	if err != nil {
		return []string{}, errors.NewCacheError("smembers conversion failed", "smembers", key, err)
	}

	return members, nil
}

// SIsMember: 특정 값이 Set에 포함되어 있는지 확인한다.
func (c *Service) SIsMember(ctx context.Context, key, member string) (bool, error) {
	resp := c.client.Do(ctx, c.client.B().Sismember().Key(key).Member(member).Build())
	if resp.Error() != nil {
		c.logger.Error("Cache sismember failed", slog.String("key", key), slog.Any("error", resp.Error()))
		return false, errors.NewCacheError("sismember failed", "sismember", key, resp.Error())
	}

	exists, err := resp.AsBool()
	if err != nil {
		return false, errors.NewCacheError("sismember conversion failed", "sismember", key, err)
	}

	return exists, nil
}

// HSet: Hash 자료구조의 특정 필드에 값을 설정한다.
func (c *Service) HSet(ctx context.Context, key, field, value string) error {
	if err := c.client.Do(ctx, c.client.B().Hset().Key(key).FieldValue().FieldValue(field, value).Build()).Error(); err != nil {
		c.logger.Error("Cache hset failed", slog.String("key", key), slog.String("field", field), slog.Any("error", err))
		return errors.NewCacheError("hset failed", "hset", key, err)
	}
	return nil
}

// HMSet: Hash 자료구조에 여러 필드와 값을 한 번에 설정한다.
func (c *Service) HMSet(ctx context.Context, key string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	builder := c.client.B().Hset().Key(key).FieldValue()
	for field, value := range fields {
		builder = builder.FieldValue(field, fmt.Sprintf("%v", value))
	}

	if err := c.client.Do(ctx, builder.Build()).Error(); err != nil {
		c.logger.Error("Cache hmset failed", slog.String("key", key), slog.Int("fields", len(fields)), slog.Any("error", err))
		return errors.NewCacheError("hmset failed", "hmset", key, err)
	}
	return nil
}

// HGet: Hash 자료구조에서 특정 필드의 값을 조회한다.
func (c *Service) HGet(ctx context.Context, key, field string) (string, error) {
	resp := c.client.Do(ctx, c.client.B().Hget().Key(key).Field(field).Build())
	if valkey.IsValkeyNil(resp.Error()) {
		return "", nil // Field doesn't exist - not an error
	}
	if resp.Error() != nil {
		c.logger.Error("Cache hash get failed", slog.String("key", key), slog.String("field", field), slog.Any("error", resp.Error()))
		return "", errors.NewCacheError("hget failed", "hget", key, resp.Error())
	}

	value, err := resp.ToString()
	if err != nil {
		return "", errors.NewCacheError("hget conversion failed", "hget", key, err)
	}

	return value, nil
}

// HGetAll: Hash의 모든 필드와 값을 조회한다.
func (c *Service) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	resp := c.client.Do(ctx, c.client.B().Hgetall().Key(key).Build())
	if resp.Error() != nil {
		c.logger.Error("Cache hgetall failed", slog.String("key", key), slog.Any("error", resp.Error()))
		return map[string]string{}, errors.NewCacheError("hgetall failed", "hgetall", key, resp.Error())
	}

	values, err := resp.AsStrMap()
	if err != nil {
		return map[string]string{}, errors.NewCacheError("hgetall conversion failed", "hgetall", key, err)
	}

	return values, nil
}

// Expire: 키의 만료 시간을 설정한다.
func (c *Service) Expire(ctx context.Context, key string, ttl time.Duration) error {
	if err := c.client.Do(ctx, c.client.B().Expire().Key(key).Seconds(int64(ttl.Seconds())).Build()).Error(); err != nil {
		c.logger.Error("Cache expire failed", slog.String("key", key), slog.Any("error", err))
		return errors.NewCacheError("expire failed", "expire", key, err)
	}
	return nil
}

// Exists: 키가 존재하는지 확인한다.
func (c *Service) Exists(ctx context.Context, key string) (bool, error) {
	resp := c.client.Do(ctx, c.client.B().Exists().Key(key).Build())
	if resp.Error() != nil {
		c.logger.Error("Cache exists failed", slog.String("key", key), slog.Any("error", resp.Error()))
		return false, errors.NewCacheError("exists failed", "exists", key, resp.Error())
	}

	count, err := resp.AsInt64()
	if err != nil {
		return false, errors.NewCacheError("exists conversion failed", "exists", key, err)
	}

	return count > 0, nil
}

// Close: 캐시 스토어 연결을 안전하게 종료한다.
func (c *Service) Close() error {
	var closeErr error

	c.closeOnce.Do(func() {
		if c.client == nil {
			return
		}

		c.client.Close()
		c.logger.Info("Cache store disconnected")
	})

	return closeErr
}

// IsConnected: 캐시 스토어와 연결되어 있는지(PING 응답 여부) 확인한다.
func (c *Service) IsConnected(ctx context.Context) bool {
	return c.client.Do(ctx, c.client.B().Ping().Build()).Error() == nil
}

// WaitUntilReady: 캐시 스토어 연결이 완료될 때까지 대기한다. (타임아웃 적용)
func (c *Service) WaitUntilReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for cache store to be ready")
		case <-ticker.C:
			if c.IsConnected(ctx) {
				return nil
			}
		}
	}
}

// InitializeMemberDatabase: 멤버 데이터베이스를 캐시 스토어에 초기화한다. (기존 데이터 삭제 후 갱신)
func (c *Service) InitializeMemberDatabase(ctx context.Context, memberData map[string]string) error {
	if err := c.client.Do(ctx, c.client.B().Del().Key(memberHashKey).Build()).Error(); err != nil {
		c.logger.Error("Failed to clear member database", slog.Any("error", err))
		return errors.NewCacheError("del failed", "del", memberHashKey, err)
	}

	if len(memberData) == 0 {
		c.logger.Info("Member database cleared (no members provided)")
		return nil
	}

	builder := c.client.B().Hset().Key(memberHashKey).FieldValue()
	for name, channelID := range memberData {
		builder = builder.FieldValue(name, channelID)
	}

	if err := c.client.Do(ctx, builder.Build()).Error(); err != nil {
		c.logger.Error("Failed to initialize member database", slog.Any("error", err))
		return errors.NewCacheError("hset failed", "hset", memberHashKey, err)
	}

	c.logger.Info("Member database initialized",
		slog.Int("members", len(memberData)),
	)
	return nil
}

// GetMemberChannelID: 멤버 이름으로 채널 ID를 조회한다.
func (c *Service) GetMemberChannelID(ctx context.Context, memberName string) (string, error) {
	if memberName == "" {
		return "", nil
	}

	resp := c.client.Do(ctx, c.client.B().Hget().Key(memberHashKey).Field(memberName).Build())
	if valkey.IsValkeyNil(resp.Error()) {
		return "", nil
	}
	if resp.Error() != nil {
		c.logger.Error("Failed to get member channel ID", slog.String("member", memberName), slog.Any("error", resp.Error()))
		return "", errors.NewCacheError("hget failed", "hget", memberHashKey, resp.Error())
	}

	value, err := resp.ToString()
	if err != nil {
		return "", errors.NewCacheError("hget conversion failed", "hget", memberHashKey, err)
	}

	return value, nil
}

// GetAllMembers: 캐시에 저장된 모든 멤버 정보를 조회한다.
func (c *Service) GetAllMembers(ctx context.Context) (map[string]string, error) {
	resp := c.client.Do(ctx, c.client.B().Hgetall().Key(memberHashKey).Build())
	if resp.Error() != nil {
		c.logger.Error("Failed to get all members", slog.Any("error", resp.Error()))
		return map[string]string{}, errors.NewCacheError("hgetall failed", "hgetall", memberHashKey, resp.Error())
	}

	values, err := resp.AsStrMap()
	if err != nil {
		return map[string]string{}, errors.NewCacheError("hgetall conversion failed", "hgetall", memberHashKey, err)
	}

	return values, nil
}

// AddMember: 멤버 정보를 캐시에 추가하거나 갱신한다.
func (c *Service) AddMember(ctx context.Context, memberName, channelID string) error {
	if memberName == "" || channelID == "" {
		return fmt.Errorf("member name and channel ID must be provided")
	}

	if err := c.client.Do(ctx, c.client.B().Hset().Key(memberHashKey).FieldValue().FieldValue(memberName, channelID).Build()).Error(); err != nil {
		c.logger.Error("Failed to add member", slog.String("member", memberName), slog.String("channel_id", channelID), slog.Any("error", err))
		return errors.NewCacheError("hset failed", "hset", memberHashKey, err)
	}
	c.logger.Info("Member added/updated",
		slog.String("member", memberName),
		slog.String("channel_id", channelID),
	)
	return nil
}

// GetStreams: 캐시된 방송 목록을 조회한다.
func (c *Service) GetStreams(ctx context.Context, key string) ([]*domain.Stream, bool) {
	var streams []*domain.Stream
	if err := c.Get(ctx, key, &streams); err != nil {
		c.logger.Debug("Cache miss or error", slog.String("key", key))
		return nil, false
	}

	if streams == nil {
		return nil, false
	}

	return streams, true
}

// SetStreams: 방송 목록을 캐시에 저장한다. (TTL 적용)
func (c *Service) SetStreams(ctx context.Context, key string, streams []*domain.Stream, ttl time.Duration) {
	if err := c.Set(ctx, key, streams, ttl); err != nil {
		c.logger.Error("Failed to cache streams", slog.String("key", key), slog.Any("error", err))
	}
}

// GetClient returns the underlying Valkey client for advanced operations
func (c *Service) GetClient() valkey.Client {
	return c.client
}
