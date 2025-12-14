package io.github.kapu.turtlesoup.utils

private val CODE_BLOCK_REGEX = Regex("""^```(?:json)?\s*([\s\S]*?)```$""")

/** AI 응답에서 마크다운 코드블록 래퍼 제거 */
fun String.cleanJsonResponse(): String {
    val trimmed = trim()
    return CODE_BLOCK_REGEX.find(trimmed)?.groupValues?.get(1)?.trim() ?: trimmed
}
