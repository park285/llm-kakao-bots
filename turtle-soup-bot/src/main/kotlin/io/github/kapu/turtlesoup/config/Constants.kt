package io.github.kapu.turtlesoup.config

object GameConstants {
    const val MAX_HINTS = 3
    const val SESSION_TTL_MINUTES = 1440
    const val EARLY_STAGE_DIVISOR = 3
}

object ValidationConstants {
    const val MIN_QUESTION_LENGTH = 2
    const val MAX_QUESTION_LENGTH = 200
    const val MIN_ANSWER_LENGTH = 1
    const val MAX_ANSWER_LENGTH = 500
    const val KAKAO_MESSAGE_MAX_LENGTH = 500
}

object RedisKeys {
    private const val APP_PREFIX = "turtle"
    const val SESSION = "$APP_PREFIX:session"
    const val CHAT = "$APP_PREFIX:chat"
    const val LOCK = "$APP_PREFIX:lock"
    const val SURRENDER_VOTE = "$APP_PREFIX:vote"
    const val PLAYERS = "$APP_PREFIX:players"
    const val PENDING_MESSAGES = "$APP_PREFIX:pending"
    const val PROCESSING = "$APP_PREFIX:processing"
    const val PUZZLE_GLOBAL = "$APP_PREFIX:puzzle:global"
    const val PUZZLE_CHAT = "$APP_PREFIX:puzzle:chat"
}

object RedisConstants {
    const val SESSION_TTL_SECONDS = 86400L // 24시간
    const val LOCK_TTL_SECONDS = 120L // AI 응답 대기 시간 고려
    const val LOCK_TIMEOUT_SECONDS = 60L // 락 획득 대기 시간
    const val VOTE_TTL_SECONDS = 120L // 2분
    const val PROCESSING_TTL_SECONDS = 120L // 처리 중 상태 TTL
    const val QUEUE_TTL_SECONDS = 300L // 5분
    const val MAX_QUEUE_SIZE = 5
    const val STALE_THRESHOLD_MS = 3600_000L // 1시간
    const val CLEANUP_TIMEOUT_MS = 10_000L
}

object ChatMemoryConstants {
    const val MAX_MESSAGES = 100
    const val TTL_SECONDS = 1800L // 30분
}

object RedissonConnectionConstants {
    const val CONNECTION_POOL_SIZE = 64
    const val CONNECTION_MINIMUM_IDLE_SIZE = 10
    const val IDLE_CONNECTION_TIMEOUT_MS = 10000
    const val CONNECT_TIMEOUT_MS = 10000
    const val TIMEOUT_MS = 3000
}

object PuzzleConstants {
    const val MIN_DIFFICULTY = 1
    const val MAX_DIFFICULTY = 5
    const val DEFAULT_DIFFICULTY = 3
}

object PuzzleDedupConstants {
    const val MAX_GENERATION_RETRIES = 3
    const val GLOBAL_TTL_SECONDS = 7 * 24 * 3600L // 7일
    const val CHAT_TTL_SECONDS = 3 * 24 * 3600L // 3일
}

object TimeConstants {
    const val MILLIS_PER_SECOND = 1000L
    const val SECONDS_PER_MINUTE = 60
    const val MINUTES_PER_HOUR = 60
    const val PERCENT_MULTIPLIER = 100
    const val PERCENT_MAX = 100
    const val AI_TIMEOUT_SECONDS = 15L
    const val AI_TIMEOUT_MILLIS = AI_TIMEOUT_SECONDS * MILLIS_PER_SECOND
}

object StreamKeys {
    const val INBOUND = "kakao:turtle-soup"
    const val OUTBOUND = "kakao:bot:reply"

    // Inbound stream fields
    const val FIELD_CHAT_ID = "chatId"
    const val FIELD_USER_ID = "userId"
    const val FIELD_CONTENT = "content"
    const val FIELD_THREAD_ID = "threadId"
    const val FIELD_SENDER = "sender"

    // Outbound stream fields
    const val FIELD_TEXT = "text"
    const val FIELD_TYPE = "type"

    // Message types
    const val TYPE_WAITING = "waiting"
    const val TYPE_FINAL = "final"
    const val TYPE_ERROR = "error"
}

object MQConstants {
    const val BATCH_SIZE = 5
    const val SEMAPHORE_PERMITS = 5
    const val READ_TIMEOUT_MS = 5000L
    const val AUTOCLAIM_IDLE_MINUTES = 5L
    const val AUTOCLAIM_COUNT = 10
    const val MAX_QUEUE_PROCESS_ITERATIONS = 10

    // 스트림 길이 제한 (대략적 트림)
    const val STREAM_MAX_LEN = 1000L
}

/** 일시적 API 에러 판별용 패턴 */
object ApiErrorPatterns {
    val TRANSIENT_ERROR_PATTERNS =
        listOf(
            "503",
            "429",
            "Service Unavailable",
            "Too Many Requests",
            "RESOURCE_EXHAUSTED",
        )
}

/** API 에러 코드 */
object ApiErrorCodes {
    const val GAME_ALREADY_STARTED = "GAME_ALREADY_STARTED"
    const val SESSION_NOT_FOUND = "SESSION_NOT_FOUND"
    const val GAME_ERROR = "GAME_ERROR"
    const val MAX_HINTS_REACHED = "MAX_HINTS_REACHED"
    const val INVALID_REQUEST = "INVALID_REQUEST"
    const val INTERNAL_ERROR = "INTERNAL_ERROR"
}
