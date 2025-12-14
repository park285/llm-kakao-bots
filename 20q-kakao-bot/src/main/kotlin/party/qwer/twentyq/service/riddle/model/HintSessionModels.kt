package party.qwer.twentyq.service.riddle.model

import party.qwer.twentyq.model.SecretForHint

data class HintSessionContext(
    val chatId: String,
    val secretForHint: SecretForHint,
    val secretTarget: String,
    val currentHintCount: Int,
    val maxHints: Int,
    val questionCount: Int,
    val selectedCategory: String,
)

data class HintModelResponse(
    val text: String,
    val thoughtSignature: String?,
)
