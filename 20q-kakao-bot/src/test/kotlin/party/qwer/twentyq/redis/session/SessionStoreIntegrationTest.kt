package party.qwer.twentyq.redis.session

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.delay
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertNotEquals
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.test.annotation.DirtiesContext
import java.time.Duration

@SpringBootTest
@org.springframework.test.context.ActiveProfiles("integration")
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
class SessionStoreIntegrationTest {
    @Autowired
    private lateinit var store: SessionStore

    private val testRoomId = "test-room-${System.currentTimeMillis()}"

    @BeforeEach
    fun cleanup() =
        runTest {
            store.deleteAsync(testRoomId)
        }

    @Test
    fun `saveAsync and getAsync should store and retrieve session data`() =
        runTest {
            val sessionData = """{"state":"PLAYING","secret":"apple"}"""

            store.saveAsync(testRoomId, sessionData)
            val retrieved = store.getAsync(testRoomId)

            assertEquals(sessionData, retrieved)
        }

    @Test
    fun `getAsync should return null when session does not exist`() =
        runTest {
            val result = store.getAsync("nonexistent-room")
            assertNull(result)
        }

    @Test
    fun `saveAsync should overwrite existing session data`() =
        runTest {
            val data1 = """{"state":"IDLE"}"""
            val data2 = """{"state":"PLAYING"}"""

            store.saveAsync(testRoomId, data1)
            store.saveAsync(testRoomId, data2)

            val retrieved = store.getAsync(testRoomId)
            assertEquals(data2, retrieved)
        }

    @Test
    fun `concurrent saveAsync calls should handle last-write-wins`() =
        runTest {
            val sessions = (1..10).map { """{"iteration":$it}""" }
            sessions
                .map { data ->
                    async(Dispatchers.IO) {
                        store.saveAsync(testRoomId, data)
                    }
                }.awaitAll()

            val retrieved = store.getAsync(testRoomId)
            assertTrue(retrieved?.contains("iteration") == true, "최종 저장된 데이터가 존재해야 함")
        }

    @Test
    fun `deleteAsync should remove session data`() =
        runTest {
            store.saveAsync(testRoomId, """{"state":"PLAYING"}""")

            store.deleteAsync(testRoomId)
            val retrieved = store.getAsync(testRoomId)

            assertNull(retrieved)
        }

    @Test
    fun `setTtlAsync should return true when session exists`() =
        runTest {
            store.saveAsync(testRoomId, """{"state":"PLAYING"}""")

            val result = store.setTtlAsync(testRoomId, Duration.ofMinutes(10))

            assertTrue(result)
        }

    @Test
    fun `setTtlAsync should return false when session does not exist`() =
        runTest {
            val result = store.setTtlAsync("nonexistent-room", Duration.ofMinutes(10))

            assertFalse(result)
        }
}
