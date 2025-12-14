package io.github.kapu.turtlesoup.security

import io.github.kapu.turtlesoup.rest.LlmRestClient
import io.github.kapu.turtlesoup.utils.InputInjectionException
import io.github.kapu.turtlesoup.utils.MalformedInputException
import io.github.oshai.kotlinlogging.KotlinLogging

private val logger = KotlinLogging.logger {}

private const val LOG_TEXT_LIMIT = 100

/** MCP REST API 기반 인젝션 검사 (항상 활성화) */
class McpInjectionGuard(
    private val restClient: LlmRestClient,
) {
    /** 악성 입력 여부 검사 */
    suspend fun isMalicious(input: String): Boolean {
        val result = restClient.guardIsMalicious(input)
        if (result) {
            logger.warn { "guard_blocked input='${input.take(LOG_TEXT_LIMIT)}'" }
        }
        return result
    }

    /** 검증 및 sanitize (실패 시 예외) */
    suspend fun validateOrThrow(input: String): String {
        if (input.isBlank()) {
            throw MalformedInputException("Empty input")
        }

        if (isMalicious(input)) {
            logger.warn { "injection_blocked input='${input.take(LOG_TEXT_LIMIT)}'" }
            throw InputInjectionException("Potentially malicious input detected", null)
        }

        return sanitize(input)
    }

    private fun sanitize(input: String): String =
        input
            .trim()
            .replace(Regex("\\s+"), " ")
}
