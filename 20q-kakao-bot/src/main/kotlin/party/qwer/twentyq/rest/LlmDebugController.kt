package party.qwer.twentyq.rest

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/internal/llm")
class LlmDebugController(
    private val properties: LlmRestProperties,
) {
    @GetMapping("/config")
    fun getConfig(): LlmConfigResponse =
        LlmConfigResponse(
            baseUrl = properties.baseUrl,
            http2Enabled = properties.http2Enabled,
            timeoutSeconds = properties.timeoutSeconds,
            connectTimeoutSeconds = properties.connectTimeoutSeconds,
            healthEnabled = properties.healthEnabled,
            healthPath = properties.healthPath,
            healthIntervalMillis = properties.healthIntervalMillis,
            healthFailureThreshold = properties.healthFailureThreshold,
            healthRestartCommand = properties.healthRestartCommand,
        )
}

data class LlmConfigResponse(
    val baseUrl: String,
    val http2Enabled: Boolean,
    val timeoutSeconds: Long,
    val connectTimeoutSeconds: Long,
    val healthEnabled: Boolean,
    val healthPath: String,
    val healthIntervalMillis: Long,
    val healthFailureThreshold: Int,
    val healthRestartCommand: String,
)
