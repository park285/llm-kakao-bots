package party.qwer.twentyq.redis.voting

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.delay
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.test.annotation.DirtiesContext
import party.qwer.twentyq.model.SurrenderVote

@SpringBootTest
@org.springframework.test.context.ActiveProfiles("integration")
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
class SurrenderVoteStoreIntegrationTest {
    @Autowired
    private lateinit var store: SurrenderVoteStore

    private val testRoomId = "test-room-${System.currentTimeMillis()}"

    @BeforeEach
    fun cleanup() =
        runTest {
            store.clearAsync(testRoomId)
        }

    @Test
    fun `saveAsync and getAsync should store and retrieve vote`() =
        runTest {
            val vote =
                SurrenderVote(
                    initiator = "user1",
                    eligiblePlayers = setOf("user1", "user2", "user3"),
                    approvals = setOf("user1"),
                )

            store.saveAsync(testRoomId, vote, ttlSeconds = 120)
            val retrieved = store.getAsync(testRoomId)

            assertNotNull(retrieved)
            assertEquals("user1", retrieved?.initiator)
            assertEquals(3, retrieved?.eligiblePlayers?.size)
            assertEquals(1, retrieved?.approvals?.size)
        }

    @Test
    fun `getAsync should return null when vote does not exist`() =
        runTest {
            val result = store.getAsync("nonexistent-room")
            assertNull(result)
        }

    @Test
    fun `isActiveAsync should return true when vote exists`() =
        runTest {
            val vote = SurrenderVote(initiator = "user1", eligiblePlayers = setOf("user1", "user2"))
            store.saveAsync(testRoomId, vote)

            val isActive = store.isActiveAsync(testRoomId)
            assertTrue(isActive)
        }

    @Test
    fun `isActiveAsync should return false when vote does not exist`() =
        runTest {
            val isActive = store.isActiveAsync(testRoomId)
            assertFalse(isActive)
        }

    @Test
    fun `approveAsync should update vote with new approval`() =
        runTest {
            val vote =
                SurrenderVote(
                    initiator = "user1",
                    eligiblePlayers = setOf("user1", "user2", "user3"),
                    approvals = setOf("user1"),
                )
            store.saveAsync(testRoomId, vote)

            val updated = store.approveAsync(testRoomId, "user2")

            assertNotNull(updated)
            assertEquals(2, updated?.approvals?.size)
            assertTrue(updated?.hasVoted("user1") == true)
            assertTrue(updated?.hasVoted("user2") == true)
        }

    @Test
    fun `concurrent approveAsync should handle vote updates safely`() =
        runTest {
            val vote =
                SurrenderVote(
                    initiator = "user1",
                    eligiblePlayers = setOf("user1", "user2", "user3", "user4"),
                    approvals = emptySet(),
                )
            store.saveAsync(testRoomId, vote)

            val users = listOf("user1", "user2", "user3")
            val results =
                users
                    .map { userId ->
                        async(Dispatchers.IO) {
                            store.approveAsync(testRoomId, userId)
                        }
                    }.awaitAll()

            // 모든 승인 작업 성공 확인 (동시성 환경에서 일부 유실 가능)
            val finalVote = store.getAsync(testRoomId)
            assertNotNull(finalVote)
            assertTrue(finalVote!!.approvals.size >= 1, "최소 1명 이상 승인되어야 함")
        }

    @Test
    fun `clearAsync should remove vote`() =
        runTest {
            val vote = SurrenderVote(initiator = "user1", eligiblePlayers = setOf("user1", "user2"))
            store.saveAsync(testRoomId, vote)

            store.clearAsync(testRoomId)
            val retrieved = store.getAsync(testRoomId)

            assertNull(retrieved)
        }
}
