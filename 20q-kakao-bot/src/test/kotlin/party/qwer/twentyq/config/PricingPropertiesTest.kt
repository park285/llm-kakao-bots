package party.qwer.twentyq.config

import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Test
import party.qwer.twentyq.config.properties.GeminiModel

class PricingPropertiesTest {
    @Test
    fun `should map 3 pro preview names to PRO_30`() {
        assertEquals(GeminiModel.PRO_30, GeminiModel.fromString("gemini-3-pro-preview"))
        assertEquals(GeminiModel.PRO_30, GeminiModel.fromString("google/gemini-3-pro-preview"))
        assertEquals(GeminiModel.PRO_30, GeminiModel.fromString("pro"))
    }

    @Test
    fun `should map 2_5 pro names to PRO_25`() {
        assertEquals(GeminiModel.PRO_25, GeminiModel.fromString("gemini-2.5-pro"))
        assertEquals(GeminiModel.PRO_25, GeminiModel.fromString("pro-25"))
    }

    @Test
    fun `should map flash names to FLASH_25`() {
        assertEquals(GeminiModel.FLASH_25, GeminiModel.fromString("gemini-2.5-flash"))
        assertEquals(GeminiModel.FLASH_25, GeminiModel.fromString("flash-25"))
    }
}
