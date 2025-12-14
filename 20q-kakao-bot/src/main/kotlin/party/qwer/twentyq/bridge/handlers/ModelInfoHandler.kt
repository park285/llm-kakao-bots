package party.qwer.twentyq.bridge.handlers

import com.fasterxml.jackson.annotation.JsonIgnoreProperties
import com.fasterxml.jackson.annotation.JsonProperty
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.rest.LlmRestClient
import party.qwer.twentyq.util.game.GameMessageProvider

@Component
class ModelInfoHandler(
    private val llmRestClient: LlmRestClient,
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private val log = LoggerFactory.getLogger(ModelInfoHandler::class.java)
    }

    suspend fun handle(): String {
        val config =
            runCatching { llmRestClient.getModelConfig() }
                .onFailure { log.warn("MODEL_CONFIG_FETCH_FAILED error={}", it.message) }
                .getOrNull()
                ?: return messageProvider.get("model_info.fetch_failed")

        val mode = if (config.http2Enabled) "H2C" else "HTTP/1.1"
        val hints = config.modelHints ?: config.modelDefault
        val answer = config.modelAnswer ?: config.modelDefault
        val verify = config.modelVerify ?: config.modelDefault

        return buildString {
            appendLine(messageProvider.get("model_info.header"))
            appendLine(messageProvider.get("model_info.default", "model" to config.modelDefault))
            appendLine(messageProvider.get("model_info.hints", "model" to hints))
            appendLine(messageProvider.get("model_info.answer", "model" to answer))
            appendLine(messageProvider.get("model_info.verify", "model" to verify))
            appendLine(messageProvider.get("model_info.temperature", "value" to config.temperature))
            appendLine(messageProvider.get("model_info.max_retries", "value" to config.maxRetries))
            appendLine(messageProvider.get("model_info.timeout", "value" to config.timeoutSeconds))
            append(messageProvider.get("model_info.transport", "mode" to mode))
        }
    }
}

@JsonIgnoreProperties(ignoreUnknown = true)
data class ModelConfigResponse(
    @field:JsonProperty("model_default")
    val modelDefault: String,
    @field:JsonProperty("model_hints")
    val modelHints: String? = null,
    @field:JsonProperty("model_answer")
    val modelAnswer: String? = null,
    @field:JsonProperty("model_verify")
    val modelVerify: String? = null,
    @field:JsonProperty("temperature")
    val temperature: Double = 0.0,
    @field:JsonProperty("timeout_seconds")
    val timeoutSeconds: Int = 0,
    @field:JsonProperty("max_retries")
    val maxRetries: Int = 0,
    @field:JsonProperty("http2_enabled")
    val http2Enabled: Boolean = true,
)
