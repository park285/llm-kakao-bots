package party.qwer.twentyq.util

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Test
import party.qwer.twentyq.util.common.extensions.chunkedSafely
import party.qwer.twentyq.util.common.extensions.dropUpTo
import party.qwer.twentyq.util.common.extensions.getOrDefault
import party.qwer.twentyq.util.common.extensions.nullIfEmpty
import party.qwer.twentyq.util.common.extensions.safeGet
import party.qwer.twentyq.util.common.extensions.takeLastUpTo
import party.qwer.twentyq.util.common.extensions.takeUpTo

/**
 * ListSafetyExtensions.kt 단위 테스트
 *
 * 테스트 범위:
 * - Null 관련 (nullIfEmpty for List)
 * - 안전한 추출 (takeUpTo, dropUpTo, takeLastUpTo, safeGet, getOrDefault)
 * - 청크 분할 (chunkedSafely)
 */
class CollectionExtensionsTest {
    @Test
    fun `nullIfEmpty for List - should return null when empty`() {
        // Given
        val empty = emptyList<String>()

        // When
        val result = empty.nullIfEmpty()

        // Then
        assertNull(result)
    }

    @Test
    fun `nullIfEmpty for List - should return list when not empty`() {
        // Given
        val nonEmpty = listOf("a", "b")

        // When
        val result = nonEmpty.nullIfEmpty()

        // Then
        assertEquals(listOf("a", "b"), result)
    }

    @Test
    fun `takeUpTo - should take n elements when size is larger`() {
        // Given
        val list = listOf(1, 2, 3, 4, 5)

        // When
        val result = list.takeUpTo(3)

        // Then
        assertEquals(listOf(1, 2, 3), result)
    }

    @Test
    fun `takeUpTo - should take all elements when n is larger than size`() {
        // Given
        val list = listOf(1, 2, 3)

        // When
        val result = list.takeUpTo(10)

        // Then
        assertEquals(listOf(1, 2, 3), result)
    }

    @Test
    fun `takeUpTo - should handle empty list`() {
        // Given
        val empty = emptyList<Int>()

        // When
        val result = empty.takeUpTo(5)

        // Then
        assertEquals(emptyList<Int>(), result)
    }

    @Test
    fun `dropUpTo - should drop n elements when size is larger`() {
        // Given
        val list = listOf(1, 2, 3, 4, 5)

        // When
        val result = list.dropUpTo(2)

        // Then
        assertEquals(listOf(3, 4, 5), result)
    }

    @Test
    fun `dropUpTo - should return empty list when n is larger than size`() {
        // Given
        val list = listOf(1, 2, 3)

        // When
        val result = list.dropUpTo(10)

        // Then
        assertEquals(emptyList<Int>(), result)
    }

    @Test
    fun `takeLastUpTo - should take last n elements when size is larger`() {
        // Given
        val list = listOf(1, 2, 3, 4, 5)

        // When
        val result = list.takeLastUpTo(3)

        // Then
        assertEquals(listOf(3, 4, 5), result)
    }

    @Test
    fun `takeLastUpTo - should take all elements when n is larger than size`() {
        // Given
        val list = listOf(1, 2, 3)

        // When
        val result = list.takeLastUpTo(10)

        // Then
        assertEquals(listOf(1, 2, 3), result)
    }

    @Test
    fun `safeGet - should return element when index is valid`() {
        // Given
        val list = listOf("a", "b", "c")

        // When
        val result = list.safeGet(1)

        // Then
        assertEquals("b", result)
    }

    @Test
    fun `safeGet - should return null when index is out of bounds`() {
        // Given
        val list = listOf("a", "b", "c")

        // When
        val result = list.safeGet(10)

        // Then
        assertNull(result)
    }

    @Test
    fun `getOrDefault - should return element when index is valid`() {
        // Given
        val list = listOf("a", "b", "c")

        // When
        val result = list.getOrDefault(1, "default")

        // Then
        assertEquals("b", result)
    }

    @Test
    fun `getOrDefault - should return default when index is out of bounds`() {
        // Given
        val list = listOf("a", "b", "c")

        // When
        val result = list.getOrDefault(10, "default")

        // Then
        assertEquals("default", result)
    }

    @Test
    fun `chunkedSafely - should split list into chunks`() {
        // Given
        val list = listOf(1, 2, 3, 4, 5, 6, 7)

        // When
        val result = list.chunkedSafely(3)

        // Then
        assertEquals(3, result.size)
        assertEquals(listOf(1, 2, 3), result[0])
        assertEquals(listOf(4, 5, 6), result[1])
        assertEquals(listOf(7), result[2])
    }

    @Test
    fun `chunkedSafely - should return list as single chunk when size is 0 or negative`() {
        // Given
        val list = listOf(1, 2, 3)

        // When
        val result = list.chunkedSafely(0)

        // Then
        assertEquals(1, result.size)
        assertEquals(listOf(1, 2, 3), result[0])
    }
}
