package io.github.kapu.turtlesoup.config

import io.github.kapu.turtlesoup.bridge.CommandParser
import io.github.kapu.turtlesoup.bridge.SurrenderHandler
import io.github.kapu.turtlesoup.mq.GameMessageService
import io.github.kapu.turtlesoup.mq.MessageQueueCoordinator
import io.github.kapu.turtlesoup.mq.MessageQueueNotifier
import io.github.kapu.turtlesoup.mq.MessageQueueProcessor
import io.github.kapu.turtlesoup.mq.MessageSender
import io.github.kapu.turtlesoup.mq.ValkeyMQMessageHandler
import io.github.kapu.turtlesoup.mq.ValkeyMQReplyPublisher
import io.github.kapu.turtlesoup.mq.ValkeyMQStreamConsumer
import io.github.kapu.turtlesoup.mq.createRedissonReactiveClient
import io.github.kapu.turtlesoup.mq.handler.GameCommandHandler
import io.github.kapu.turtlesoup.mq.models.InboundMessage
import io.github.kapu.turtlesoup.redis.LockManager
import io.github.kapu.turtlesoup.redis.PendingMessageStore
import io.github.kapu.turtlesoup.redis.ProcessingLockService
import io.github.kapu.turtlesoup.redis.PuzzleDedupStore
import io.github.kapu.turtlesoup.redis.SessionStore
import io.github.kapu.turtlesoup.redis.SurrenderVoteStore
import io.github.kapu.turtlesoup.rest.LlmHealthMonitor
import io.github.kapu.turtlesoup.rest.LlmRestClient
import io.github.kapu.turtlesoup.security.McpInjectionGuard
import io.github.kapu.turtlesoup.service.GameService
import io.github.kapu.turtlesoup.service.GameSessionManager
import io.github.kapu.turtlesoup.service.GameSetupService
import io.github.kapu.turtlesoup.service.PuzzleService
import io.github.kapu.turtlesoup.service.SessionValidator
import io.github.kapu.turtlesoup.service.SurrenderVoteService
import io.github.kapu.turtlesoup.utils.AccessControl
import io.github.kapu.turtlesoup.utils.MessageProvider
import io.github.oshai.kotlinlogging.KotlinLogging
import org.koin.core.qualifier.named
import org.koin.dsl.module
import org.redisson.Redisson
import org.redisson.api.RedissonClient
import org.redisson.api.RedissonReactiveClient
import org.redisson.client.codec.StringCodec
import org.redisson.config.Config

private val log = KotlinLogging.logger {}

val appModule =
    module {
        // Settings
        single { Settings.load() }

        // Redisson Client (Base)
        single<RedissonClient> {
            val settings: Settings = get()
            val config = Config()

            val address =
                if (settings.redis.password != null) {
                    "redis://${settings.redis.password}@${settings.redis.host}:${settings.redis.port}"
                } else {
                    "redis://${settings.redis.host}:${settings.redis.port}"
                }

            config.useSingleServer()
                .setAddress(address)
                .setConnectionPoolSize(RedissonConnectionConstants.CONNECTION_POOL_SIZE)
                .setConnectionMinimumIdleSize(RedissonConnectionConstants.CONNECTION_MINIMUM_IDLE_SIZE)
                .setIdleConnectionTimeout(RedissonConnectionConstants.IDLE_CONNECTION_TIMEOUT_MS)
                .setConnectTimeout(RedissonConnectionConstants.CONNECT_TIMEOUT_MS)
                .setTimeout(RedissonConnectionConstants.TIMEOUT_MS)

            config.codec = StringCodec.INSTANCE

            Redisson.create(config)
        }

        // Redisson Reactive Client
        single<RedissonReactiveClient> {
            get<RedissonClient>().reactive()
        }

        // LLM REST Client
        single {
            val settings: Settings = get()
            log.info { "llm_rest_client_created base_url=${settings.llmRest.baseUrl}" }
            LlmRestClient(settings.llmRest)
        }

        // LLM Health Monitor
        single {
            val settings: Settings = get()
            LlmHealthMonitor(settings, get(), lockManager = get())
        }

        // Redis Stores (Reactive)
        single { SessionStore(get<RedissonReactiveClient>()) }
        single { LockManager(get<RedissonReactiveClient>()) }
        single { SurrenderVoteStore(get<RedissonReactiveClient>()) }
        single { ProcessingLockService(get<RedissonReactiveClient>()) }
        single { PendingMessageStore(get<RedissonReactiveClient>()) }
        single { PuzzleDedupStore(get<RedissonReactiveClient>()) }
        single { SessionValidator(get()) }
        single {
            SurrenderVoteService(
                sessionStore = get(),
                voteStore = get(),
                sessionValidator = get(),
            )
        }

        // PuzzleService (REST based - fetches from mcp-llm-server)
        single {
            val settings: Settings = get()
            PuzzleService(
                restClient = get(),
                puzzleConfig = settings.puzzle,
                dedupStore = get(),
            )
        }

        // Security (MCP REST 기반)
        single { McpInjectionGuard(get()) }

        // Game session helpers
        single {
            GameSessionManager(
                sessionStore = get(),
                lockManager = get(),
            )
        }
        single {
            GameSetupService(
                restClient = get(),
                puzzleService = get(),
                sessionManager = get(),
            )
        }

        // GameService (REST based)
        single {
            GameService(
                restClient = get(),
                sessionManager = get(),
                setupService = get(),
                injectionGuard = get(),
            )
        }

        // Command Parser
        single { CommandParser() }

        // Surrender Handler
        single {
            SurrenderHandler(
                gameService = get(),
                voteService = get(),
                messageProvider = get(),
            )
        }

        // Message Provider
        single {
            MessageProvider.fromClasspath("/messages/game-messages.yml")
        }

        // Access Control
        single {
            val settings: Settings = get()
            AccessControl(settings.access)
        }

        // MQ - Reactive Client
        single<RedissonReactiveClient>(named("mqClient")) {
            val settings: Settings = get()
            createRedissonReactiveClient(settings.valkeyMq)
        }

        // MQ - Publisher
        single {
            val client: RedissonReactiveClient = get(named("mqClient"))
            val settings: Settings = get()
            ValkeyMQReplyPublisher(client, settings.valkeyMq.replyStreamKey)
        }

        // MQ - Queue Components
        single { MessageQueueCoordinator(get()) }
        single { MessageQueueNotifier(get()) }
        single {
            val commandParserRef: CommandParser = get()
            MessageQueueProcessor(
                queueCoordinator = get(),
                lockManager = get(),
                processingLockService = get(),
                messageProvider = get(),
                notifier = get(),
                commandParser = get(),
                commandExecutor = { chatId, userId, content, threadId, sender, emit ->
                    val service: GameMessageService = get()
                    val command = commandParserRef.parse(content)
                    if (command != null) {
                        val message =
                            InboundMessage(
                                chatId = chatId,
                                userId = userId,
                                content = content,
                                threadId = threadId,
                                sender = sender,
                            )
                        service.handleQueuedCommand(message, command, emit)
                    }
                },
            )
        }

        // MQ - Game Command Handler
        single {
            GameCommandHandler(
                gameService = get(),
                surrenderHandler = get(),
                messageProvider = get(),
                publisher = get(),
                restClient = get(),
            )
        }

        // MQ - Message Sender
        single {
            MessageSender(
                messageProvider = get(),
                publisher = get(),
            )
        }

        // MQ - Game Message Service
        single {
            GameMessageService(
                commandHandler = get(),
                messageSender = get(),
                messageProvider = get(),
                publisher = get(),
                accessControl = get(),
                commandParser = get(),
                processingLockService = get(),
                queueProcessor = get(),
                restClient = get(),
            )
        }

        // MQ - Message Handler
        single {
            ValkeyMQMessageHandler(gameMessageService = get())
        }

        // MQ - Stream Consumer
        single {
            val client: RedissonReactiveClient = get(named("mqClient"))
            val settings: Settings = get()
            val handler: ValkeyMQMessageHandler = get()

            ValkeyMQStreamConsumer(
                redissonClient = client,
                streamKey = settings.valkeyMq.streamKey,
                consumerGroup = settings.valkeyMq.consumerGroup,
                consumerName = settings.valkeyMq.consumerName,
                messageHandler = handler,
            )
        }
    }
