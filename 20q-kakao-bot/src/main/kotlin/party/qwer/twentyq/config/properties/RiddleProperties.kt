package party.qwer.twentyq.config.properties

/**
 * 주제(토픽) 디렉토리 설정
 *
 * @property directory 토픽 파일이 위치한 디렉토리
 */
data class Topics(
    val directory: String = "classpath:topics/",
)

/** 20 Questions 게임 규칙 설정 */
data class RiddleConfig(
    val game: GameConfig = GameConfig(),
    val validation: Validation = Validation(),
    val normalize: Normalize = Normalize(),
) {
    /** 게임 기본 설정 */
    data class GameConfig(
        val defaultHintLimit: Int = 2,
        val bonusHintLimit: Int = 2,
        val bonusHintQuestionThreshold: Int = 20,
        val recentTopicsLimit: Int = 20,
    )

    /** 답변 검증 설정 */
    data class Validation(
        val enabled: Boolean = false,
        val maxRetries: Int = 1,
    )

    /** 답변 정규화 설정 (mode: always | suspicious-only | never) */
    data class Normalize(
        val mode: String = "always",
    )
}
