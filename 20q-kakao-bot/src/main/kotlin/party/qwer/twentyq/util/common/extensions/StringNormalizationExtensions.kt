package party.qwer.twentyq.util.common.extensions

fun String.normalizeWhitespace(): String = trim().replace(Regex("\\s+"), " ")

/**
 * LLM enum 응답 정규화 (따옴표 제거)
 * - JSON string 형태 ("ACCEPT") → 순수 문자열 (ACCEPT)
 */
fun String.normalizeEnumResponse(): String =
    trim()
        .replace("\"", "") // 일반 큰따옴표
        .replace("'", "") // 작은따옴표
        .replace("\u2018", "") // 왼쪽 작은따옴표 '
        .replace("\u2019", "") // 오른쪽 작은따옴표 '
        .replace("\u201C", "") // 왼쪽 큰따옴표 "
        .replace("\u201D", "") // 오른쪽 큰따옴표 "
        .trim()

fun String.smartTrim(): String =
    lines()
        .map { it.trim() }
        .filter { it.isNotBlank() }
        .joinToString("\n")

fun String.normalizeKakaoText(): String =
    this
        .replace(Regex("[\u200B-\u200D\uFEFF]"), "")
        .replace(Regex("[\uD83C-\uDBFF\uDC00-\uDFFF]+"), "")
        .normalizeWhitespace()
