package redis

import (
	"fmt"
	"strings"

	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
)

func sessionKey(sessionID string) string {
	return fmt.Sprintf("%s:%s", tsconfig.RedisKeySessionPrefix, strings.TrimSpace(sessionID))
}

func lockKey(sessionID string) string {
	return fmt.Sprintf("%s:%s", tsconfig.RedisKeyLockPrefix, strings.TrimSpace(sessionID))
}

func lockHolderKey(sessionID string) string {
	return fmt.Sprintf("%s:holder:%s", tsconfig.RedisKeyLockPrefix, strings.TrimSpace(sessionID))
}

func voteKey(chatID string) string {
	return fmt.Sprintf("%s:%s", tsconfig.RedisKeyVotePrefix, strings.TrimSpace(chatID))
}

func processingKey(chatID string) string {
	return fmt.Sprintf("%s:%s", tsconfig.RedisKeyProcessing, strings.TrimSpace(chatID))
}

func puzzleChatKey(chatID string) string {
	return fmt.Sprintf("%s:%s", tsconfig.RedisKeyPuzzleChat, strings.TrimSpace(chatID))
}
