package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// SurrenderVoteStore: 게임 항복(Surrender) 투표의 진행 상황과 상태를 Redis에 저장하고 관리하는 저장소
type SurrenderVoteStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewSurrenderVoteStore: 새로운 SurrenderVoteStore 인스턴스를 생성합니다.
func NewSurrenderVoteStore(client valkey.Client, logger *slog.Logger) *SurrenderVoteStore {
	return &SurrenderVoteStore{
		client: client,
		logger: logger,
	}
}

// Get: 현재 진행 중인 투표 상태(찬성자 목록, 만료 시간 등)를 조회합니다.
func (s *SurrenderVoteStore) Get(ctx context.Context, chatID string) (*qmodel.SurrenderVote, error) {
	key := voteKey(chatID)

	raw, ok, err := valkeyx.GetBytes(ctx, s.client, key)
	if err != nil {
		return nil, cerrors.RedisError{Operation: "vote_get", Err: err}
	}
	if !ok {
		return nil, nil
	}

	var vote qmodel.SurrenderVote
	if err := json.Unmarshal(raw, &vote); err != nil {
		return nil, cerrors.RedisError{Operation: "vote_unmarshal", Err: err}
	}
	return &vote, nil
}

// Save: 변경된 투표 상태를 Redis에 저장(덮어쓰기)하고 TTL을 설정합니다.
func (s *SurrenderVoteStore) Save(ctx context.Context, chatID string, vote qmodel.SurrenderVote) error {
	key := voteKey(chatID)

	payload, err := json.Marshal(vote)
	if err != nil {
		return fmt.Errorf("marshal surrender vote failed: %w", err)
	}

	ttl := time.Duration(qconfig.RedisVoteTTLSeconds) * time.Second
	if err := valkeyx.SetStringEX(ctx, s.client, key, string(payload), ttl); err != nil {
		return cerrors.RedisError{Operation: "vote_save", Err: err}
	}
	return nil
}

// Clear: 투표가 종료되거나 취소되었을 때 데이터를 삭제합니다.
func (s *SurrenderVoteStore) Clear(ctx context.Context, chatID string) error {
	key := voteKey(chatID)
	if err := valkeyx.DeleteKeys(ctx, s.client, key); err != nil {
		return cerrors.RedisError{Operation: "vote_clear", Err: err}
	}
	return nil
}

// Exists: 현재 활성화된(진행 중인) 투표가 있는지 키 존재 여부로 확인합니다.
func (s *SurrenderVoteStore) Exists(ctx context.Context, chatID string) (bool, error) {
	key := voteKey(chatID)
	cmd := s.client.B().Exists().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, cerrors.RedisError{Operation: "vote_exists", Err: err}
	}
	return n > 0, nil
}

// Approve: 특정 사용자의 '찬성' 의사를 투표 상태에 반영합니다.
// 투표 상태를 조회(Get)하고, 찬성 처리(Approve) 후, 다시 저장(Save)하는 과정을 수행합니다.
func (s *SurrenderVoteStore) Approve(ctx context.Context, chatID string, userID string) (*qmodel.SurrenderVote, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, fmt.Errorf("invalid user id")
	}

	vote, err := s.Get(ctx, chatID)
	if err != nil {
		return nil, err
	}
	if vote == nil {
		return nil, nil
	}

	next, approveErr := vote.Approve(userID)
	if approveErr != nil {
		return nil, fmt.Errorf("vote approve failed: %w", approveErr)
	}

	if err := s.Save(ctx, chatID, next); err != nil {
		s.logger.Warn("vote_save_failed", "chat_id", chatID, "err", err)
		return nil, err
	}
	return &next, nil
}
