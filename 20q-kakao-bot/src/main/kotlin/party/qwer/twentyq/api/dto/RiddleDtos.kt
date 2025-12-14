package party.qwer.twentyq.api.dto

data class RiddleCreateRequest(
    val category: String? = null,
)

data class RiddleCreateResponse(
    val message: String,
)

data class RiddleHintsRequest(
    val count: Int = 2,
)

data class RiddleHintsResponse(
    val hints: List<String>,
)

data class RiddleAnswerRequest(
    val question: String,
)

data class RiddleAnswerResponse(
    val scale: String,
)

data class QuestionHistory(
    val questionNumber: Int,
    val question: String,
    val answer: String,
    val isChain: Boolean = false,
    val thoughtSignature: String? = null,
    val userId: String? = null,
)

data class HintHistory(
    val hintNumber: Int,
    val content: String,
)

data class RiddleStatusResponse(
    val questionCount: Int,
    val questions: List<QuestionHistory>,
    val hints: List<HintHistory>,
    val hintCount: Int,
    val maxHints: Int,
    val selectedCategory: String? = null,
)
