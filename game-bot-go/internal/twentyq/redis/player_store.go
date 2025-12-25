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

// PlayerStore: 현재 게임에 참여 중인 플레이어 목록(ID, 이름 등)을 Redis에 저장하고 관리하는 저장소
type PlayerStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewPlayerStore: 새로운 PlayerStore 인스턴스를 생성한다.
func NewPlayerStore(client valkey.Client, logger *slog.Logger) *PlayerStore {
	return &PlayerStore{
		client: client,
		logger: logger,
	}
}

// Add: 특정 플레이어를 참여자 목록에 추가한다. 이미 존재하는 경우 정보를 갱신한다.
// 목록 전체를 조회 후 수정하여 덮어쓰는 방식을 사용한다. (Lock 필요 레벨에서 동시성 제어 권장)
func (s *PlayerStore) Add(ctx context.Context, chatID string, userID string, sender string) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false, fmt.Errorf("invalid user id")
	}
	sender = strings.TrimSpace(sender)

	current, err := s.GetAll(ctx, chatID)
	if err != nil {
		return false, err
	}

	isNew := true
	updated := make([]qmodel.PlayerInfo, 0, len(current)+1)
	for _, p := range current {
		if p.UserID == userID {
			isNew = false
			if sender != "" {
				p.Sender = sender
			}
		}
		updated = append(updated, p)
	}

	if isNew {
		updated = append(updated, qmodel.PlayerInfo{UserID: userID, Sender: sender})
	}

	payload, err := json.Marshal(updated)
	if err != nil {
		return false, fmt.Errorf("marshal player list failed: %w", err)
	}

	key := playersKey(chatID)
	cmd := s.client.B().Set().Key(key).Value(string(payload)).Ex(time.Duration(qconfig.RedisSessionTTLSeconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return false, cerrors.RedisError{Operation: "players_save", Err: err}
	}

	s.logger.Debug("player_saved", "chat_id", chatID, "user_id", userID, "is_new", isNew, "total", len(updated))
	return isNew, nil
}

// GetAll: Redis에 저장된 JSON 형태의 플레이어 목록을 파싱하여 반환한다.
func (s *PlayerStore) GetAll(ctx context.Context, chatID string) ([]qmodel.PlayerInfo, error) {
	key := playersKey(chatID)

	cmd := s.client.B().Get().Key(key).Build()
	raw, err := s.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, cerrors.RedisError{Operation: "players_get", Err: err}
	}

	var out []qmodel.PlayerInfo
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, cerrors.RedisError{Operation: "players_unmarshal", Err: err}
	}
	return out, nil
}

// Clear: 게임 종료 시 참여자 목록 데이터를 삭제한다.
func (s *PlayerStore) Clear(ctx context.Context, chatID string) error {
	key := playersKey(chatID)
	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return cerrors.RedisError{Operation: "players_clear", Err: err}
	}
	return nil
}
