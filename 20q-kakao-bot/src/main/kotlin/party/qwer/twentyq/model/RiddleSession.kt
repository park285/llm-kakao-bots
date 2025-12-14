package party.qwer.twentyq.model

data class RiddleSession(
    val chatId: String,
    val userId: String,
    val secret: RiddleSecret,
    val questionCount: Int = 0,
    val hintCount: Int = 0,
    val selectedCategory: String? = null,
)
