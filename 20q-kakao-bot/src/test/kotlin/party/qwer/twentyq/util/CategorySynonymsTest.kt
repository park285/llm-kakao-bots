package party.qwer.twentyq.util

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import party.qwer.twentyq.util.game.CategorySynonyms

class CategorySynonymsTest {
    @Test
    fun `should return forbidden words for known category`() {
        val words = CategorySynonyms.getForbiddenWords("영화")

        assertTrue(words.contains("영화"))
        assertTrue(words.contains("매체"))
        assertTrue(words.contains("미디어"))
        assertTrue(words.contains("콘텐츠"))
    }

    @Test
    fun `should return forbidden words for food category`() {
        val words = CategorySynonyms.getForbiddenWords("음식")

        assertTrue(words.contains("음식"))
        assertTrue(words.contains("먹거리"))
        assertTrue(words.contains("식품"))
    }

    @Test
    fun `should return empty set for unknown category`() {
        val words = CategorySynonyms.getForbiddenWords("unknown")

        assertEquals(emptySet<String>(), words)
    }

    @Test
    fun `should format forbidden words as comma-separated string`() {
        val formatted = CategorySynonyms.toForbiddenWordsString("영화")

        assertTrue(formatted.contains("영화"))
        assertTrue(formatted.contains("매체"))
        assertTrue(formatted.contains(","))
    }

    @Test
    fun `should return placeholder for unknown category in formatted string`() {
        val formatted = CategorySynonyms.toForbiddenWordsString("unknown")

        assertEquals("(no forbidden words)", formatted)
    }

    @Test
    fun `should include various transportation terms for 교통수단 category`() {
        val words = CategorySynonyms.getForbiddenWords("교통수단")

        assertTrue(words.contains("교통수단"))
        assertTrue(words.contains("이동수단"))
        assertTrue(words.contains("운송수단"))
    }

    @Test
    fun `should include animal terms for 동물 category`() {
        val words = CategorySynonyms.getForbiddenWords("동물")

        assertTrue(words.contains("동물"))
        assertTrue(words.contains("생물"))
        assertTrue(words.contains("생명체"))
    }
}
