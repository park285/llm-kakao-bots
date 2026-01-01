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
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/gamesession"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
	qmodel "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/model"
)

// SessionStore: 스무고개 게임의 정답(Secret) 및 세션 정보를 Redis에 저장하고 관리하는 저장소
// 공통 gamesession.Store를 내부적으로 사용하여 핵심 CRUD 로직을 위임합니다.
type SessionStore struct {
	base *gamesession.Store[qmodel.RiddleSecret]
}

// NewSessionStore: 새로운 SessionStore 인스턴스를 생성합니다.
func NewSessionStore(client valkey.Client, logger *slog.Logger) *SessionStore {
	return &SessionStore{
		base: gamesession.NewStore[qmodel.RiddleSecret](client, logger, gamesession.Config{
			KeyFunc: sessionKey,
			TTL:     time.Duration(qconfig.RedisSessionTTLSeconds) * time.Second,
		}),
	}
}

// SaveSecret: 게임의 정답(비밀) 정보를 Redis에 저장합니다. (TTL 설정됨)
func (s *SessionStore) SaveSecret(ctx context.Context, chatID string, secret qmodel.RiddleSecret) error {
	if err := s.base.Save(ctx, chatID, secret); err != nil {
		return fmt.Errorf("save secret: %w", err)
	}
	s.base.Logger().Info("secret_saved", "chat_id", chatID)
	return nil
}

// GetSecret: 현재 진행 중인 게임의 정답 정보를 조회합니다. (없으면 nil 반환)
func (s *SessionStore) GetSecret(ctx context.Context, chatID string) (*qmodel.RiddleSecret, error) {
	secret, err := s.base.Load(ctx, chatID)
	if err != nil {
		return nil, fmt.Errorf("get secret: %w", err)
	}
	return secret, nil
}

// Delete: 게임 세션(정답 정보)을 삭제합니다. (게임 종료 시)
func (s *SessionStore) Delete(ctx context.Context, chatID string) error {
	if err := s.base.Delete(ctx, chatID); err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}
	s.base.Logger().Info("secret_deleted", "chat_id", chatID)
	return nil
}

// Exists: 특정 채팅방에 진행 중인 게임 세션이 존재하는지 확인합니다.
func (s *SessionStore) Exists(ctx context.Context, chatID string) (bool, error) {
	exists, err := s.base.Exists(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("secret exists: %w", err)
	}
	return exists, nil
}

// RefreshTTL: 세션의 유효 기간(TTL)을 초기화하여 연장합니다.
func (s *SessionStore) RefreshTTL(ctx context.Context, chatID string) (bool, error) {
	ok, err := s.base.RefreshTTL(ctx, chatID)
	if err != nil {
		return false, fmt.Errorf("refresh ttl failed: %w", err)
	}
	return ok, nil
}

// Get: GetSecret의 별칭으로, 세션 정보를 조회합니다.
func (s *SessionStore) Get(ctx context.Context, chatID string) (*qmodel.RiddleSecret, error) {
	return s.GetSecret(ctx, chatID)
}

// PlayerInfo 플레이어 정보 (닉네임 조회용).
type PlayerInfo struct {
	UserID string
	Sender string
}

// GetPlayerByNickname: 현재 세션의 참여자 목록에서 닉네임(Sender)을 기준으로 플레이어 정보를 조회합니다.
// 닉네임 매칭은 대소문자를 구분하지 않으며, 일치하는 사용자가 없거나 조회 실패 시 nil을 반환합니다.
func (s *SessionStore) GetPlayerByNickname(ctx context.Context, chatID string, nickname string) *PlayerInfo {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return nil
	}

	client := s.base.Client()
	logger := s.base.Logger()

	key := playersKey(chatID)
	cmd := client.B().Get().Key(key).Build()
	raw, err := client.Do(ctx, cmd).AsBytes()
	if err != nil {
		if !valkeyx.IsNil(err) {
			logger.Warn("get_player_by_nickname_redis_failed", "chat_id", chatID, "nickname", nickname, "err", err)
		}
		return nil
	}

	var players []qmodel.PlayerInfo
	if err := json.Unmarshal(raw, &players); err != nil {
		logger.Warn("get_player_by_nickname_unmarshal_failed", "chat_id", chatID, "err", err)
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

// ClearAllData: 채팅방에 연관된 스무고개 관련 Redis 데이터(세션, 히스토리, 락 등)를 삭제합니다. (초기화)
func (s *SessionStore) ClearAllData(ctx context.Context, chatID string) error {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil
	}

	client := s.base.Client()
	logger := s.base.Logger()

	// 먼저 players 조회하여 유저별 wrongGuesses 키도 삭제
	playersCmd := client.B().Get().Key(playersKey(chatID)).Build()
	playersRaw, err := client.Do(ctx, playersCmd).AsBytes()
	var players []qmodel.PlayerInfo
	if err != nil && !valkeyx.IsNil(err) {
		logger.Warn("clear_all_players_get_failed", "chat_id", chatID, "err", err)
	}
	if playersRaw != nil {
		if err := json.Unmarshal(playersRaw, &players); err != nil {
			logger.Warn("clear_all_players_unmarshal_failed", "chat_id", chatID, "err", err)
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
		fmt.Sprintf("%s:data:{%s}", qconfig.RedisKeyPendingPrefix, chatID),  // 20q:pending-messages:data:{chatID}
		fmt.Sprintf("%s:order:{%s}", qconfig.RedisKeyPendingPrefix, chatID), // 20q:pending-messages:order:{chatID}
		fmt.Sprintf("%s:%s", qconfig.RedisKeyTopics, chatID),                // 20q:topics:{chatID}
		lockKey(chatID),       // 20q:lock:{chatID}
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
		cmd := client.B().Del().Key(keys...).Build()
		if err := client.Do(ctx, cmd).Error(); err != nil {
			return fmt.Errorf("clear all data: %w", cerrors.RedisError{Operation: "clear_all", Err: err})
		}
	}

	logger.Info("all_data_cleared", "chat_id", chatID, "keys_deleted", len(keys))
	return nil
}
