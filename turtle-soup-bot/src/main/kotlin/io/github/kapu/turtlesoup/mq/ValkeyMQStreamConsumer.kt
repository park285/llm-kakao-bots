package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.config.MQConstants
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.reactor.awaitSingleOrNull
import kotlinx.coroutines.sync.Semaphore
import org.redisson.api.RStreamReactive
import org.redisson.api.RedissonReactiveClient
import org.redisson.api.StreamMessageId
import org.redisson.api.stream.StreamCreateGroupArgs
import org.redisson.api.stream.StreamReadGroupArgs
import org.redisson.client.codec.StringCodec
import java.time.Duration

/** Valkey Stream Consumer (메시지 소비) */
class ValkeyMQStreamConsumer(
    private val redissonClient: RedissonReactiveClient,
    private val streamKey: String,
    private val consumerGroup: String,
    private val consumerName: String,
    private val messageHandler: ValkeyMQMessageHandler,
) {
    // SupervisorJob으로 개별 처리 실패 시 전체 루프가 중단되지 않도록 보호
    private val scope = CoroutineScope(Dispatchers.Default + SupervisorJob())
    private val semaphore = Semaphore(MQConstants.SEMAPHORE_PERMITS)
    private var consumerJob: Job? = null

    // Iris (Jedis)와 호환성을 위해 StringCodec 사용
    private fun getStream(): RStreamReactive<String, String> = redissonClient.getStream(streamKey, StringCodec.INSTANCE)

    fun start() {
        logger.info {
            "consumer_starting " +
                "stream=$streamKey " +
                "group=$consumerGroup " +
                "consumer=$consumerName"
        }

        consumerJob =
            scope.launch {
                try {
                    resetConsumerGroupOnStartup()
                } catch (e: CancellationException) {
                    throw e
                } catch (e: Exception) {
                    logger.error(e) {
                        "consumer_start_failed stream=$streamKey group=$consumerGroup consumer=$consumerName"
                    }
                    return@launch
                }

                logger.info { "consumer_started" }
                runConsumerLoop()
            }
    }

    /**
     * Consumer 중지
     */
    fun stop() {
        logger.info { "consumer_stopping" }
        consumerJob?.cancel()
        logger.info { "consumer_stopped" }
    }

    private suspend fun resetConsumerGroupOnStartup() {
        val stream = getStream()

        runCatching { stream.removeGroup(consumerGroup).awaitSingleOrNull() }
            .onSuccess {
                logger.info { "consumer_group_removed stream=$streamKey group=$consumerGroup" }
            }.onFailure { error ->
                val message = error.message.orEmpty()
                if (message.contains("NOGROUP") || message.contains("no such key", ignoreCase = true)) {
                    logger.info {
                        "consumer_group_remove_skipped stream=$streamKey group=$consumerGroup reason=not_found"
                    }
                } else {
                    logger.warn(error) { "consumer_group_remove_failed stream=$streamKey group=$consumerGroup" }
                }
            }

        try {
            val args =
                StreamCreateGroupArgs
                    .name(consumerGroup)
                    .id(StreamMessageId.NEWEST)
                    .makeStream()

            stream.createGroup(args).awaitSingleOrNull()
            logger.info { "consumer_group_created stream=$streamKey group=$consumerGroup id=NEWEST" }
        } catch (e: org.redisson.client.RedisException) {
            // BUSYGROUP = 이미 존재하는 그룹
            if (!e.message.orEmpty().contains("BUSYGROUP")) {
                logger.error(e) { "consumer_group_creation_failed stream=$streamKey group=$consumerGroup" }
                throw e
            } else {
                logger.info { "consumer_group_exists stream=$streamKey group=$consumerGroup" }
            }
        }

        runCatching { stream.createConsumer(consumerGroup, consumerName).awaitSingleOrNull() }
            .onSuccess {
                logger.info {
                    "consumer_created stream=$streamKey group=$consumerGroup consumer=$consumerName"
                }
            }.onFailure { error ->
                logger.info(error) {
                    "consumer_create_skipped stream=$streamKey group=$consumerGroup consumer=$consumerName"
                }
            }
    }

    /**
     * Consumer 루프 실행
     */
    private suspend fun runConsumerLoop() {
        val stream = getStream()

        while (scope.isActive) {
            semaphore.acquire()

            val messages =
                runCatching { readBatch(stream) }
                    .onFailure { error -> logger.error(error) { "consumer_loop_error" } }
                    .getOrNull()

            if (messages.isNullOrEmpty()) {
                semaphore.release()
                continue
            }

            scope.launch {
                try {
                    processMessages(stream, messages)
                } finally {
                    semaphore.release()
                }
            }
        }
    }

    /**
     * 배치 메시지 처리
     */
    private suspend fun processMessages(
        stream: RStreamReactive<String, String>,
        messages: Map<StreamMessageId, Map<String, String>>,
    ) {
        val ackTargets = mutableListOf<StreamMessageId>()

        messages.forEach { (messageId, fields) ->
            val messageIdString = messageId.toString()

            runCatching {
                messageHandler.handleStreamMessage(messageIdString, fields)
            }.onFailure { error ->
                logger.error(error) { "message_processing_failed message_id=$messageId" }
            }

            ackTargets += messageId
        }

        if (ackTargets.isNotEmpty()) {
            acknowledge(stream, ackTargets)
        }
    }

    private suspend fun readBatch(stream: RStreamReactive<String, String>): Map<StreamMessageId, Map<String, String>> {
        return stream.readGroup(
            consumerGroup,
            consumerName,
            StreamReadGroupArgs.greaterThan(StreamMessageId.NEVER_DELIVERED)
                .count(MQConstants.BATCH_SIZE)
                .timeout(Duration.ofMillis(MQConstants.READ_TIMEOUT_MS)),
        ).awaitSingleOrNull() ?: emptyMap()
    }

    private suspend fun acknowledge(
        stream: RStreamReactive<String, String>,
        messageIds: Collection<StreamMessageId>,
    ) {
        if (messageIds.isEmpty()) return

        runCatching {
            stream.ack(consumerGroup, *messageIds.toTypedArray()).awaitSingleOrNull()
        }.onFailure { error ->
            logger.warn(error) { "message_ack_failed message_ids=${messageIds.joinToString(",")}" }
        }
    }

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
