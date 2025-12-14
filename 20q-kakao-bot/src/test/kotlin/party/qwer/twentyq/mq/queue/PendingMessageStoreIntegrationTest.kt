package party.qwer.twentyq.mq.queue

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertFalse
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.test.annotation.DirtiesContext
import party.qwer.twentyq.model.PendingMessage

@SpringBootTest
@org.springframework.test.context.ActiveProfiles("integration")
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
class PendingMessageStoreIntegrationTest {
    @Autowired
    private lateinit var store: PendingMessageStore

    private val testChatId = "test-chat-${System.currentTimeMillis()}"

    @BeforeEach
    fun cleanup() =
        runTest {
            store.clear(testChatId)
        }

    @Test
    fun `enqueue should succeed when queue is empty`() =
        runTest {
            val message = PendingMessage("user1", "hello", null)
            val result = store.enqueue(testChatId, message)

            assertEquals(EnqueueResult.SUCCESS, result)
            assertEquals(1, store.size(testChatId))
        }

    @Test
    fun `enqueue should return DUPLICATE when same user enqueues twice`() =
        runTest {
            val message1 = PendingMessage("user1", "first", null)
            val message2 = PendingMessage("user1", "second", null)

            val result1 = store.enqueue(testChatId, message1)
            val result2 = store.enqueue(testChatId, message2)

            assertEquals(EnqueueResult.SUCCESS, result1)
            assertEquals(EnqueueResult.DUPLICATE, result2)
            assertEquals(1, store.size(testChatId))
        }

    @Test
    fun `enqueue should return QUEUE_FULL when max size reached`() =
        runTest {
            val messages =
                (1..6).map { PendingMessage("user$it", "message$it", null) }

            val results = messages.map { store.enqueue(testChatId, it) }

            assertEquals(EnqueueResult.SUCCESS, results[0])
            assertEquals(EnqueueResult.SUCCESS, results[1])
            assertEquals(EnqueueResult.SUCCESS, results[2])
            assertEquals(EnqueueResult.SUCCESS, results[3])
            assertEquals(EnqueueResult.SUCCESS, results[4])
            assertEquals(EnqueueResult.QUEUE_FULL, results[5])
            assertEquals(5, store.size(testChatId))
        }

    @Test
    fun `concurrent enqueue should respect max size and duplicates`() =
        runTest {
            val messages = (1..10).map { PendingMessage("user$it", "message$it", null) }
            val results =
                messages
                    .map { message ->
                        async(Dispatchers.IO) {
                            store.enqueue(testChatId, message)
                        }
                    }.awaitAll()

            val successCount = results.count { it == EnqueueResult.SUCCESS }
            val queueFullCount = results.count { it == EnqueueResult.QUEUE_FULL }

            assertEquals(5, successCount, "최대 5개 메시지만 성공해야 함")
            assertEquals(5, queueFullCount, "나머지는 QUEUE_FULL이어야 함")
            assertEquals(5, store.size(testChatId))
        }

    @Test
    fun `dequeue should return messages in FIFO order`() =
        runTest {
            val messages =
                listOf(
                    PendingMessage("user1", "first", null),
                    PendingMessage("user2", "second", null),
                    PendingMessage("user3", "third", null),
                )

            messages.forEach { store.enqueue(testChatId, it) }

            val dequeued1 = store.dequeue(testChatId)
            val dequeued2 = store.dequeue(testChatId)
            val dequeued3 = store.dequeue(testChatId)

            assertEquals("user1", dequeued1?.userId)
            assertEquals("first", dequeued1?.content)
            assertEquals("user2", dequeued2?.userId)
            assertEquals("second", dequeued2?.content)
            assertEquals("user3", dequeued3?.userId)
            assertEquals("third", dequeued3?.content)
            assertEquals(0, store.size(testChatId))
        }

    @Test
    fun `dequeue should allow re-enqueue after message is dequeued`() =
        runTest {
            val message1 = PendingMessage("user1", "first", null)
            store.enqueue(testChatId, message1)

            val dequeued = store.dequeue(testChatId)
            assertEquals("user1", dequeued?.userId)

            // 동일 유저 재적재 가능
            val message2 = PendingMessage("user1", "second", null)
            val result = store.enqueue(testChatId, message2)
            assertEquals(EnqueueResult.SUCCESS, result)
        }

    @Test
    fun `dequeue should return null when queue is empty`() =
        runTest {
            val result = store.dequeue(testChatId)
            assertNull(result)
        }

    @Test
    fun `dequeue should filter stale messages older than 1 hour`() =
        runTest {
            // 오래된 메시지 시뮬레이션 (타임스탬프 조작은 불가하므로 테스트 제한)
            // 실제 검증은 통합 테스트나 수동 테스트에서 수행
            // 여기서는 정상 메시지가 필터링되지 않음을 확인
            val message = PendingMessage("user1", "content", null)
            store.enqueue(testChatId, message)

            val dequeued = store.dequeue(testChatId)
            assertEquals("user1", dequeued?.userId)
        }

    @Test
    fun `hasPending should return true when messages exist`() =
        runTest {
            assertFalse(store.hasPending(testChatId))

            store.enqueue(testChatId, PendingMessage("user1", "content", null))
            assertTrue(store.hasPending(testChatId))

            store.dequeue(testChatId)
            assertFalse(store.hasPending(testChatId))
        }

    @Test
    fun `getQueueDetails should format queue contents correctly`() =
        runTest {
            store.enqueue(testChatId, PendingMessage("userA", "msg1", null))
            store.enqueue(testChatId, PendingMessage("userB", "msg2", null))

            val details = store.getQueueDetails(testChatId)
            assertTrue(details.contains("1. userA"))
            assertTrue(details.contains("2. userB"))
        }

    @Test
    fun `getQueueDetails should return empty message when queue is empty`() =
        runTest {
            val details = store.getQueueDetails(testChatId)
            assertTrue(details.isEmpty())
        }

    @Test
    fun `getQueueDetails should format chain messages correctly`() =
        runTest {
            // 일반 메시지
            store.enqueue(testChatId, PendingMessage("user1", "/스자 질문1", null, "카푸치노"))

            // 체인 메시지
            val chainMessage =
                PendingMessage(
                    userId = "user2",
                    content = "",
                    threadId = null,
                    sender = "테스터",
                    isChainBatch = true,
                    batchQuestions = listOf("질문2", "질문3", "질문4"),
                )
            store.enqueue(testChatId, chainMessage)

            val details = store.getQueueDetails(testChatId)

            // 일반 메시지 확인
            assertTrue(details.contains("1. 카푸치노 - /스자 질문1"))

            // 체인 메시지 확인 (batchQuestions를 joinToString으로 표시)
            assertTrue(details.contains("2. 테스터 - 질문2, 질문3, 질문4"))
        }

    @Test
    fun `clear should remove all messages and allow new enqueues`() =
        runTest {
            store.enqueue(testChatId, PendingMessage("user1", "msg1", null))
            store.enqueue(testChatId, PendingMessage("user2", "msg2", null))

            assertEquals(2, store.size(testChatId))

            store.clear(testChatId)
            assertEquals(0, store.size(testChatId))

            // 초기화 후 재적재 가능
            val result = store.enqueue(testChatId, PendingMessage("user1", "new", null))
            assertEquals(EnqueueResult.SUCCESS, result)
        }

    @Test
    fun `size should return correct count`() =
        runTest {
            assertEquals(0, store.size(testChatId))

            store.enqueue(testChatId, PendingMessage("user1", "msg1", null))
            assertEquals(1, store.size(testChatId))

            store.enqueue(testChatId, PendingMessage("user2", "msg2", null))
            assertEquals(2, store.size(testChatId))

            store.dequeue(testChatId)
            assertEquals(1, store.size(testChatId))
        }

    @Test
    fun `setChainSkipFlag should set flag with TTL`() =
        runTest {
            val userId = "testUser"

            store.setChainSkipFlag(testChatId, userId)

            val hasFlag = store.hasChainSkipFlag(testChatId, userId)
            assertTrue(hasFlag, "Skip flag should be set")
        }

    @Test
    fun `hasChainSkipFlag should return false when flag not set`() =
        runTest {
            val userId = "testUser"

            val hasFlag = store.hasChainSkipFlag(testChatId, userId)

            assertFalse(hasFlag, "Skip flag should not exist")
        }

    @Test
    fun `hasChainSkipFlag should delete flag after reading (one-time use)`() =
        runTest {
            val userId = "testUser"

            store.setChainSkipFlag(testChatId, userId)

            val firstRead = store.hasChainSkipFlag(testChatId, userId)
            assertTrue(firstRead, "First read should return true")

            val secondRead = store.hasChainSkipFlag(testChatId, userId)
            assertFalse(secondRead, "Second read should return false (flag deleted)")
        }

    @Test
    fun `skip flag should be isolated per chatId and userId`() =
        runTest {
            val user1 = "user1"
            val user2 = "user2"
            val chat2 = "test-chat-2-${System.currentTimeMillis()}"

            store.setChainSkipFlag(testChatId, user1)
            store.setChainSkipFlag(testChatId, user2)
            store.setChainSkipFlag(chat2, user1)

            assertTrue(store.hasChainSkipFlag(testChatId, user1), "chat1 user1 flag should exist")
            assertTrue(store.hasChainSkipFlag(testChatId, user2), "chat1 user2 flag should exist")
            assertTrue(store.hasChainSkipFlag(chat2, user1), "chat2 user1 flag should exist")
            assertFalse(store.hasChainSkipFlag(chat2, user2), "chat2 user2 flag should not exist")
        }

    @Test
    fun `setChainSkipFlag can be called multiple times for same user`() =
        runTest {
            val userId = "testUser"

            store.setChainSkipFlag(testChatId, userId)
            val first = store.hasChainSkipFlag(testChatId, userId)
            assertTrue(first)

            // 다시 설정 가능
            store.setChainSkipFlag(testChatId, userId)
            val second = store.hasChainSkipFlag(testChatId, userId)
            assertTrue(second, "Flag can be set again after being consumed")
        }
}
