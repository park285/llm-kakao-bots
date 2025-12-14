package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.bridge.CommandParser
import io.github.kapu.turtlesoup.models.Command
import io.github.kapu.turtlesoup.mq.handler.GameCommandHandler
import io.github.kapu.turtlesoup.mq.models.InboundMessage
import io.github.kapu.turtlesoup.mq.models.OutboundMessage
import io.github.kapu.turtlesoup.redis.ProcessingLockService
import io.github.kapu.turtlesoup.rest.LlmRestClient
import io.github.kapu.turtlesoup.utils.AccessControl
import io.github.kapu.turtlesoup.utils.MessageKeys
import io.github.kapu.turtlesoup.utils.MessageProvider
import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.collections.shouldContainExactly
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.mockk
import kotlinx.coroutines.runBlocking

private data class Fixture(
    val service: GameMessageService,
    val commandHandler: GameCommandHandler,
    val messageSender: MessageSender,
    val messageProvider: MessageProvider,
    val accessControl: AccessControl,
    val commandParser: CommandParser,
    val processingLockService: ProcessingLockService,
    val restClient: LlmRestClient,
)

private fun createFixture(): Fixture {
    val commandHandler = mockk<GameCommandHandler>()
    val messageSender = mockk<MessageSender>()
    val messageProvider = mockk<MessageProvider>()
    val publisher = mockk<ValkeyMQReplyPublisher>()
    val accessControl = mockk<AccessControl>()
    val commandParser = mockk<CommandParser>()
    val processingLockService = mockk<ProcessingLockService>()
    val queueProcessor = mockk<MessageQueueProcessor>()
    val restClient = mockk<LlmRestClient>()

    val service =
        GameMessageService(
            commandHandler = commandHandler,
            messageSender = messageSender,
            messageProvider = messageProvider,
            publisher = publisher,
            accessControl = accessControl,
            commandParser = commandParser,
            processingLockService = processingLockService,
            queueProcessor = queueProcessor,
            restClient = restClient,
        )

    return Fixture(
        service = service,
        commandHandler = commandHandler,
        messageSender = messageSender,
        messageProvider = messageProvider,
        accessControl = accessControl,
        commandParser = commandParser,
        processingLockService = processingLockService,
        restClient = restClient,
    )
}

class GameMessageServiceTest : StringSpec({
    "handleMessage blocks Start without waiting when LLM unhealthy" {
        val fixture = createFixture()

        val message =
            InboundMessage(
                chatId = "c1",
                userId = "u1",
                content = "/스프 시작",
                threadId = "t1",
                sender = "s1",
            )
        val unavailableText = "AI 서버 점검 중입니다. 잠시 후 다시 시도해주세요."

        every { fixture.accessControl.getDenialReason("u1", "c1") } returns null
        every { fixture.commandParser.parse("/스프 시작") } returns Command.Start()
        coEvery { fixture.restClient.isHealthy() } returns false
        every { fixture.messageProvider.get(MessageKeys.ERROR_AI_UNAVAILABLE) } returns unavailableText
        coEvery { fixture.messageSender.sendFinal(message, unavailableText) } returns Unit

        runBlocking { fixture.service.handleMessage(message) }

        coVerify(exactly = 1) { fixture.messageSender.sendFinal(message, unavailableText) }
        coVerify(exactly = 0) { fixture.messageSender.sendWaiting(any(), any()) }
        coVerify(exactly = 0) { fixture.processingLockService.isProcessing(any()) }
        coVerify(exactly = 0) { fixture.commandHandler.processCommand(any(), any()) }
    }

    "handleMessage blocks Help when LLM unhealthy" {
        val fixture = createFixture()

        val message =
            InboundMessage(
                chatId = "c1",
                userId = "u1",
                content = "/스프",
                threadId = "t1",
                sender = "s1",
            )
        val unavailableText = "AI 서버 점검 중입니다. 잠시 후 다시 시도해주세요."

        every { fixture.accessControl.getDenialReason("u1", "c1") } returns null
        every { fixture.commandParser.parse("/스프") } returns Command.Help
        coEvery { fixture.restClient.isHealthy() } returns false
        every { fixture.messageProvider.get(MessageKeys.ERROR_AI_UNAVAILABLE) } returns unavailableText
        coEvery { fixture.messageSender.sendFinal(message, unavailableText) } returns Unit

        runBlocking { fixture.service.handleMessage(message) }

        coVerify(exactly = 1) { fixture.messageSender.sendFinal(message, unavailableText) }
        coVerify(exactly = 0) { fixture.commandHandler.processCommand(any(), any()) }
    }

    "handleQueuedCommand blocks Start without waiting when LLM unhealthy" {
        val fixture = createFixture()

        val message =
            InboundMessage(
                chatId = "c1",
                userId = "u1",
                content = "/스프 시작",
                threadId = "t1",
                sender = "s1",
            )
        val unavailableText = "AI 서버 점검 중입니다. 잠시 후 다시 시도해주세요."

        coEvery { fixture.restClient.isHealthy() } returns false
        every { fixture.messageProvider.get(MessageKeys.ERROR_AI_UNAVAILABLE) } returns unavailableText

        val emitted = mutableListOf<OutboundMessage>()
        runBlocking {
            fixture.service.handleQueuedCommand(message, Command.Start(), emit = { outbound -> emitted.add(outbound) })
        }

        emitted.shouldContainExactly(
            OutboundMessage.Final(chatId = "c1", text = unavailableText, threadId = "t1"),
        )
    }
})
