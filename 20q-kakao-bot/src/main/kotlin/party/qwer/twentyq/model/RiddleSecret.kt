package party.qwer.twentyq.model

data class RiddleSecret(
    val target: String,
    val category: String,
    val intro: String,
    val description: String? = null,
)
