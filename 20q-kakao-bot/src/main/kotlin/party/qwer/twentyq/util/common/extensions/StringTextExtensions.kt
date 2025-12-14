package party.qwer.twentyq.util.common.extensions

import party.qwer.twentyq.util.game.constants.ValidationConstants.KAKAO_MESSAGE_MAX_LENGTH

fun String.limitLines(maxLines: Int): String =
    lines()
        .take(maxLines)
        .joinToString("\n")

/**
 * 문자열이 특정 패턴으로 시작하는지 검사 (대소문자 무시)
 */
fun String.startsWithIgnoreCase(prefix: String): Boolean = lowercase().startsWith(prefix.lowercase())

/**
 * 문자열이 특정 패턴으로 끝나는지 검사 (대소문자 무시)
 */
fun String.endsWithIgnoreCase(suffix: String): Boolean = lowercase().endsWith(suffix.lowercase())

/**
 * 안전한 substring
 * - IndexOutOfBounds 방지
 */
fun String.safeSubstring(
    startIndex: Int,
    endIndex: Int = length,
): String =
    substring(
        startIndex.coerceAtLeast(0).coerceAtMost(length),
        endIndex.coerceAtLeast(0).coerceAtMost(length),
    )

/**
 * 문자열을 최대 길이로 자르고 말줄임표 추가
 */
fun String.truncate(
    maxLength: Int,
    suffix: String = "...",
): String =
    if (length <= maxLength) {
        this
    } else {
        take(maxLength - suffix.length) + suffix
    }

/**
 * 빈 문자열을 null로 변환
 */
fun String.nullIfEmpty(): String? = takeIf { isNotEmpty() }

/**
 * 빈 문자열이면 기본값 반환
 */
fun String.ifEmpty(defaultValue: () -> String): String = if (isEmpty()) defaultValue() else this

/**
 * 긴 텍스트를 줄 단위로 분할 (카카오톡 전체보기 방지)
 *
 * 각 chunk는 maxLength를 초과하지 않으며, 줄 단위로 분할됨.
 * 단일 줄이 maxLength를 초과하면 잘림.
 */
fun String.chunkedByLines(maxLength: Int = KAKAO_MESSAGE_MAX_LENGTH): List<String> {
    val lines = split("\n")
    val chunks = mutableListOf<String>()
    var current = StringBuilder()
    var currentLength = 0

    for (raw in lines) {
        val line = if (raw.length > maxLength) raw.take(maxLength) else raw
        val separator = if (currentLength == 0) 0 else 1

        if (currentLength + separator + line.length <= maxLength) {
            if (separator == 1) current.append('\n')
            current.append(line)
            currentLength += separator + line.length
        } else {
            if (currentLength > 0) chunks.add(current.toString())
            current = StringBuilder(line)
            currentLength = line.length
        }
    }

    if (currentLength > 0) chunks.add(current.toString())
    return chunks
}
