package redis

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	json "github.com/goccy/go-json"
	"github.com/valkey-io/valkey-go"

	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/errors"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// SessionStore 는 타입이다.
type SessionStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewSessionStore 는 동작을 수행한다.
func NewSessionStore(client valkey.Client, logger *slog.Logger) *SessionStore {
	return &SessionStore{
		client: client,
		logger: logger,
	}
}

// SaveSecret 는 동작을 수행한다.
func (s *SessionStore) SaveSecret(ctx context.Context, chatID string, secret qmodel.RiddleSecret) error {
	key := sessionKey(chatID)

	payload, err := json.Marshal(secret)
	if err != nil {
		return qerrors.RedisError{Operation: "marshal_secret", Err: err}
	}

	cmd := s.client.B().Set().Key(key).Value(string(payload)).Ex(time.Duration(qconfig.RedisSessionTTLSeconds) * time.Second).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return qerrors.RedisError{Operation: "save_secret", Err: err}
	}

	s.logger.Info("secret_saved", "chat_id", chatID)
	return nil
}

// GetSecret 는 동작을 수행한다.
func (s *SessionStore) GetSecret(ctx context.Context, chatID string) (*qmodel.RiddleSecret, error) {
	key := sessionKey(chatID)

	cmd := s.client.B().Get().Key(key).Build()
	raw, err := s.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, qerrors.RedisError{Operation: "get_secret", Err: err}
	}

	var secret qmodel.RiddleSecret
	if err := json.Unmarshal(raw, &secret); err != nil {
		return nil, qerrors.RedisError{Operation: "unmarshal_secret", Err: err}
	}
	return &secret, nil
}

// Delete 는 동작을 수행한다.
func (s *SessionStore) Delete(ctx context.Context, chatID string) error {
	key := sessionKey(chatID)

	cmd := s.client.B().Del().Key(key).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return qerrors.RedisError{Operation: "delete_secret", Err: err}
	}
	s.logger.Info("secret_deleted", "chat_id", chatID)
	return nil
}

// Exists 는 동작을 수행한다.
func (s *SessionStore) Exists(ctx context.Context, chatID string) (bool, error) {
	key := sessionKey(chatID)

	cmd := s.client.B().Exists().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		return false, qerrors.RedisError{Operation: "secret_exists", Err: err}
	}
	return n > 0, nil
}

// RefreshTTL 는 동작을 수행한다.
func (s *SessionStore) RefreshTTL(ctx context.Context, chatID string) (bool, error) {
	key := sessionKey(chatID)

	cmd := s.client.B().Expire().Key(key).Seconds(int64(qconfig.RedisSessionTTLSeconds)).Build()
	ok, err := s.client.Do(ctx, cmd).AsBool()
	if err != nil {
		return false, fmt.Errorf("refresh ttl failed: %w", qerrors.RedisError{Operation: "secret_refresh_ttl", Err: err})
	}
	return ok, nil
}

// Get 세션 조회 (GetSecret 별칭).
func (s *SessionStore) Get(ctx context.Context, chatID string) (*qmodel.RiddleSecret, error) {
	return s.GetSecret(ctx, chatID)
}

// PlayerInfo 플레이어 정보 (닉네임 조회용).
type PlayerInfo struct {
	UserID string
	Sender string
}

// GetPlayerByNickname 닉네임으로 플레이어 조회.
// 현재 진행 중인 세션의 참여자 목록에서 닉네임으로 검색.
// 조회 실패 또는 닉네임 미존재 시 nil 반환 (graceful degradation).
func (s *SessionStore) GetPlayerByNickname(ctx context.Context, chatID string, nickname string) *PlayerInfo {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return nil
	}

	key := playersKey(chatID)
	cmd := s.client.B().Get().Key(key).Build()
	raw, err := s.client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if !valkeyx.IsNil(err) {
			s.logger.Warn("get_player_by_nickname_redis_failed", "chat_id", chatID, "nickname", nickname, "err", err)
		}
		return nil
	}

	var players []qmodel.PlayerInfo
	if err := json.Unmarshal(raw, &players); err != nil {
		s.logger.Warn("get_player_by_nickname_unmarshal_failed", "chat_id", chatID, "err", err)
		return nil
	}

	for _, p := range players {
		if strings.EqualFold(strings.TrimSpace(p.Sender), nickname) {
			return &PlayerInfo{
				UserID: p.UserID,
				Sender: p.Sender,
			}
		}
	}
	return nil
}

// ClearAllData 채팅방의 모든 20Q 관련 데이터 삭제.
func (s *SessionStore) ClearAllData(ctx context.Context, chatID string) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil
	}

	// 먼저 players 조회하여 유저별 wrongGuesses 키도 삭제
	playersCmd := s.client.B().Get().Key(playersKey(chatID)).Build()
	playersRaw, err := s.client.Do(ctx, playersCmd).AsBytes()
	var players []qmodel.PlayerInfo
	if err != nil && !valkeyx.IsNil(err) {
		s.logger.Warn("clear_all_players_get_failed", "chat_id", chatID, "err", err)
	}
	if playersRaw != nil {
		if err := json.Unmarshal(playersRaw, &players); err != nil {
			s.logger.Warn("clear_all_players_unmarshal_failed", "chat_id", chatID, "err", err)
		}
	}

	// 고정 키 목록 (SCAN 없이 직접 삭제)
	keys := []string{
		sessionKey(chatID),           // 20q:riddle:session:{chatID}
		playersKey(chatID),           // 20q:players:{chatID}
		historyKey(chatID),           // 20q:history:{chatID}
		categoryKey(chatID),          // 20q:category:{chatID}
		hintCountKey(chatID),         // 20q:hints:{chatID}
		wrongGuessSessionKey(chatID), // 20q:wrongGuesses:{chatID}
		voteKey(chatID),              // 20q:surrender:vote:{chatID}
		fmt.Sprintf("%s:%s", qconfig.RedisKeyPendingPrefix, chatID), // 20q:pending-messages:{chatID}
		fmt.Sprintf("%s:%s", qconfig.RedisKeyTopics, chatID),        // 20q:topics:{chatID}
		lockKey(chatID),       // 20q:lock:{chatID}
		readLockKey(chatID),   // 20q:lock:{chatID}:read
		processingKey(chatID), // 20q:lock:processing:{chatID}
	}

	// 유저별 wrongGuesses 및 chain_skip 키 추가
	for _, p := range players {
		userID := strings.TrimSpace(p.UserID)
		if userID == "" {
			continue
		}
		keys = append(keys, wrongGuessUserKey(chatID, userID), chainSkipFlagKey(chatID, userID))
	}

	// 카테고리별 topics 키 추가 (중앙 관리 목록 사용)
	for _, cat := range qconfig.AllCategories {
		keys = append(keys, topicsCategoryKey(chatID, cat))
	}

	// 단일 DEL 명령으로 모든 키 삭제 (파이프라인 불필요)
	if len(keys) > 0 {
		cmd := s.client.B().Del().Key(keys...).Build()
		if err := s.client.Do(ctx, cmd).Error(); err != nil {
			return fmt.Errorf("clear all data: %w", qerrors.RedisError{Operation: "clear_all", Err: err})
		}
	}

	s.logger.Info("all_data_cleared", "chat_id", chatID, "keys_deleted", len(keys))
	return nil
}
