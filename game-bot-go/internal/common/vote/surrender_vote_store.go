package vote

import (
	"context"
	"fmt"
	"time"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	domainmodels "github.com/park285/llm-kakao-bots/game-bot-go/internal/domain/models"
)

// SurrenderVoteKeyFunc: 채팅방 ID로 투표 저장 키를 생성하는 함수 타입입니다.
type SurrenderVoteKeyFunc func(chatID string) string

// SurrenderVoteStore: 항복 투표 상태를 Redis에 저장하고 관리하는 공통 저장소입니다.
// 게임별로 다른 키 프리픽스/TTL만 주입하여 동일한 저장 로직을 재사용하기 위함입니다.
type SurrenderVoteStore struct {
	client  valkey.Client
	keyFunc SurrenderVoteKeyFunc
	ttl     time.Duration
}

// NewSurrenderVoteStore: 새로운 SurrenderVoteStore 인스턴스를 생성합니다.
func NewSurrenderVoteStore(client valkey.Client, keyFunc SurrenderVoteKeyFunc, ttl time.Duration) *SurrenderVoteStore {
	return &SurrenderVoteStore{
		client:  client,
		keyFunc: keyFunc,
		ttl:     ttl,
	}
}

// Get: 현재 활성화된 투표 상태를 조회합니다. 투표가 없으면 nil을 반환합니다.
func (s *SurrenderVoteStore) Get(ctx context.Context, chatID string) (*domainmodels.SurrenderVote, error) {
	key := s.keyFunc(chatID)

	raw, ok, err := valkeyx.GetBytes(ctx, s.client, key)
	if err != nil {
		return nil, cerrors.RedisError{Operation: "vote_get", Err: err}
	}
	if !ok {
		return nil, nil
	}

	var vote domainmodels.SurrenderVote
	if err := json.Unmarshal(raw, &vote); err != nil {
		return nil, cerrors.RedisError{Operation: "vote_unmarshal", Err: err}
	}
	return &vote, nil
}

// Save: 변경된 투표 상태를 Redis에 저장하고 TTL을 갱신합니다.
func (s *SurrenderVoteStore) Save(ctx context.Context, chatID string, vote domainmodels.SurrenderVote) error {
	key := s.keyFunc(chatID)

	raw, err := json.Marshal(vote)
	if err != nil {
		return cerrors.RedisError{Operation: "vote_marshal", Err: err}
	}

	if err := valkeyx.SetStringEX(ctx, s.client, key, string(raw), s.ttl); err != nil {
		return cerrors.RedisError{Operation: "vote_save", Err: err}
	}
	return nil
}

// Clear: 투표 상태를 Redis에서 삭제합니다. (투표 완료 또는 취소 시)
func (s *SurrenderVoteStore) Clear(ctx context.Context, chatID string) error {
	key := s.keyFunc(chatID)

	if err := valkeyx.DeleteKeys(ctx, s.client, key); err != nil {
		return cerrors.RedisError{Operation: "vote_clear", Err: err}
	}
	return nil
}

// Exists: 현재 활성화된(진행 중인) 투표가 있는지 키 존재 여부로 확인합니다.
func (s *SurrenderVoteStore) Exists(ctx context.Context, chatID string) (bool, error) {
	key := s.keyFunc(chatID)

	cmd := s.client.B().Exists().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, cerrors.RedisError{Operation: "vote_exists", Err: err}
	}
	return n > 0, nil
}

// Approve: 특정 사용자의 '찬성' 의사를 투표에 반영하고, 갱신된 투표 상태를 반환합니다.
func (s *SurrenderVoteStore) Approve(ctx context.Context, chatID string, userID string) (*domainmodels.SurrenderVote, error) {
	vote, err := s.Get(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if vote == nil {
		return nil, nil
	}

	updated, err := vote.Approve(userID)
	if err != nil {
		return nil, fmt.Errorf("vote approve failed: %w", err)
	}
	if err := s.Save(ctx, chatID, updated); err != nil {
		return nil, err
	}
	return &updated, nil
}
