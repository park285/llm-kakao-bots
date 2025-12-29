// Package redis 는 TurtleSoup 게임의 Redis/Valkey 키 생성 함수들을 정의합니다.
package redis

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	tsconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/turtlesoup/config"
)

// sessionKey: 게임 세션 데이터 저장용 키를 생성합니다.
// 형식: turtle:session:{sessionID}
func sessionKey(sessionID string) string {
	return valkeyx.BuildKey(tsconfig.RedisKeySessionPrefix, sessionID)
}

// lockKey: 세션 락 키를 생성합니다.
// 형식: turtle:lock:{sessionID}
func lockKey(sessionID string) string {
	return valkeyx.BuildKey(tsconfig.RedisKeyLockPrefix, sessionID)
}

// lockHolderKey: 락 보유자 정보 저장용 키를 생성합니다.
// 형식: turtle:lock:holder:{sessionID}
func lockHolderKey(sessionID string) string {
	return valkeyx.BuildKeySuffix(tsconfig.RedisKeyLockPrefix, "holder", sessionID)
}

// voteKey: 포기 투표 저장용 키를 생성합니다.
// 형식: turtle:vote:{chatID}
func voteKey(chatID string) string {
	return valkeyx.BuildKey(tsconfig.RedisKeyVotePrefix, chatID)
}

// processingKey: 메시지 처리 중 상태 저장용 키를 생성합니다.
// 형식: turtle:processing:{chatID}
func processingKey(chatID string) string {
	return valkeyx.BuildKey(tsconfig.RedisKeyProcessing, chatID)
}

// puzzleChatKey: 채팅방별 퍼즐 중복 방지용 키를 생성합니다.
// 형식: turtle:puzzle:chat:{chatID}
func puzzleChatKey(chatID string) string {
	return valkeyx.BuildKey(tsconfig.RedisKeyPuzzleChat, chatID)
}

func pendingKeyPrefix() string {
	return tsconfig.RedisKeyPendingPrefix
}
