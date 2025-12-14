package party.qwer.twentyq.mq

import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.withTimeout
import org.redisson.api.RStreamReactive
import org.redisson.api.RedissonReactiveClient
import org.redisson.api.StreamMessageId
import org.redisson.api.stream.StreamCreateGroupArgs
import org.redisson.client.RedisException
import org.redisson.client.codec.StringCodec
import org.slf4j.LoggerFactory
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.config.properties.ValkeyMQ
import party.qwer.twentyq.redis.awaitSingleOrNull
import party.qwer.twentyq.util.common.extensions.seconds

/** Valkey Stream Consumer Group 관리 */
internal class ValkeyGroupManager(
    private val appProperties: AppProperties,
    private val redissonMQReactiveClient: RedissonReactiveClient,
    private val pendingCleanupTimeoutSeconds: Long,
) {
    companion object {
        private val log = LoggerFactory.getLogger(ValkeyGroupManager::class.java)
    }

    suspend fun clearPendingMessagesOnStartup() {
        val mq = appProperties.mq
        val stream: RStreamReactive<String, String> =
            redissonMQReactiveClient.getStream(mq.streamKey, StringCodec.INSTANCE)

        runCatching {
            withTimeout(pendingCleanupTimeoutSeconds.seconds.toMillis()) {
                removeGroupAndRecreate(stream, mq)
            }
        }.onFailure { ex -> handlePendingCleanupFailure(ex, mq) }
    }

    suspend fun ensureGroupAndConsumer(
        stream: RStreamReactive<String, String>,
        streamKey: String,
        groupName: String,
        consumerName: String,
    ) {
        createGroupIfNeeded(stream, streamKey, groupName)
        createConsumerIfNeeded(stream, streamKey, groupName, consumerName)
    }

    private suspend fun removeGroupAndRecreate(
        stream: RStreamReactive<String, String>,
        mq: ValkeyMQ,
    ) {
        // 재시작 시 pending 재처리 폭주/이상동작 방지 목적: group 리셋으로 pending 폐기
        stream.removeGroup(mq.consumerGroup).awaitSingleOrNull()

        log.info(
            "PENDING_CLEANUP_SUCCESS streamKey={}, groupName={} - group removed, pending messages cleared",
            mq.streamKey,
            mq.consumerGroup,
        )

        ensureGroupAndConsumer(stream, mq.streamKey, mq.consumerGroup, mq.consumerName)
    }

    private fun handlePendingCleanupFailure(
        ex: Throwable,
        mq: ValkeyMQ,
    ) {
        when (ex) {
            is TimeoutCancellationException ->
                log.warn(
                    "PENDING_CLEANUP_TIMEOUT streamKey={}, groupName={}, timeout={}s",
                    mq.streamKey,
                    mq.consumerGroup,
                    pendingCleanupTimeoutSeconds,
                )

            is RedisException ->
                log.warn(
                    "PENDING_CLEANUP_SKIPPED streamKey={}, groupName={}, redisError={}",
                    mq.streamKey,
                    mq.consumerGroup,
                    ex.message,
                )

            else ->
                log.warn(
                    "PENDING_CLEANUP_SKIPPED streamKey={}, groupName={}, reason={}",
                    mq.streamKey,
                    mq.consumerGroup,
                    ex.message,
                )
        }
    }

    private suspend fun createGroupIfNeeded(
        stream: RStreamReactive<String, String>,
        streamKey: String,
        groupName: String,
    ) {
        runCatching {
            stream
                .createGroup(
                    StreamCreateGroupArgs
                        .name(groupName)
                        .makeStream()
                        .id(StreamMessageId.NEWEST),
                ).awaitSingleOrNull()
        }.onSuccess {
            log.info("VALKEY_MQ_GROUP_CREATED streamKey={}, groupName={}", streamKey, groupName)
        }.onFailure { ex ->
            log.info(
                "VALKEY_MQ_GROUP_CREATE_SKIPPED streamKey={}, groupName={}, reason={}",
                streamKey,
                groupName,
                ex.message,
            )
        }
    }

    private suspend fun createConsumerIfNeeded(
        stream: RStreamReactive<String, String>,
        streamKey: String,
        groupName: String,
        consumerName: String,
    ) {
        runCatching { stream.createConsumer(groupName, consumerName).awaitSingleOrNull() }
            .onSuccess {
                log.info(
                    "VALKEY_MQ_CONSUMER_CREATED streamKey={}, groupName={}, consumerName={}",
                    streamKey,
                    groupName,
                    consumerName,
                )
            }.onFailure { ex ->
                log.info(
                    "VALKEY_MQ_CONSUMER_CREATE_SKIPPED streamKey={}, groupName={}, consumerName={}, reason={}",
                    streamKey,
                    groupName,
                    consumerName,
                    ex.message,
                )
            }
    }
}
