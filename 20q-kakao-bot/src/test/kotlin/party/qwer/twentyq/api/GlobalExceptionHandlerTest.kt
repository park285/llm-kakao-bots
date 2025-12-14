package party.qwer.twentyq.api

import io.mockk.mockk
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test
import org.springframework.http.HttpStatus
import party.qwer.twentyq.util.game.GameMessageProvider
import java.util.concurrent.TimeoutException

class GlobalExceptionHandlerTest {
    private val handler = GlobalExceptionHandler(mockk<GameMessageProvider>(relaxed = true))

    @Test
    fun `java timeout maps to gateway timeout response`() {
        val response = handler.handleJavaTimeout(TimeoutException("timeout"))

        assertEquals(HttpStatus.GATEWAY_TIMEOUT, response.statusCode)
        assertEquals("AI_TIMEOUT", response.body?.error)
    }
}
