package party.qwer.twentyq.service

import java.io.Serializable

data class NormalizeResponse(
    val normalized: String,
) : Serializable {
    companion object {
        private const val serialVersionUID = 1L

        val RESPONSE_SCHEMA: Map<String, Any> =
            mapOf(
                "type" to "object",
                "properties" to
                    mapOf(
                        "normalized" to mapOf("type" to "string"),
                    ),
                "required" to listOf("normalized"),
            )
    }
}
