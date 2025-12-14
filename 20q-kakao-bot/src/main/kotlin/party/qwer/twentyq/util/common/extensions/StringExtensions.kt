package party.qwer.twentyq.util.common.extensions

import party.qwer.twentyq.util.game.constants.ValidationConstants.MASK_PREFIX_LENGTH
import party.qwer.twentyq.util.game.constants.ValidationConstants.MASK_SUFFIX_LENGTH
import party.qwer.twentyq.util.game.constants.ValidationConstants.MAX_QUESTION_LENGTH
import party.qwer.twentyq.util.game.constants.ValidationConstants.MIN_MASK_REVEAL_LENGTH
import party.qwer.twentyq.util.game.constants.ValidationConstants.MIN_QUESTION_LENGTH
import party.qwer.twentyq.util.game.constants.ValidationConstants.TOKEN_MASK_MIN_LENGTH
import party.qwer.twentyq.util.game.constants.ValidationConstants.TOKEN_PREFIX_LENGTH
import party.qwer.twentyq.util.game.constants.ValidationConstants.TOKEN_SUFFIX_LENGTH

fun String.isValidQuestion(): Boolean =
    length in MIN_QUESTION_LENGTH..MAX_QUESTION_LENGTH &&
        matches(Regex("[가-힣a-zA-Z0-9\\s?!.]+"))

/**
 * 정답 제출 명령어인지 확인
 */
fun String.isAnswerCommand(): Boolean = trim().startsWith("정답", ignoreCase = true)

fun String.normalizeForComparison(): String =
    lowercase()
        .replace(Regex("[^가-힣a-z0-9]"), "")
        .trim()

fun String.maskSensitive(): String =
    when {
        isEmpty() -> ""
        length <= MIN_MASK_REVEAL_LENGTH -> "*".repeat(length)
        else ->
            take(MASK_PREFIX_LENGTH) +
                "*".repeat((length - MIN_MASK_REVEAL_LENGTH).coerceAtLeast(1)) +
                takeLast(MASK_SUFFIX_LENGTH)
    }

fun String.maskToken(): String =
    when {
        isEmpty() -> ""
        length <= TOKEN_MASK_MIN_LENGTH -> "*".repeat(length)
        else -> take(TOKEN_PREFIX_LENGTH) + "..." + takeLast(TOKEN_SUFFIX_LENGTH)
    }

fun String.isValidCategory(): Boolean = this in listOf("인물", "사물", "동물", "장소", "음식", "추상")

fun String.toCategoryIcon(): String =
    when (this) {
        "인물" -> "👤"
        "사물" -> "📦"
        "동물" -> "🐾"
        "장소" -> "📍"
        "음식" -> "🍽️"
        "추상" -> "💭"
        else -> "❓"
    }

fun String.parseYesNo(): Boolean? =
    when (this.trim().lowercase()) {
        "네", "예", "yes", "y", "맞아", "맞습니다" -> true
        "아니", "아니오", "no", "n", "아니야", "틀려" -> false
        else -> null
    }

fun String.fillTemplate(params: Map<String, Any>): String =
    params.entries.fold(this) { text, (key, value) ->
        text.replace($$"${$$key}", value.toString())
    }
