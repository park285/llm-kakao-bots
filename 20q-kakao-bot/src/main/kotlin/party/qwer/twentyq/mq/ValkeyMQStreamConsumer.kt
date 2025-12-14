package party.qwer.twentyq.mq

import jakarta.annotation.PostConstruct
import jakarta.annotation.PreDestroy
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Semaphore
import kotlinx.coroutines.sync.withPermit
import kotlinx.coroutines.withTimeoutOrNull
import org.redisson.api.RStreamReactive
import org.redisson.api.RedissonReactiveClient
import org.redisson.api.StreamMessageId
import org.redisson.api.stream.StreamCreateGroupArgs
import org.redisson.api.stream.StreamReadGroupArgs
import org.redisson.api.stream.StreamTrimArgs
import org.redisson.client.RedisException
import org.redisson.client.codec.StringCodec
import org.slf4j.Logger
import org.slf4j.LoggerFactory
import org.springframework.beans.factory.annotation.Qualifier
import org.springframework.context.annotation.Profile
import org.springframework.scheduling.annotation.Scheduled
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.config.properties.ValkeyMQ
import party.qwer.twentyq.redis.awaitSingleOrNull
import party.qwer.twentyq.util.common.extensions.seconds
import java.time.Duration

@Component
@Profile("!test")
class ValkeyMQStreamConsumer(
    private val appProperties: AppProperties,
    @param:Qualifier("redissonMQReactiveClient")
    private val redissonMQReactiveClient: RedissonReactiveClient,
    private val messageHandler: ValkeyMQMessageHandler,
) {
    companion object {
        private val log = LoggerFactory.getLogger(ValkeyMQStreamConsumer::class.java)
        private const val EMPTY_SLEEP_MILLIS: Long = 200
        private const val ERROR_SLEEP_MILLIS: Long = 1000
        private const val BATCH_SIZE: Int = 10
        private const val MAX_CONCURRENT_PROCESSING: Int = 10
        private val READ_TIMEOUT: Duration = 5.seconds
        private const val MAX_MESSAGES: Long = 10000

        // Pending 메시지 cleanup 타임아웃
        private const val PENDING_CLEANUP_TIMEOUT_SECONDS: Long = 10

        // 오래된 메시지 정리 주기 (1시간)
        private const val TRIM_INTERVAL_MS: Long = 3600000

        // ACK 재시도 설정
        private const val ACK_MAX_RETRIES: Int = 3
        private const val ACK_RETRY_DELAY_MS: Long = 100

        // Graceful shutdown 설정
        private const val SHUTDOWN_TIMEOUT_MS: Long = 30_000
        private const val SHUTDOWN_POLL_INTERVAL_MS: Long = 100
    }

    private val scope =
        CoroutineScope(
            Dispatchers.IO + SupervisorJob(),
        )

    // LLM backpressure: 동시 처리 제한
    private val processingLimiter = Semaphore(MAX_CONCURRENT_PROCESSING)
    private val groupManager =
        ValkeyGroupManager(
            appProperties,
            redissonMQReactiveClient,
            PENDING_CLEANUP_TIMEOUT_SECONDS,
        )

    @PostConstruct
    fun start() {
        scope.launch {
            groupManager.clearPendingMessagesOnStartup()
            runConsumerLoop()
        }
    }

    @PreDestroy
    fun stop() {
        log.info("VALKEY_MQ_CONSUMER_SHUTDOWN_INITIATED inProgress={}", inProgressCount())
        scope.cancel()

        CoroutineScope(Dispatchers.IO + SupervisorJob()).launch {
            val completed =
                withTimeoutOrNull(SHUTDOWN_TIMEOUT_MS) {
                    while (inProgressCount() > 0) {
                        log.info("VALKEY_MQ_CONSUMER_SHUTDOWN_WAITING inProgress={}", inProgressCount())
                        delay(SHUTDOWN_POLL_INTERVAL_MS)
                    }
                    true
                } ?: false

            if (completed) {
                log.info("VALKEY_MQ_CONSUMER_SHUTDOWN_COMPLETED")
            } else {
                log.warn(
                    "VALKEY_MQ_CONSUMER_SHUTDOWN_TIMEOUT remainingTasks={}",
                    inProgressCount(),
                )
            }
        }
    }

    private fun inProgressCount(): Int = MAX_CONCURRENT_PROCESSING - processingLimiter.availablePermits

    // 오래된 메시지 정리
    @Scheduled(fixedRate = TRIM_INTERVAL_MS)
    fun trimOldMessages() {
        scope.launch {
            runCatching {
                val mq = appProperties.mq
                val stream: RStreamReactive<String, String> =
                    redissonMQReactiveClient.getStream(
                        mq.streamKey,
                        StringCodec.INSTANCE,
                    )

                val removed =
                    stream
                        .trimNonStrict(
                            StreamTrimArgs
                                .maxLen(MAX_MESSAGES.toInt())
                                .noLimit(),
                        ).awaitSingleOrNull() ?: 0L

                log.info(
                    "VALKEY_MQ_TRIM_COMPLETED streamKey={}, maxMessages={}, removed={}",
                    mq.streamKey,
                    MAX_MESSAGES,
                    removed,
                )
            }.onFailure { ex ->
                log.error(
                    "VALKEY_MQ_TRIM_ERROR error={}",
                    ex.message,
                    ex,
                )
            }
        }
    }

    private suspend fun runConsumerLoop() {
        val mq = appProperties.mq
        val stream: RStreamReactive<String, String> =
            redissonMQReactiveClient.getStream(
                mq.streamKey,
                StringCodec.INSTANCE,
            )

        groupManager.ensureGroupAndConsumer(stream, mq.streamKey, mq.consumerGroup, mq.consumerName)

        log.info(
            "VALKEY_MQ_CONSUMER_STARTED streamKey={}, groupName={}, consumerName={}",
            mq.streamKey,
            mq.consumerGroup,
            mq.consumerName,
        )

        while (scope.isActive) {
            val failure =
                runCatching {
                    processMessageBatch(stream, mq.streamKey, mq.consumerGroup, mq.consumerName)
                }.exceptionOrNull()

            when (failure) {
                null -> Unit
                is CancellationException -> {
                    log.info("VALKEY_MQ_CONSUMER_CANCELLED streamKey={}", mq.streamKey)
                    throw failure
                }

                else -> {
                    handleConsumerLoopError(failure, stream, mq)
                    delay(ERROR_SLEEP_MILLIS)
                }
            }
        }

        log.info("VALKEY_MQ_CONSUMER_STOPPED streamKey={}", mq.streamKey)
    }

    // 소비자 루프 에러 핸들러
    private suspend fun handleConsumerLoopError(
        ex: Throwable,
        stream: RStreamReactive<String, String>,
        mq: ValkeyMQ,
    ) {
        if (ex is RedisException && ex.message?.contains("NOGROUP") == true) {
            log.warn(
                "VALKEY_MQ_CONSUMER_NOGROUP streamKey={}, groupName={}, consumerName={}, error={}",
                mq.streamKey,
                mq.consumerGroup,
                mq.consumerName,
                ex.message,
            )

            // NOGROUP 에러 복구 시도
            runCatching {
                groupManager.ensureGroupAndConsumer(stream, mq.streamKey, mq.consumerGroup, mq.consumerName)
            }.onFailure { retryEx ->
                log.error(
                    "VALKEY_MQ_CONSUMER_NOGROUP_RECOVER_FAILED streamKey={}, groupName={}, consumerName={}, error={}",
                    mq.streamKey,
                    mq.consumerGroup,
                    mq.consumerName,
                    retryEx.message,
                    retryEx,
                )
            }
        } else {
            log.error(
                "VALKEY_MQ_CONSUMER_ERROR streamKey={}, error={}",
                mq.streamKey,
                ex.message,
                ex,
            )
        }
    }

    /**
     * Stream에서 메시지 읽기 및 타입 검증
     * - Redisson API 버그 방어: emptyList 반환 시 null 반환
     */
    private suspend fun readMessagesFromStream(
        stream: RStreamReactive<String, String>,
        groupName: String,
        consumerName: String,
        args: StreamReadGroupArgs,
        streamKey: String,
    ): Map<StreamMessageId, Map<String, String>>? {
        val result: Any? = stream.readGroup(groupName, consumerName, args).awaitSingleOrNull()

        if (result == null) return null

        if (result !is Map<*, *>) {
            log.warn(
                "VALKEY_MQ_UNEXPECTED_TYPE streamKey={}, type={}, result={}",
                streamKey,
                result::class.simpleName,
                result,
            )
            return null
        }

        return convertToTypedMap(result, streamKey)
    }

    /** Map<*, *>를 Map<StreamMessageId, Map<String, String>>으로 안전하게 변환 */
    private fun convertToTypedMap(
        rawMap: Map<*, *>,
        streamKey: String,
    ): Map<StreamMessageId, Map<String, String>>? {
        val converted = mutableMapOf<StreamMessageId, Map<String, String>>()

        for ((key, value) in rawMap) {
            val messageId = key as? StreamMessageId
            val fields =
                (value as? Map<*, *>)?.let { fieldsMap ->
                    fieldsMap.entries.associate { (k, v) ->
                        (k?.toString() ?: "") to (v?.toString() ?: "")
                    }
                }

            if (messageId == null || fields == null) {
                log.warn(
                    "VALKEY_MQ_INVALID_ENTRY streamKey={}, keyType={}, valueType={}",
                    streamKey,
                    key?.javaClass?.simpleName,
                    value?.javaClass?.simpleName,
                )
                continue
            }

            converted[messageId] = fields
        }

        return converted.ifEmpty { null }
    }

    /**
     * 개별 메시지 처리: handleMessage 호출 및 ack
     */
    private fun processIndividualMessage(
        stream: RStreamReactive<String, String>,
        streamKey: String,
        groupName: String,
        id: StreamMessageId,
        fields: Map<String, String>,
    ) {
        scope.launch {
            processingLimiter.withPermit {
                val handleResult =
                    runCatching {
                        messageHandler.handleMessage(streamKey, id, fields)
                    }

                handleResult.onFailure { ex ->
                    log.error(
                        "VALKEY_MQ_HANDLE_ERROR streamKey={}, id={}, error={}",
                        streamKey,
                        id,
                        ex.message,
                        ex,
                    )
                    return@withPermit
                }

                // 메시지 처리 성공 시 ACK 재시도
                ackWithRetry(stream, groupName, streamKey, id)
            }
        }
    }

    /** ACK 재시도 로직 */
    private suspend fun ackWithRetry(
        stream: RStreamReactive<String, String>,
        groupName: String,
        streamKey: String,
        id: StreamMessageId,
    ) {
        repeat(ACK_MAX_RETRIES) { attempt ->
            val ackResult =
                runCatching {
                    stream.ack(groupName, id).awaitSingleOrNull()
                }

            if (ackResult.isSuccess) return

            log.warn(
                "VALKEY_MQ_ACK_RETRY streamKey={}, id={}, attempt={}, error={}",
                streamKey,
                id,
                attempt + 1,
                ackResult.exceptionOrNull()?.message,
            )

            if (attempt < ACK_MAX_RETRIES - 1) {
                delay(ACK_RETRY_DELAY_MS)
            }
        }

        // ACK 실패 → pending 유지. 단, 재시작 시 group 리셋 정책으로 pending은 정리됨(메시지 폭주 방지 목적)
        log.error(
            "VALKEY_MQ_ACK_FAILED streamKey={}, id={}, maxRetries={}, action=WILL_RETRY_ON_RESTART",
            streamKey,
            id,
            ACK_MAX_RETRIES,
        )
    }

    private suspend fun processMessageBatch(
        stream: RStreamReactive<String, String>,
        streamKey: String,
        groupName: String,
        consumerName: String,
    ) {
        val args =
            StreamReadGroupArgs
                .neverDelivered()
                .count(BATCH_SIZE)
                .timeout(READ_TIMEOUT)

        try {
            val messages =
                readMessagesFromStream(stream, groupName, consumerName, args, streamKey)
                    ?: run {
                        delay(EMPTY_SLEEP_MILLIS)
                        return
                    }

            if (messages.isEmpty()) {
                delay(EMPTY_SLEEP_MILLIS)
                return
            }

            messages.forEach { (id, fields) ->
                processIndividualMessage(stream, streamKey, groupName, id, fields)
            }
        } catch (ex: ClassCastException) {
            log.error(
                "VALKEY_MQ_TYPE_CAST_ERROR streamKey={}, error={}",
                streamKey,
                ex.message,
                ex,
            )
            delay(EMPTY_SLEEP_MILLIS)
        }
    }
}
