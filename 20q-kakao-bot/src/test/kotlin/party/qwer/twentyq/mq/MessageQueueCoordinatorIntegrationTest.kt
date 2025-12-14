package party.qwer.twentyq.mq

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
import party.qwer.twentyq.model.PendingMessage
import party.qwer.twentyq.mq.queue.EnqueueResult

@SpringBootTest
@org.springframework.test.context.ActiveProfiles("integration")
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
class MessageQueueCoordinatorIntegrationTest {
    @Autowired
    private lateinit var coordinator: MessageQueueCoordinator

    private val testChatId = "test-chat-coordinator-${System.currentTimeMillis()}"

    @BeforeEach
    fun cleanup() =
        runTest {
            coordinator.clear(testChatId)
        }

    @Test
    fun `enqueue and dequeue should work correctly`() =
        runTest {
            val message = PendingMessage("user1", "content", null)
            val result = coordinator.enqueue(testChatId, message)

            assertEquals(EnqueueResult.SUCCESS, result)
            assertTrue(coordinator.hasPending(testChatId))

            val dequeued = coordinator.dequeue(testChatId)
            assertNotNull(dequeued)
            assertEquals("user1", dequeued?.userId)
            assertFalse(coordinator.hasPending(testChatId))
        }

    @Test
    fun `enqueue should return DUPLICATE for same user`() =
        runTest {
            val message1 = PendingMessage("user1", "first", null)
            val message2 = PendingMessage("user1", "second", null)

            val result1 = coordinator.enqueue(testChatId, message1)
            val result2 = coordinator.enqueue(testChatId, message2)

            assertEquals(EnqueueResult.SUCCESS, result1)
            assertEquals(EnqueueResult.DUPLICATE, result2)
        }

    @Test
    fun `enqueue should return QUEUE_FULL when max size reached`() =
        runTest {
            // MAX_QUEUE_SIZE=5이므로 6개 메시지 필요
            val messages =
                (1..6).map { PendingMessage("user$it", "message$it", null) }

            val results = messages.map { coordinator.enqueue(testChatId, it) }

            assertEquals(EnqueueResult.SUCCESS, results[0])
            assertEquals(EnqueueResult.SUCCESS, results[1])
            assertEquals(EnqueueResult.SUCCESS, results[2])
            assertEquals(EnqueueResult.SUCCESS, results[3])
            assertEquals(EnqueueResult.SUCCESS, results[4])
            assertEquals(EnqueueResult.QUEUE_FULL, results[5])
        }

    @Test
    fun `hasPending should reflect queue state`() =
        runTest {
            assertFalse(coordinator.hasPending(testChatId))

            coordinator.enqueue(testChatId, PendingMessage("user1", "msg", null))
            assertTrue(coordinator.hasPending(testChatId))

            coordinator.dequeue(testChatId)
            assertFalse(coordinator.hasPending(testChatId))
        }

    @Test
    fun `size should return correct queue size`() =
        runTest {
            assertEquals(0, coordinator.size(testChatId))

            coordinator.enqueue(testChatId, PendingMessage("user1", "msg1", null))
            assertEquals(1, coordinator.size(testChatId))

            coordinator.enqueue(testChatId, PendingMessage("user2", "msg2", null))
            assertEquals(2, coordinator.size(testChatId))
        }

    @Test
    fun `getQueueDetails should return formatted queue info`() =
        runTest {
            coordinator.enqueue(testChatId, PendingMessage("userA", "msg1", null))
            coordinator.enqueue(testChatId, PendingMessage("userB", "msg2", null))

            val details = coordinator.getQueueDetails(testChatId)
            assertTrue(details.contains("userA"))
            assertTrue(details.contains("userB"))
        }

    @Test
    fun `clear should remove all messages`() =
        runTest {
            coordinator.enqueue(testChatId, PendingMessage("user1", "msg1", null))
            coordinator.enqueue(testChatId, PendingMessage("user2", "msg2", null))

            assertEquals(2, coordinator.size(testChatId))

            coordinator.clear(testChatId)
            assertEquals(0, coordinator.size(testChatId))
            assertNull(coordinator.dequeue(testChatId))
        }

    @Test
    fun `dequeue should return null when queue is empty`() =
        runTest {
            val result = coordinator.dequeue(testChatId)
            assertNull(result)
        }
}
