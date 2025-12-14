package io.github.kapu.turtlesoup.utils

/**
 * 사용자 메시지 템플릿 키 상수
 */
object MessageKeys {
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Start (게임 시작)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val START_WAITING = "start.waiting"
    const val START_SCENARIO = "start.scenario"
    const val START_INSTRUCTION = "start.instruction"
    const val START_RESUME = "start.resume"
    const val START_RESUME_STATUS = "start.resume_status"
    const val START_INVALID_DIFFICULTY = "start.invalid_difficulty"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Answer (응답)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val ANSWER_RESPONSE_WITH_HISTORY = "answer.response_with_history"
    const val ANSWER_RESPONSE_SINGLE = "answer.response_single"
    const val ANSWER_HISTORY_HEADER = "answer.history_header"
    const val ANSWER_HISTORY_ITEM = "answer.history_item"
    const val ANSWER_CORRECT = "answer.correct"
    const val ANSWER_INCORRECT = "answer.incorrect"
    const val ANSWER_CLOSE_CALL = "answer.close_call"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Hint (힌트)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val HINT_WAITING = "hint.waiting"
    const val HINT_GENERATED = "hint.generated"
    const val HINT_COOLDOWN = "hint.cooldown"
    const val HINT_LIMIT_REACHED = "hint.limit_reached"
    const val HINT_EARLY_STAGE = "hint.early_stage"
    const val HINT_SECTION_USED = "hint.section_used"
    const val HINT_SECTION_NONE = "hint.section_none"
    const val HINT_ITEM = "hint.hint_item"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Problem (제시문)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val PROBLEM_DISPLAY = "problem.display"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Surrender (포기)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val SURRENDER_RESULT = "surrender.result"
    const val SURRENDER_HINT_BLOCK_HEADER = "surrender.hint_block_header"
    const val SURRENDER_HINT_ITEM = "surrender.hint_item"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Vote (포기 투표)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val VOTE_START = "vote.start"
    const val VOTE_IN_PROGRESS = "vote.in_progress"
    const val VOTE_ALREADY_ACTIVE = "vote.already_active"
    const val VOTE_NOT_FOUND = "vote.not_found"
    const val VOTE_ALREADY_VOTED = "vote.already_voted"
    const val VOTE_AGREE_PROGRESS = "vote.agree_progress"
    const val VOTE_PASSED = "vote.passed"
    const val VOTE_EXPIRED = "vote.expired"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Summary (히스토리 정리)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val SUMMARY_HEADER = "summary.header"
    const val SUMMARY_ITEM = "summary.item"
    const val SUMMARY_EMPTY = "summary.empty"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Lock / Queue (대기열)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val LOCK_REQUEST_IN_PROGRESS = "lock.request_in_progress"
    const val LOCK_REQUEST_IN_PROGRESS_WITH_HOLDER = "lock.request_in_progress_with_holder"
    const val LOCK_MESSAGE_QUEUED = "lock.message_queued"
    const val LOCK_ALREADY_QUEUED = "lock.already_queued"
    const val LOCK_QUEUE_FULL = "lock.queue_full"

    const val QUEUE_PROCESSING = "queue.processing"
    const val QUEUE_CHAINED_QUESTIONS = "queue.chained_questions"
    const val QUEUE_MESSAGE_QUEUED = "queue.message_queued"
    const val QUEUE_ALREADY_QUEUED = "queue.already_queued"
    const val QUEUE_FULL = "queue.full"
    const val QUEUE_EMPTY = "queue.empty"
    const val QUEUE_RETRY = "queue.retry"
    const val QUEUE_RETRY_DUPLICATE = "queue.retry_duplicate"
    const val QUEUE_RETRY_FAILED = "queue.retry_failed"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Processing (AI 처리 중)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val PROCESSING_THINKING = "processing.thinking"
    const val PROCESSING_WAITING = "processing.waiting"
    const val PROCESSING_GENERATING_HINT = "processing.generating_hint"
    const val PROCESSING_VALIDATING = "processing.validating"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Help (도움말)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val HELP_MESSAGE = "help.message"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Error (에러)
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val ERROR_NO_SESSION = "error.no_session"
    const val ERROR_NO_SESSION_SHORT = "error.no_session_short"
    const val ERROR_INVALID_QUESTION = "error.invalid_question"
    const val ERROR_MAX_HINTS = "error.max_hints"
    const val ERROR_INVALID_ANSWER = "error.invalid_answer"
    const val ERROR_GAME_ALREADY_STARTED = "error.game_already_started"
    const val ERROR_GAME_ALREADY_SOLVED = "error.game_already_solved"
    const val ERROR_PUZZLE_GENERATION = "error.puzzle_generation"
    const val ERROR_LOCK_FAILED = "error.lock_failed"
    const val ERROR_INTERNAL = "error.internal"
    const val ERROR_UNKNOWN_COMMAND = "error.unknown_command"

    // Access Control
    const val ERROR_ACCESS_DENIED = "error.access_denied"
    const val ERROR_USER_BLOCKED = "error.user_blocked"
    const val ERROR_CHAT_BLOCKED = "error.chat_blocked"

    // AI errors
    const val ERROR_AI_TIMEOUT = "error.ai_timeout"
    const val ERROR_AI_SAFETY_BLOCK = "error.ai_safety_block"
    const val ERROR_AI_EMPTY_RESPONSE = "error.ai_empty_response"
    const val ERROR_AI_UNAVAILABLE = "error.ai_unavailable"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // Fallback
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val FALLBACK_PUZZLE_NOT_FOUND = "fallback.puzzle_not_found"

    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    // User
    // ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    const val USER_ANONYMOUS = "user.anonymous"
    const val USER_ANONYMOUS_ID = "user.anonymous_id"
}
