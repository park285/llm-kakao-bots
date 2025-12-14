package party.qwer.twentyq.config.properties

// Gemini 모델별 가격 ($/1M tokens)
private const val FLASH_25_INPUT_PRICE = 0.30
private const val FLASH_25_OUTPUT_PRICE = 2.50
private const val PRO_25_INPUT_PRICE = 1.25
private const val PRO_25_OUTPUT_PRICE = 10.00
private const val PRO_30_INPUT_PRICE = 2.00
private const val PRO_30_OUTPUT_PRICE = 12.00
private const val TOKENS_PER_MILLION = 1_000_000.0

enum class GeminiModel(
    val displayName: String,
    val inputPrice: Double,
    val outputPrice: Double,
) {
    // 2.5 Flash (Preview 포함) - thinking 포함
    FLASH_25("2.5 Flash", FLASH_25_INPUT_PRICE, FLASH_25_OUTPUT_PRICE),

    // 2.5 Pro - thinking 포함
    PRO_25("2.5 Pro", PRO_25_INPUT_PRICE, PRO_25_OUTPUT_PRICE),

    // 3.0 Pro Preview - thinking 포함
    PRO_30("3.0 Pro", PRO_30_INPUT_PRICE, PRO_30_OUTPUT_PRICE),
    ;

    companion object {
        const val USD_TO_KRW = 1450.0

        fun fromString(value: String?): GeminiModel {
            val normalized =
                value
                    ?.lowercase()
                    ?.replace("_", "-")
                    ?.removePrefix("google/")
                    ?: return FLASH_25

            return when {
                normalized == "pro" -> PRO_30
                normalized.contains("flash") -> FLASH_25
                normalized.contains("pro-25") -> PRO_25
                normalized.contains("2.5") && normalized.contains("pro") -> PRO_25
                normalized.contains("pro") -> PRO_30
                normalized.contains("2.5") -> FLASH_25
                else -> FLASH_25
            }
        }
    }

    // 비용 계산 (tokens -> USD)
    fun calculateCostUsd(
        inputTokens: Long,
        outputTokens: Long,
        reasoningTokens: Long = 0,
    ): Double {
        val totalOutput = outputTokens + reasoningTokens
        return (inputTokens * inputPrice + totalOutput * outputPrice) / TOKENS_PER_MILLION
    }

    // 비용 계산 (tokens -> KRW)
    fun calculateCostKrw(
        inputTokens: Long,
        outputTokens: Long,
        reasoningTokens: Long = 0,
    ): Double = calculateCostUsd(inputTokens, outputTokens, reasoningTokens) * USD_TO_KRW
}

data class Pricing(
    val model: String = "flash-25",
) {
    fun getGeminiModel(): GeminiModel = GeminiModel.fromString(model)
}
