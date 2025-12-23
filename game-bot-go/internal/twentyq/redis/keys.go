package redis

import (
	"fmt"
	"strings"

	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

func sessionKey(chatID string) string {
	return fmt.Sprintf("%s:%s", qconfig.RedisKeySessionPrefix, strings.TrimSpace(chatID))
}

func historyKey(chatID string) string {
	return fmt.Sprintf("%s:%s", qconfig.RedisKeyHistoryPrefix, strings.TrimSpace(chatID))
}

func categoryKey(chatID string) string {
	return fmt.Sprintf("%s:%s", qconfig.RedisKeyCategory, strings.TrimSpace(chatID))
}

func hintCountKey(chatID string) string {
	return fmt.Sprintf("%s:%s", qconfig.RedisKeyHints, strings.TrimSpace(chatID))
}

func playersKey(chatID string) string {
	return fmt.Sprintf("%s:%s", qconfig.RedisKeyPlayers, strings.TrimSpace(chatID))
}

func wrongGuessSessionKey(chatID string) string {
	return fmt.Sprintf("%s:%s", qconfig.RedisKeyWrongGuesses, strings.TrimSpace(chatID))
}

func wrongGuessUserKey(chatID string, userID string) string {
	return fmt.Sprintf("%s:%s:%s", qconfig.RedisKeyWrongGuesses, strings.TrimSpace(chatID), strings.TrimSpace(userID))
}

func topicsGlobalKey(chatID string) string {
	return fmt.Sprintf("%s:%s", qconfig.RedisKeyTopics, strings.TrimSpace(chatID))
}

func topicsCategoryKey(chatID string, category string) string {
	return fmt.Sprintf("%s:%s:%s", qconfig.RedisKeyTopics, strings.TrimSpace(chatID), strings.TrimSpace(category))
}

func voteKey(chatID string) string {
	return fmt.Sprintf("%s:%s", qconfig.RedisKeyVotePrefix, strings.TrimSpace(chatID))
}

func processingKey(chatID string) string {
	return fmt.Sprintf("%s:processing:%s", qconfig.RedisKeyLockPrefix, strings.TrimSpace(chatID))
}

func lockKey(chatID string) string {
	return fmt.Sprintf("%s:%s", qconfig.RedisKeyLockPrefix, strings.TrimSpace(chatID))
}

func readLockKey(chatID string) string {
	return fmt.Sprintf("%s:%s:read", qconfig.RedisKeyLockPrefix, strings.TrimSpace(chatID))
}

func chainSkipFlagKey(chatID string, userID string) string {
	return fmt.Sprintf("%s:chain_skip:%s:%s", qconfig.RedisKeyPendingPrefix, strings.TrimSpace(chatID), strings.TrimSpace(userID))
}
