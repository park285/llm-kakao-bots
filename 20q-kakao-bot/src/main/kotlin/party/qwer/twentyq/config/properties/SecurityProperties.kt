package party.qwer.twentyq.config.properties

/**
 * 보안 정책 설정
 *
 * @property injection 인젝션 공격 탐지 설정
 * @property metaQuestionLlmEnabled 메타 질문 LLM 검증 활성화
 */
data class Security(
    val injection: Injection = Injection(),
    val metaQuestionLlmEnabled: Boolean = true,
) {
    /**
     * 인젝션 공격 탐지 설정
     *
     * @property enabled 탐지 활성화 여부
     * @property threshold 탐지 임계값 (0.0 ~ 1.0)
     * @property rulepacks 룰팩 파일 경로 목록
     */
    data class Injection(
        val enabled: Boolean = true,
        val threshold: Double = 0.7,
        val rulepacks: List<String> =
            listOf(
                "security/rulepacks/injection-en.yml",
                "security/rulepacks/injection-ko.yml",
            ),
    )
}

/**
 * 접근 제어 설정
 *
 * @property enabled 접근 제어 활성화 여부
 * @property allowedChatIds 허용된 채팅방 ID 목록
 * @property blockedChatIds 차단된 채팅방 ID 목록 (ACL enabled 시만 적용)
 * @property blockedUserIds 차단된 사용자 ID 목록 (ACL 무관 글로벌 차단)
 * @property passthrough 우회 모드 (테스트용)
 */
data class Access(
    val enabled: Boolean = false,
    val allowedChatIds: List<String> = emptyList(),
    val blockedChatIds: List<String> = emptyList(),
    val blockedUserIds: List<String> = emptyList(),
    val passthrough: Boolean = false,
)

/**
 * 명령어 설정
 *
 * @property prefix 명령어 접두사 (예: "/20q")
 */
data class Commands(
    val prefix: String = "/20q",
)

/**
 * 관리자 설정
 *
 * @property userIds 관리자 사용자 ID 목록
 */
data class Admin(
    val userIds: List<String> = emptyList(),
)
