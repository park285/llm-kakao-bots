package party.qwer.twentyq.redis

object RedisKeys {
    private const val APP_PREFIX = "20q"

    // 세션 키
    const val SESSION = "$APP_PREFIX:riddle:session"
    const val HISTORY = "$APP_PREFIX:history"
    const val CATEGORY = "$APP_PREFIX:category"
    const val LLM_SESSION = "$APP_PREFIX:llmSession"

    // 추적 키
    const val HINTS = "$APP_PREFIX:hints"
    const val PLAYERS = "$APP_PREFIX:players"
    const val WRONG_GUESSES = "$APP_PREFIX:wrongGuesses"
    const val TOPICS = "$APP_PREFIX:topics"

    // 투표 키
    const val SURRENDER_VOTE = "$APP_PREFIX:surrender:vote"
    const val CANDIDATE_COUNT = "$APP_PREFIX:candidateCount"

    // 시스템 키
    const val LOCK = "$APP_PREFIX:lock"
    const val PENDING_MESSAGES = "$APP_PREFIX:pending-messages"
    const val STATS = "$APP_PREFIX:stats"
}
