package party.qwer.twentyq.util.common.extensions

fun String.toKoreanAnswer(): String =
    when (parseYesNo()) {
        true -> "네"
        false -> "아니오"
        else -> this
    }
