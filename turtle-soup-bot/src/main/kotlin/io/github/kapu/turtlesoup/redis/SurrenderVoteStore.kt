package io.github.kapu.turtlesoup.redis

import io.github.kapu.turtlesoup.config.JsonConfig
import io.github.kapu.turtlesoup.config.RedisConstants
import io.github.kapu.turtlesoup.config.RedisKeys
import io.github.kapu.turtlesoup.models.SurrenderVote
import io.github.kapu.turtlesoup.utils.RedisException
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.reactor.awaitSingleOrNull
import kotlinx.serialization.SerializationException
import kotlinx.serialization.encodeToString
import org.redisson.api.RedissonReactiveClient
import java.time.Duration
import org.redisson.client.RedisException as RedissonException

/** 항복 투표 저장소 (Reactive) */
class SurrenderVoteStore(
    private val redisson: RedissonReactiveClient,
) {
    private val json = JsonConfig.lenient

    suspend fun get(chatId: String): SurrenderVote? {
        val key = key(chatId)
        return try {
            val bucket = redisson.getBucket<String>(key)
            val value = bucket.get().awaitSingleOrNull() ?: return null
            json.decodeFromString<SurrenderVote>(value)
        } catch (e: RedissonException) {
            logger.error(e) { "vote_load_failed chat_id=$chatId" }
            throw RedisException("Failed to load vote: $chatId", e)
        } catch (e: SerializationException) {
            logger.error(e) { "vote_load_failed chat_id=$chatId" }
            throw RedisException("Failed to load vote: $chatId", e)
        }
    }

    suspend fun save(
        chatId: String,
        vote: SurrenderVote,
    ) {
        val key = key(chatId)
        try {
            val value = json.encodeToString(vote)
            val bucket = redisson.getBucket<String>(key)
            bucket.set(value, Duration.ofSeconds(RedisConstants.VOTE_TTL_SECONDS)).awaitSingleOrNull()
            logger.debug { "vote_saved chat_id=$chatId approvals=${vote.approvals.size}" }
        } catch (e: RedissonException) {
            logger.error(e) { "vote_save_failed chat_id=$chatId" }
            throw RedisException("Failed to save vote: $chatId", e)
        } catch (e: SerializationException) {
            logger.error(e) { "vote_save_failed chat_id=$chatId" }
            throw RedisException("Failed to save vote: $chatId", e)
        }
    }

    suspend fun approve(
        chatId: String,
        userId: String,
    ): SurrenderVote? {
        val vote = get(chatId) ?: return null
        val updated = vote.approve(userId)
        save(chatId, updated)
        logger.debug {
            "vote_approved chat_id=$chatId user_id=$userId " +
                "approvals=${updated.approvals.size}/${updated.requiredApprovals()}"
        }
        return updated
    }

    suspend fun clear(chatId: String) {
        val key = key(chatId)
        try {
            val bucket = redisson.getBucket<String>(key)
            bucket.delete().awaitSingleOrNull()
            logger.debug { "vote_cleared chat_id=$chatId" }
        } catch (e: RedissonException) {
            logger.error(e) { "vote_clear_failed chat_id=$chatId" }
        }
    }

    private fun key(chatId: String) = "${RedisKeys.SURRENDER_VOTE}:$chatId"

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
