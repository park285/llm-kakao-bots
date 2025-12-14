package party.qwer.twentyq.bridge

import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.mockk
import io.mockk.slot
import kotlinx.coroutines.test.runTest
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import party.qwer.twentyq.model.OutboundMessage
import party.qwer.twentyq.model.PendingMessage
import party.qwer.twentyq.mq.MessageQueueCoordinator
import party.qwer.twentyq.mq.queue.EnqueueResult
import party.qwer.twentyq.redis.LockCoordinator
import party.qwer.twentyq.redis.ProcessingLockService
import party.qwer.twentyq.util.game.GameMessageProvider

/** Lock 획득 실패 시 재큐잉 시나리오 테스트 */
class MessageQueueProcessorReenqueueTest {
    private val queueCoordinator = mockk<MessageQueueCoordinator>()
    private val lockCoordinator = mockk<LockCoordinator>()
    private val processingLockService = mockk<ProcessingLockService>()
    private val messageProvider = mockk<GameMessageProvider>()
    private val exceptionHandler = mockk<MessageExceptionHandler>()
    private val commandParser = mockk<CommandParser>()
    private val commandExecutor = mockk<MessageCommandExecutor>()

    private lateinit var lockingSupport: LockingSupport
    private lateinit var messagingSupport: MessagingSupport
    private lateinit var notifier: MessageQueueNotifier
    private lateinit var processor: MessageQueueProcessor

    private val chatId = "test-chat"
    private val userId = "test-user"
    private val threadId = "test-thread"

    @BeforeEach
    fun setUp() {
        lockingSupport = LockingSupport(lockCoordinator, processingLockService)
        messagingSupport = MessagingSupport(messageProvider, exceptionHandler)
        notifier = MessageQueueNotifier(messageProvider, exceptionHandler)
        processor =
            MessageQueueProcessor(
                queueCoordinator = queueCoordinator,
                lockingSupport = lockingSupport,
                messagingSupport = messagingSupport,
                commandParser = commandParser,
                commandExecutor = commandExecutor,
                notifier = notifier,
                maxQueueProcessIterations = 10,
            )

        // 공통 mock 설정
        every { messageProvider.get("user.anonymous") } returns "익명"
        every { messageProvider.get("queue.processing", any()) } returns "처리 중"
        coEvery { commandParser.parse(any()) } returns null
    }

    @Test
    fun `should send retry notification when re-enqueue succeeds after lock failure`() =
        runTest {
            // Given
            val pending = PendingMessage(userId, "테스트", threadId)
            val emittedMessages = mutableListOf<OutboundMessage>()

            coEvery { queueCoordinator.dequeue(chatId) } returns pending
            coEvery { lockCoordinator.withLock<Unit>(chatId, userId, true, any(), any()) } returns null
            coEvery { queueCoordinator.enqueue(chatId, pending) } returns EnqueueResult.SUCCESS
            every { messageProvider.get("queue.retry", any()) } returns "잠시 후 다시 처리됩니다"

            // When
            processor.processQueuedMessages(chatId) { emittedMessages.add(it) }

            // Then
            coVerify { queueCoordinator.enqueue(chatId, pending) }
            assertThat(emittedMessages).anyMatch {
                it is OutboundMessage.Waiting && it.text.contains("잠시 후 다시 처리됩니다")
            }
        }

    @Test
    fun `should send duplicate notification when re-enqueue returns DUPLICATE`() =
        runTest {
            // Given
            val pending = PendingMessage(userId, "테스트", threadId)
            val emittedMessages = mutableListOf<OutboundMessage>()

            coEvery { queueCoordinator.dequeue(chatId) } returns pending
            coEvery { lockCoordinator.withLock<Unit>(chatId, userId, true, any(), any()) } returns null
            coEvery { queueCoordinator.enqueue(chatId, pending) } returns EnqueueResult.DUPLICATE
            every { messageProvider.get("queue.retry_duplicate", any()) } returns "이미 대기 중인 요청이 있습니다"

            // When
            processor.processQueuedMessages(chatId) { emittedMessages.add(it) }

            // Then
            coVerify { queueCoordinator.enqueue(chatId, pending) }
            assertThat(emittedMessages).anyMatch {
                it is OutboundMessage.Waiting && it.text.contains("이미 대기 중인 요청")
            }
        }

    @Test
    fun `should send error notification when re-enqueue returns QUEUE_FULL`() =
        runTest {
            // Given
            val pending = PendingMessage(userId, "테스트", threadId)
            val emittedMessages = mutableListOf<OutboundMessage>()

            coEvery { queueCoordinator.dequeue(chatId) } returns pending
            coEvery { lockCoordinator.withLock<Unit>(chatId, userId, true, any(), any()) } returns null
            coEvery { queueCoordinator.enqueue(chatId, pending) } returns EnqueueResult.QUEUE_FULL
            every { messageProvider.get("queue.retry_failed", any()) } returns "대기열이 가득 찼습니다"

            // When
            processor.processQueuedMessages(chatId) { emittedMessages.add(it) }

            // Then
            coVerify { queueCoordinator.enqueue(chatId, pending) }
            // QUEUE_FULL은 Error 타입으로 전송
            assertThat(emittedMessages).anyMatch {
                it is OutboundMessage.Error && it.text.contains("대기열이 가득")
            }
        }

    @Test
    fun `should call enqueue before notification on lock failure`() =
        runTest {
            // Given: 순서 검증용
            val pending = PendingMessage(userId, "테스트", threadId)
            val callOrder = mutableListOf<String>()

            coEvery { queueCoordinator.dequeue(chatId) } returns pending
            coEvery { lockCoordinator.withLock<Unit>(chatId, userId, true, any(), any()) } returns null
            coEvery { queueCoordinator.enqueue(chatId, pending) } answers {
                callOrder.add("enqueue")
                EnqueueResult.SUCCESS
            }
            every { messageProvider.get("queue.retry", any()) } answers {
                callOrder.add("notify")
                "잠시 후 다시 처리됩니다"
            }

            // When
            processor.processQueuedMessages(chatId) { }

            // Then: enqueue가 notify보다 먼저 호출되어야 함
            assertThat(callOrder).containsExactly("enqueue", "notify")
        }
}
