package party.qwer.twentyq.util.game.constants

/**
 * 20Q 게임 규칙 상수
 *
 * 게임 진행, 단계 구분, 임계값 등 핵심 게임 로직에서 사용
 */
object GameConstants {
    const val MIN_HINT_REQUEST = 1
    const val MAX_HINT_REQUEST = 10

    // 게임 단계 구분
    const val EARLY_STAGE_THRESHOLD = 3 // questionCount <= 3
    const val MID_STAGE_START = 4 // questionCount in 4..10
    const val MID_STAGE_END = 10
    const val LATE_STAGE_THRESHOLD = 11 // questionCount >= 11
    const val FINAL_STAGE_OFFSET = 2 // maxQuestions - 2

    // 메시지 처리 대기 시간 (초)
    const val WAITING_MESSAGE_DELAY_SECONDS = 4L
}
