package party.qwer.twentyq.rest

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component

/**
 * LLM 서버 가용성 확인 게이트
 * - healthEnabled=false 인 경우에는 가용하다고 간주
 */
@Component
class LlmAvailabilityGuard(
    private val properties: LlmRestProperties,
    private val restClient: LlmRestClient,
) {
    companion object {
        private val log = LoggerFactory.getLogger(LlmAvailabilityGuard::class.java)
    }

    suspend fun isAvailable(): Boolean {
        if (!properties.healthEnabled) {
            return true
        }

        return runCatching {
            restClient.checkHealth(properties.healthPath) is HealthCheckResult.Healthy
        }.onFailure { ex ->
            log.warn("LLM_AVAILABILITY_CHECK_FAILED error={}", ex.message)
        }.getOrDefault(false)
    }
}
