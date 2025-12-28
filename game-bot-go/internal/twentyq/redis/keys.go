// Package redis 는 TwentyQ 게임의 Redis/Valkey 키 생성 함수들을 정의한다.
package redis

import (
	"github.com/park285/llm-kakao-bots/game-bot-go/internal/common/valkeyx"
	qconfig "github.com/park285/llm-kakao-bots/game-bot-go/internal/twentyq/config"
)

// sessionKey 는 게임 세션 데이터 저장용 키를 생성한다.
// 형식: 20q:riddle:session:{chatID}
func sessionKey(chatID string) string {
	return valkeyx.BuildKey(qconfig.RedisKeySessionPrefix, chatID)
}

// historyKey 는 질문/답변 기록 저장용 키를 생성한다.
// 형식: 20q:history:{chatID}
func historyKey(chatID string) string {
	return valkeyx.BuildKey(qconfig.RedisKeyHistoryPrefix, chatID)
}

// categoryKey 는 선택된 카테고리 저장용 키를 생성한다.
// 형식: 20q:category:{chatID}
func categoryKey(chatID string) string {
	return valkeyx.BuildKey(qconfig.RedisKeyCategory, chatID)
}

// hintCountKey 는 힌트 사용 횟수 저장용 키를 생성한다.
// 형식: 20q:hints:{chatID}
func hintCountKey(chatID string) string {
	return valkeyx.BuildKey(qconfig.RedisKeyHints, chatID)
}

// playersKey 는 참여 플레이어 목록 저장용 키를 생성한다.
// 형식: 20q:players:{chatID}
func playersKey(chatID string) string {
	return valkeyx.BuildKey(qconfig.RedisKeyPlayers, chatID)
}

// wrongGuessSessionKey 는 세션 전체 오답 횟수 저장용 키를 생성한다.
// 형식: 20q:wrongGuesses:{chatID}
func wrongGuessSessionKey(chatID string) string {
	return valkeyx.BuildKey(qconfig.RedisKeyWrongGuesses, chatID)
}

// wrongGuessUserKey 는 특정 사용자의 오답 횟수 저장용 키를 생성한다.
// 형식: 20q:wrongGuesses:{chatID}:{userID}
func wrongGuessUserKey(chatID string, userID string) string {
	return valkeyx.BuildKey2(qconfig.RedisKeyWrongGuesses, chatID, userID)
}

// topicsGlobalKey 는 전체 사용된 토픽 저장용 키를 생성한다.
// 형식: 20q:topics:{chatID}
func topicsGlobalKey(chatID string) string {
	return valkeyx.BuildKey(qconfig.RedisKeyTopics, chatID)
}

// topicsCategoryKey 는 카테고리별 사용된 토픽 저장용 키를 생성한다.
// 형식: 20q:topics:{chatID}:{category}
func topicsCategoryKey(chatID string, category string) string {
	return valkeyx.BuildKey2(qconfig.RedisKeyTopics, chatID, category)
}

// voteKey 는 포기 투표 저장용 키를 생성한다.
// 형식: 20q:surrender:vote:{chatID}
func voteKey(chatID string) string {
	return valkeyx.BuildKey(qconfig.RedisKeyVotePrefix, chatID)
}

// processingKey 는 메시지 처리 중 락 키를 생성한다.
// 형식: 20q:lock:processing:{chatID}
func processingKey(chatID string) string {
	return valkeyx.BuildKeySuffix(qconfig.RedisKeyLockPrefix, "processing", chatID)
}

// lockKey 는 세션 쓰기 락 키를 생성한다.
// 형식: 20q:lock:{chatID}
func lockKey(chatID string) string {
	return valkeyx.BuildKey(qconfig.RedisKeyLockPrefix, chatID)
}

// chainSkipFlagKey 는 체인 질문 스킵 플래그 키를 생성한다.
// 형식: 20q:pending-messages:chain_skip:{chatID}:{userID}
func chainSkipFlagKey(chatID string, userID string) string {
	return valkeyx.BuildKey3(qconfig.RedisKeyPendingPrefix, "chain_skip", chatID, userID)
}
