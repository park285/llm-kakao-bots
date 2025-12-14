package io.github.kapu.turtlesoup.redis

import io.github.kapu.turtlesoup.config.JsonConfig
import io.github.kapu.turtlesoup.config.RedisConstants
import io.github.kapu.turtlesoup.config.RedisKeys
import io.github.kapu.turtlesoup.models.GameState
import io.github.kapu.turtlesoup.utils.RedisException
import io.github.oshai.kotlinlogging.KotlinLogging
import kotlinx.coroutines.reactor.awaitSingle
import kotlinx.coroutines.reactor.awaitSingleOrNull
import kotlinx.serialization.SerializationException
import kotlinx.serialization.encodeToString
import org.redisson.api.RedissonReactiveClient
import java.time.Duration
import org.redisson.client.RedisException as RedissonException

class SessionStore(
    private val redisson: RedissonReactiveClient,
) {
    private val json = JsonConfig.lenient

    suspend fun saveGameState(state: GameState) {
        val key = "${RedisKeys.SESSION}:${state.sessionId}"
        try {
            val value = json.encodeToString(state)
            val bucket = redisson.getBucket<String>(key)
            bucket.set(value, Duration.ofSeconds(RedisConstants.SESSION_TTL_SECONDS)).awaitSingleOrNull()

            logger.info { "game_state_saved session_id=${state.sessionId}" }
        } catch (e: RedissonException) {
            logger.error(e) { "game_state_save_failed session_id=${state.sessionId}" }
            throw RedisException("Failed to save game state: ${state.sessionId}", e)
        } catch (e: SerializationException) {
            logger.error(e) { "game_state_save_failed session_id=${state.sessionId}" }
            throw RedisException("Failed to save game state: ${state.sessionId}", e)
        }
    }

    suspend fun loadGameState(sessionId: String): GameState? {
        val key = "${RedisKeys.SESSION}:$sessionId"
        return try {
            val bucket = redisson.getBucket<String>(key)
            val value = bucket.get().awaitSingleOrNull() ?: return null
            json.decodeFromString<GameState>(value)
        } catch (e: RedissonException) {
            logger.error(e) { "game_state_load_failed session_id=$sessionId" }
            throw RedisException("Failed to load game state: $sessionId", e)
        } catch (e: SerializationException) {
            logger.error(e) { "game_state_load_failed session_id=$sessionId" }
            throw RedisException("Failed to load game state: $sessionId", e)
        }
    }

    suspend fun deleteSession(sessionId: String) {
        val sessionKey = "${RedisKeys.SESSION}:$sessionId"
        try {
            val bucket = redisson.getBucket<String>(sessionKey)
            bucket.delete().awaitSingleOrNull()
            logger.info { "session_deleted session_id=$sessionId" }
        } catch (e: RedissonException) {
            logger.error(e) { "session_delete_failed session_id=$sessionId" }
            throw RedisException("Failed to delete session: $sessionId", e)
        }
    }

    suspend fun sessionExists(sessionId: String): Boolean {
        val key = "${RedisKeys.SESSION}:$sessionId"
        return try {
            val bucket = redisson.getBucket<String>(key)
            bucket.isExists.awaitSingle()
        } catch (e: RedissonException) {
            logger.error(e) { "session_exists_check_failed session_id=$sessionId" }
            throw RedisException("Failed to check session existence: $sessionId", e)
        }
    }

    suspend fun refreshTtl(sessionId: String): Boolean {
        val sessionKey = "${RedisKeys.SESSION}:$sessionId"
        return try {
            val bucket = redisson.getBucket<String>(sessionKey)
            val result = bucket.expire(Duration.ofSeconds(RedisConstants.SESSION_TTL_SECONDS)).awaitSingle()
            if (result) {
                logger.debug { "ttl_refreshed session_id=$sessionId" }
            }
            result
        } catch (e: RedissonException) {
            logger.error(e) { "ttl_refresh_failed session_id=$sessionId" }
            throw RedisException("Failed to refresh TTL: $sessionId", e)
        }
    }

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
