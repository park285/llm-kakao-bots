package party.qwer.twentyq.util

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import party.qwer.twentyq.model.RiddleSecret
import party.qwer.twentyq.model.RiddleSession
import party.qwer.twentyq.util.game.extensions.canUseHint
import party.qwer.twentyq.util.game.extensions.hasCategory
import party.qwer.twentyq.util.game.extensions.hasNoCategory
import party.qwer.twentyq.util.game.extensions.isActive
import party.qwer.twentyq.util.game.extensions.isCategory
import party.qwer.twentyq.util.game.extensions.isExpired
import party.qwer.twentyq.util.game.extensions.progress
import party.qwer.twentyq.util.game.extensions.progressPercent
import party.qwer.twentyq.util.game.extensions.remainingHints
import party.qwer.twentyq.util.game.extensions.remainingQuestions

class SessionExtensionsTest {
    private lateinit var mockSecret: RiddleSecret
    private lateinit var baseSession: RiddleSession

    @BeforeEach
    fun setup() {
        mockSecret =
            RiddleSecret(
                target = "고양이",
                category = "동물",
                intro = "네 발로 걷는 애완동물",
            )

        baseSession =
            RiddleSession(
                chatId = "test-chat",
                userId = "test-user",
                secret = mockSecret,
                questionCount = 0,
                hintCount = 0,
                selectedCategory = null,
            )
    }

    @Test
    fun `isExpired - should return true when question count reaches max`() {
        // Given
        val session = baseSession.copy(questionCount = 20)

        // When/Then
        assertTrue(session.isExpired(20))
    }

    @Test
    fun `isExpired - should return false when question count below max`() {
        // Given
        val session = baseSession.copy(questionCount = 19)

        // When/Then
        assertFalse(session.isExpired(20))
    }

    @Test
    fun `isActive - should return true when not expired`() {
        // Given
        val session = baseSession.copy(questionCount = 10)

        // When/Then
        assertTrue(session.isActive(20))
    }

    @Test
    fun `isActive - should return false when expired`() {
        // Given
        val session = baseSession.copy(questionCount = 20)

        // When/Then
        assertFalse(session.isActive(20))
    }

    @Test
    fun `remainingQuestions - should return correct count`() {
        // Given
        val session = baseSession.copy(questionCount = 15)

        // When
        val result = session.remainingQuestions(20)

        // Then
        assertEquals(5, result)
    }

    @Test
    fun `remainingQuestions - should return 0 when expired`() {
        // Given
        val session = baseSession.copy(questionCount = 25)

        // When
        val result = session.remainingQuestions(20)

        // Then
        assertEquals(0, result)
    }

    @Test
    fun `canUseHint - should return true when under limit`() {
        // Given
        val session = baseSession.copy(hintCount = 2)

        // When/Then
        assertTrue(session.canUseHint(maxHints = 3))
    }

    @Test
    fun `canUseHint - should return false when at limit`() {
        // Given
        val session = baseSession.copy(hintCount = 3)

        // When/Then
        assertFalse(session.canUseHint(maxHints = 3))
    }

    @Test
    fun `remainingHints - should return correct count`() {
        // Given
        val session = baseSession.copy(hintCount = 1)

        // When
        val result = session.remainingHints(3)

        // Then
        assertEquals(2, result)
    }

    @Test
    fun `remainingHints - should return 0 when at limit`() {
        // Given
        val session = baseSession.copy(hintCount = 3)

        // When
        val result = session.remainingHints(3)

        // Then
        assertEquals(0, result)
    }

    @Test
    fun `progress - should calculate correct ratio`() {
        // Given
        val session = baseSession.copy(questionCount = 10)

        // When
        val result = session.progress(20)

        // Then
        assertEquals(0.5, result, 0.001)
    }

    @Test
    fun `progress - should return 0 for max questions of 0`() {
        // Given
        val session = baseSession.copy(questionCount = 10)

        // When
        val result = session.progress(0)

        // Then
        assertEquals(0.0, result, 0.001)
    }

    @Test
    fun `progressPercent - should calculate correct percentage`() {
        // Given
        val session = baseSession.copy(questionCount = 15)

        // When
        val result = session.progressPercent(20)

        // Then
        assertEquals(75, result)
    }

    @Test
    fun `hasCategory - should return true when category is set`() {
        // Given
        val session = baseSession.copy(selectedCategory = "동물")

        // When/Then
        assertTrue(session.hasCategory())
    }

    @Test
    fun `hasCategory - should return false when category is null`() {
        // Given
        val session = baseSession.copy(selectedCategory = null)

        // When/Then
        assertFalse(session.hasCategory())
    }

    @Test
    fun `hasNoCategory - should return true when category is null`() {
        // Given
        val session = baseSession.copy(selectedCategory = null)

        // When/Then
        assertTrue(session.hasNoCategory())
    }

    @Test
    fun `hasNoCategory - should return false when category is set`() {
        // Given
        val session = baseSession.copy(selectedCategory = "동물")

        // When/Then
        assertFalse(session.hasNoCategory())
    }

    @Test
    fun `isCategory - should return true when matching`() {
        // Given
        val session = baseSession.copy(selectedCategory = "동물")

        // When/Then
        assertTrue(session.isCategory("동물"))
        assertFalse(session.isCategory("사물"))
    }
}
