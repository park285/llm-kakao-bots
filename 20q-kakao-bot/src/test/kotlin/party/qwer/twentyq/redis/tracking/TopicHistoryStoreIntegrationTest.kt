package party.qwer.twentyq.redis.tracking

import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.test.annotation.DirtiesContext
import org.springframework.test.context.ActiveProfiles

@SpringBootTest
@ActiveProfiles("integration")
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
class TopicHistoryStoreIntegrationTest {
    @Autowired
    private lateinit var store: TopicHistoryStore

    private val testRoomId = "test-room-${System.currentTimeMillis()}"

    @BeforeEach
    fun cleanup() =
        runTest {
            store.clearAllAsync(testRoomId)
        }

    @Test
    fun `addAsync should record both global and category history`() =
        runTest {
            store.addAsync(testRoomId, "food", "apple")

            val global = store.getRecentAsync(testRoomId)
            val category = store.getRecentAsync(testRoomId, "food")

            assertEquals(listOf("apple"), global)
            assertEquals(listOf("apple"), category)
        }

    @Test
    fun `addAsync should keep order across multiple categories`() =
        runTest {
            store.addAsync(testRoomId, "food", "apple")
            store.addAsync(testRoomId, "movie", "matrix")
            store.addAsync(testRoomId, "food", "banana")

            val global = store.getRecentAsync(testRoomId)
            val food = store.getRecentAsync(testRoomId, "food")
            val movie = store.getRecentAsync(testRoomId, "movie")

            assertEquals(listOf("banana", "matrix", "apple"), global)
            assertEquals(listOf("banana", "apple"), food)
            assertEquals(listOf("matrix"), movie)
        }

    @Test
    fun `getBannedTopics should merge distinct global and category history`() =
        runTest {
            store.addAsync(testRoomId, "food", "apple")
            store.addAsync(testRoomId, "movie", "matrix")
            store.addAsync(testRoomId, "food", "banana")

            val banned = store.getBannedTopics(testRoomId, "food", 5)

            assertEquals(listOf("banana", "matrix", "apple"), banned)
        }

    @Test
    fun `getBannedTopics should merge all categories when category is null`() =
        runTest {
            repeat(20) { idx -> store.addAsync(testRoomId, "food", "food-$idx") }
            repeat(20) { idx -> store.addAsync(testRoomId, "movie", "movie-$idx") }

            val banned = store.getBannedTopics(testRoomId, null, 20)

            assertEquals(40, banned.size)
            assertTrue((0 until 20).all { banned.contains("food-$it") })
            assertTrue((0 until 20).all { banned.contains("movie-$it") })
        }
}
