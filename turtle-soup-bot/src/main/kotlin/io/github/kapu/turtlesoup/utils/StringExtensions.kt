package io.github.kapu.turtlesoup.utils

import io.github.kapu.turtlesoup.config.ValidationConstants

fun String.isValidQuestion(): Boolean =
    length in ValidationConstants.MIN_QUESTION_LENGTH..ValidationConstants.MAX_QUESTION_LENGTH &&
        isNotBlank()

fun String.isValidAnswer(): Boolean =
    length in ValidationConstants.MIN_ANSWER_LENGTH..ValidationConstants.MAX_ANSWER_LENGTH &&
        isNotBlank()

/** 안전한 에러 메시지 추출 (null -> 빈 문자열) */
val Throwable.safeMessage: String
    get() = message ?: cause?.message ?: ""

/** 긴 텍스트를 줄 단위로 분할 (카카오톡 전체보기 방지) */
fun String.chunkedByLines(maxLength: Int = ValidationConstants.KAKAO_MESSAGE_MAX_LENGTH): List<String> {
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
