package party.qwer.twentyq.model

/** 사용량 조회 기간 */
enum class UsagePeriod {
    TODAY,
    WEEKLY,
    MONTHLY,
}

/**
 * 사용자 명령어 타입
 *
 * CommandParser에서 파싱된 명령어를 나타내는 sealed interface
 */
sealed interface Command {
    /**
     * 게임 시작 명령어
     *
     * @property categories 선택적 카테고리 목록 (공백 구분, 랜덤 선택)
     */
    data class Start(
        val categories: List<String>? = null,
    ) : Command

    /**
     * 도움말 요청 명령어
     */
    data object Help : Command

    /**
     * 힌트 요청 명령어
     *
     * @property count 요청할 힌트 개수 (기본 2개)
     */
    data class Hints(
        val count: Int = 2,
    ) : Command

    /**
     * 질문 명령어
     *
     * @property question 질문 내용
     */
    data class Ask(
        val question: String,
    ) : Command

    /**
     * 체인 질문 명령어
     *
     * 쉼표로 구분된 여러 질문을 순차적으로 실행
     *
     * @property questions 순차 실행할 질문 목록
     * @property condition 실행 조건 (기본값: ALWAYS)
     */
    data class ChainedQuestion(
        val questions: List<String>,
        val condition: ChainCondition = ChainCondition.ALWAYS,
    ) : Command

    /**
     * 항복 명령어
     */
    data object Surrender : Command

    /**
     * 항복 동의 명령어
     */
    data object Agree : Command

    /**
     * 항복 거부 명령어
     */
    data object Reject : Command

    /**
     * 관리자: 게임 강제 종료
     */
    data object AdminForceEnd : Command

    /**
     * 관리자: 전체 세션 삭제
     */
    data object AdminClearAll : Command

    /**
     * 관리자: 캐시 갱신
     */
    data object AdminRefreshCache : Command

    /**
     * 관리자: 모든 봇 재시작
     */
    data object AdminRestartAll : Command

    /**
     * 헬스체크 명령어
     */
    data object HealthCheck : Command

    /**
     * 상태 확인 명령어
     */
    data object Status : Command

    /**
     * 관리자: 토큰 사용량 조회
     *
     * @property period 조회 기간 (기본: TODAY)
     * @property modelOverride 비용 계산용 모델 (null이면 기본값 사용)
     */
    data class AdminUsage(
        val period: UsagePeriod = UsagePeriod.TODAY,
        val modelOverride: String? = null,
    ) : Command

    /** 모델/전송 설정 조회 */
    data object ModelInfo : Command

    /**
     * 사용자 전적 조회 명령어
     *
     * @property targetNickname 조회 대상 닉네임 (null이면 본인 전적)
     * @property roomPeriod 방 전적 조회 기간 (null이 아니면 방 전적 조회)
     */
    data class UserStats(
        val targetNickname: String? = null,
        val roomPeriod: String? = null,
    ) : Command
}

/**
 * Command가 Write Lock을 필요로 하는지 판단
 *
 * @return true: WRITE Lock 필요 (세션 수정), false: READ Lock 가능 (조회만)
 */
fun Command.requiresWriteLock(): Boolean =
    when (this) {
        is Command.Status, is Command.HealthCheck, is Command.UserStats, is Command.AdminUsage -> false
        else -> true // WRITE: 세션 수정
    }

fun Command.requiresLlm(): Boolean =
    when (this) {
        is Command.Help,
        is Command.Start,
        is Command.Hints,
        is Command.Ask,
        is Command.ChainedQuestion,
        -> true
        else -> false
    }
