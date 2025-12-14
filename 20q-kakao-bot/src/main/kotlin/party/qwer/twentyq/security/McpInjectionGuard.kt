package party.qwer.twentyq.security

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.rest.GuardRestClient
import party.qwer.twentyq.util.logging.LoggingConstants

/** REST API 기반 인젝션 검사 */
@Component
class McpInjectionGuard(
    private val guardClient: GuardRestClient,
    private val appProperties: AppProperties,
) {
    companion object {
        private val log = LoggerFactory.getLogger(McpInjectionGuard::class.java)
    }

    /** 악성 입력 여부 검사 */
    suspend fun isMalicious(input: String): Boolean {
        if (!appProperties.security.injection.enabled) {
            return false
        }

        val result = guardClient.isMalicious(input)
        if (result) {
            log.warn(
                "GUARD_BLOCKED input='{}'",
                input.take(LoggingConstants.LOG_TEXT_MEDIUM),
            )
        }
        return result
    }
}
