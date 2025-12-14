package party.qwer.twentyq.bridge

import io.mockk.mockk
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Test
import party.qwer.twentyq.bridge.handlers.MessageHandlers
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.config.properties.RedisDefaults
import party.qwer.twentyq.model.Command
import party.qwer.twentyq.mq.MessageQueueCoordinator
import party.qwer.twentyq.redis.LockCoordinator
import party.qwer.twentyq.redis.ProcessingLockService
import party.qwer.twentyq.redis.tracking.PlayerSetStore
import party.qwer.twentyq.rest.LlmAvailabilityGuard
import party.qwer.twentyq.rest.NlpRestClient
import party.qwer.twentyq.service.UserStatsService
import party.qwer.twentyq.util.game.GameMessageProvider

class TwentyQMessageServiceTest {
    private val appProperties = AppProperties(cache = RedisDefaults())
    private val service =
        TwentyQMessageService(
            appProperties = appProperties,
            lockCoordinator = mockk<LockCoordinator>(relaxed = true),
            processingLockService = mockk<ProcessingLockService>(relaxed = true),
            queueCoordinator = mockk<MessageQueueCoordinator>(relaxed = true),
            playerSetStore = mockk<PlayerSetStore>(relaxed = true),
            handlers = mockk<MessageHandlers>(relaxed = true),
            messageProvider = mockk<GameMessageProvider>(relaxed = true),
            exceptionHandler = mockk<MessageExceptionHandler>(relaxed = true),
            nlpRestClient = mockk<NlpRestClient>(relaxed = true),
            userStatsService = mockk<UserStatsService>(relaxed = true),
            chainedQuestionProcessor = mockk<ChainedQuestionProcessor>(relaxed = true),
            waitingMessageCoordinator = mockk<WaitingMessageCoordinator>(relaxed = true),
            messageEmitter = mockk<MessageEmitter>(relaxed = true),
            llmAvailabilityGuard = mockk<LlmAvailabilityGuard>(relaxed = true),
        )

    @Test
    fun `model info command should bypass existing session requirement`() {
        val method = TwentyQMessageService::class.java.getDeclaredMethod("requiresExistingSession", Command::class.java)
        method.isAccessible = true

        val requiresSession = method.invoke(service, Command.ModelInfo) as Boolean

        assertThat(requiresSession).isFalse()
    }
}
