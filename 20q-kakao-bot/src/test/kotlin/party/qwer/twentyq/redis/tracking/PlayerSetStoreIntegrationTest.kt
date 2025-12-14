package party.qwer.twentyq.redis.tracking

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.delay
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.test.annotation.DirtiesContext
import party.qwer.twentyq.model.PlayerInfo
import java.time.Duration

@SpringBootTest
@org.springframework.test.context.ActiveProfiles("integration")
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
class PlayerSetStoreIntegrationTest {
    @Autowired
    private lateinit var store: PlayerSetStore

    private val testRoomId = "test-room-${System.currentTimeMillis()}"

    @BeforeEach
    fun cleanup() =
        runTest {
            store.clearAsync(testRoomId)
        }

    @Test
    fun `addAsync should add player to set`() =
        runTest {
            store.addAsync(testRoomId, "user1", "사용자1")
            store.addAsync(testRoomId, "user2", "사용자2")

            val players = store.getAllAsync(testRoomId)
            assertEquals(2, players.size)
            assertTrue(players.any { it.userId == "user1" })
            assertTrue(players.any { it.userId == "user2" })
        }

    @Test
    fun `addAsync should not duplicate players`() =
        runTest {
            store.addAsync(testRoomId, "user1", "사용자1")
            store.addAsync(testRoomId, "user1", "사용자1-수정")
            store.addAsync(testRoomId, "user1", "사용자1-재수정")

            val players = store.getAllAsync(testRoomId)
            assertEquals(1, players.size, "중복 플레이어는 추가되지 않아야 함")
            assertTrue(players.any { it.userId == "user1" })
        }

    @Test
    fun `concurrent addAsync should handle set operations correctly`() =
        runTest {
            val userIds = (1..20).map { "user$it" }
            userIds
                .map { userId ->
                    async(Dispatchers.IO) {
                        store.addAsync(testRoomId, userId, "사용자$userId")
                    }
                }.awaitAll()

            val players = store.getAllAsync(testRoomId)
            assertEquals(20, players.size, "20명의 플레이어가 모두 추가되어야 함")
            userIds.forEach { userId ->
                assertTrue(players.any { it.userId == userId }, "$userId 가 Set에 존재해야 함")
            }
        }

    @Test
    fun `getAllAsync should return empty set when no players exist`() =
        runTest {
            val players = store.getAllAsync(testRoomId)
            assertTrue(players.isEmpty())
        }

    @Test
    fun `clearAsync should remove all players`() =
        runTest {
            store.addAsync(testRoomId, "user1", "사용자1")
            store.addAsync(testRoomId, "user2", "사용자2")

            store.clearAsync(testRoomId)
            val players = store.getAllAsync(testRoomId)

            assertTrue(players.isEmpty())
        }
}
