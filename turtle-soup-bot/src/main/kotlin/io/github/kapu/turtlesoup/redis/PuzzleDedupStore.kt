package io.github.kapu.turtlesoup.redis

import io.github.kapu.turtlesoup.config.PuzzleDedupConstants
import io.github.kapu.turtlesoup.config.RedisKeys
import io.github.kapu.turtlesoup.utils.RedisException
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.reactor.awaitSingle
import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RedissonReactiveClient
import org.redisson.client.codec.StringCodec
import java.time.Duration

class PuzzleDedupStore(
    private val redisson: RedissonReactiveClient,
) {
    private val codec = StringCodec.INSTANCE

    suspend fun isDuplicate(
        signature: String,
        chatId: String,
    ): Boolean {
        val globalKey = RedisKeys.PUZZLE_GLOBAL
        val chatKey = "${RedisKeys.PUZZLE_CHAT}:$chatId"
        return try {
            val globalExists =
                redisson.getSet<String>(globalKey, codec)
                    .contains(signature)
                    .awaitSingleOrNull() ?: false
            val chatExists =
                redisson.getSet<String>(chatKey, codec)
                    .contains(signature)
                    .awaitSingleOrNull() ?: false
            globalExists || chatExists
        } catch (e: org.redisson.client.RedisException) {
            throw RedisException("Failed to check puzzle dedup", e)
        } catch (e: IllegalStateException) {
            throw RedisException("Failed to check puzzle dedup", e)
        }
    }

    suspend fun markUsed(
        signature: String,
        chatId: String,
    ) {
        val globalKey = RedisKeys.PUZZLE_GLOBAL
        val chatKey = "${RedisKeys.PUZZLE_CHAT}:$chatId"
        try {
            val globalSet = redisson.getSet<String>(globalKey, codec)
            val chatSet = redisson.getSet<String>(chatKey, codec)

            globalSet.add(signature).awaitSingle()
            chatSet.add(signature).awaitSingle()

            globalSet.expire(Duration.ofSeconds(PuzzleDedupConstants.GLOBAL_TTL_SECONDS)).awaitSingleOrNull()
            chatSet.expire(Duration.ofSeconds(PuzzleDedupConstants.CHAT_TTL_SECONDS)).awaitSingleOrNull()

            logger.info { "puzzle_dedup_marked chat_id=$chatId" }
        } catch (e: org.redisson.client.RedisException) {
            throw RedisException("Failed to mark puzzle dedup", e)
        } catch (e: IllegalStateException) {
            throw RedisException("Failed to mark puzzle dedup", e)
        }
    }

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
