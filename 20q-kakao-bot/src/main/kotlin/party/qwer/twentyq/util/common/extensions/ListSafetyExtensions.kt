package party.qwer.twentyq.util.common.extensions

/**
 * List 연산 확장 함수 (7개)
 *
 * 안전한 인덱스 접근, 제한된 슬라이싱, 청크 분할
 */

fun <T> List<T>.nullIfEmpty(): List<T>? = takeIf { isNotEmpty() }

fun <T> List<T>.takeUpTo(n: Int): List<T> = take(minOf(n, size))

fun <T> List<T>.dropUpTo(n: Int): List<T> = drop(minOf(n, size))

fun <T> List<T>.takeLastUpTo(n: Int): List<T> = takeLast(minOf(n, size))

fun <T> List<T>.safeGet(index: Int): T? = getOrNull(index)

fun <T> List<T>.getOrDefault(
    index: Int,
    defaultValue: T,
): T = getOrElse(index) { defaultValue }

fun <T> List<T>.chunkedSafely(size: Int): List<List<T>> =
    if (size <= 0) {
        listOf(this)
    } else {
        chunked(size)
    }
