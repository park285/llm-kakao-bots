package redis

import (
	"context"
	"log/slog"
	"strings"

	"github.com/valkey-io/valkey-go"

	cerrors "github.com/park285/llm-kakao-bots/game-bot-go/internal/common/errors"
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

// WrongGuessStore: 사용자 및 세션별 오답 기록을 Redis에 저장하고 조회하는 저장소
type WrongGuessStore struct {
	client valkey.Client
	logger *slog.Logger
}

// NewWrongGuessStore: 새로운 WrongGuessStore 인스턴스를 생성합니다.
func NewWrongGuessStore(client valkey.Client, logger *slog.Logger) *WrongGuessStore {
	return &WrongGuessStore{
		client: client,
		logger: logger,
	}
}

// Add: 사용자 및 세션의 오답 기록을 추가합니다.
func (s *WrongGuessStore) Add(ctx context.Context, chatID string, userID string, guess string) error {
	guess = strings.TrimSpace(guess)
	if guess == "" {
		return nil
	}

	sessionKey := wrongGuessSessionKey(chatID)
	userKey := wrongGuessUserKey(chatID, userID)
	ttl := int64(qconfig.RedisSessionTTLSeconds)

	// DoMulti로 4개 명령을 단일 RTT로 처리
	saddSessionCmd := s.client.B().Sadd().Key(sessionKey).Member(guess).Build()
	saddUserCmd := s.client.B().Sadd().Key(userKey).Member(guess).Build()
	expireSessionCmd := s.client.B().Expire().Key(sessionKey).Seconds(ttl).Build()
	expireUserCmd := s.client.B().Expire().Key(userKey).Seconds(ttl).Build()

	results := s.client.DoMulti(ctx, saddSessionCmd, saddUserCmd, expireSessionCmd, expireUserCmd)
	for _, r := range results {
		if err := r.Error(); err != nil && !valkeyx.IsNil(err) {
			return cerrors.RedisError{Operation: "wrong_guess_add", Err: err}
		}
	}
	return nil
}

// GetSessionWrongGuesses: 현재 세션에서 발생한 모든 오답 목록을 조회합니다.
func (s *WrongGuessStore) GetSessionWrongGuesses(ctx context.Context, chatID string) ([]string, error) {
	key := wrongGuessSessionKey(chatID)
	cmd := s.client.B().Smembers().Key(key).Build()
	values, err := s.client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, cerrors.RedisError{Operation: "wrong_guess_get_session", Err: err}
	}
	return values, nil
}

// GetUserWrongGuesses: 특정 사용자가 해당 세션에서 입력한 오답 목록을 조회합니다.
func (s *WrongGuessStore) GetUserWrongGuesses(ctx context.Context, chatID string, userID string) ([]string, error) {
	key := wrongGuessUserKey(chatID, userID)
	cmd := s.client.B().Smembers().Key(key).Build()
	values, err := s.client.Do(ctx, cmd).AsStrSlice()
	if err != nil {
		if valkeyx.IsNil(err) {
			return nil, nil
		}
		return nil, cerrors.RedisError{Operation: "wrong_guess_get_user", Err: err}
	}
	return values, nil
}

// GetUserWrongGuessCount: 특정 사용자의 오답 횟수를 조회합니다.
func (s *WrongGuessStore) GetUserWrongGuessCount(ctx context.Context, chatID string, userID string) (int, error) {
	key := wrongGuessUserKey(chatID, userID)
	cmd := s.client.B().Scard().Key(key).Build()
	n, err := s.client.Do(ctx, cmd).AsInt64()
	if err != nil {
		if valkeyx.IsNil(err) {
			return 0, nil
		}
		return 0, cerrors.RedisError{Operation: "wrong_guess_count_user", Err: err}
	}
	if n < 0 {
		return 0, nil
	}
	return int(n), nil
}

// GetUserWrongGuessCountBatch 여러 유저의 오답 수를 파이프라인으로 일괄 조회.
func (s *WrongGuessStore) GetUserWrongGuessCountBatch(ctx context.Context, chatID string, userIDs []string) (map[string]int, error) {
	if len(userIDs) == 0 {
		return make(map[string]int), nil
	}

	// 명령 빌드
	cmds := make([]valkey.Completed, 0, len(userIDs))
	validUserIDs := make([]string, 0, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		key := wrongGuessUserKey(chatID, userID)
		cmds = append(cmds, s.client.B().Scard().Key(key).Build())
		validUserIDs = append(validUserIDs, userID)
	}

	if len(cmds) == 0 {
		return make(map[string]int), nil
	}

	results := s.client.DoMulti(ctx, cmds...)

	result := make(map[string]int, len(validUserIDs))
	for i, r := range results {
		if i >= len(validUserIDs) {
			break
		}
		userID := validUserIDs[i]
		n, err := r.AsInt64()
		if err != nil {
			result[userID] = 0
			continue
		}
		if n < 0 {
			n = 0
		}
		result[userID] = int(n)
	}
	return result, nil
}

// Delete: 세션 종료 시 오답 기록을 일괄 삭제합니다.
func (s *WrongGuessStore) Delete(ctx context.Context, chatID string, userIDs []string) error {
	keys := make([]string, 0, 1+len(userIDs))
	keys = append(keys, wrongGuessSessionKey(chatID))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		keys = append(keys, wrongGuessUserKey(chatID, userID))
	}

	cmd := s.client.B().Del().Key(keys...).Build()
	if err := s.client.Do(ctx, cmd).Error(); err != nil {
		return cerrors.RedisError{Operation: "wrong_guess_delete", Err: err}
	}
	return nil
}
