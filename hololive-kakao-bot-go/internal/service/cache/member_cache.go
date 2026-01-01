package cache

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/valkey-io/valkey-go"

	"github.com/kapu/hololive-kakao-bot-go/pkg/errors"
)

const memberHashKey = "hololive:members"

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

// GetMemberChannelID: 멤버 이름으로 채널 ID를 조회합니다.
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

// GetAllMembers: 캐시에 저장된 모든 멤버 정보를 조회합니다.
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

// AddMember: 멤버 정보를 캐시에 추가하거나 갱신합니다.
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
