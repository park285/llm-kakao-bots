package io.github.kapu.turtlesoup.service

import io.github.kapu.turtlesoup.models.GameState
import io.github.kapu.turtlesoup.redis.LockManager
import io.github.kapu.turtlesoup.redis.SessionStore
import io.github.kapu.turtlesoup.utils.SessionNotFoundException

class GameSessionManager(
    private val sessionStore: SessionStore,
    private val lockManager: LockManager,
) {
    suspend fun <T> withLock(
        sessionId: String,
        holderName: String? = null,
        block: suspend () -> T,
    ): T = lockManager.withLock(sessionId, holderName) { block() }

    suspend fun <T> withOwnerLock(
        sessionId: String,
        block: suspend () -> T,
    ): T {
        val holderName = sessionStore.loadGameState(sessionId)?.userId
        return lockManager.withLock(sessionId, holderName) { block() }
    }

    suspend fun load(sessionId: String): GameState? = sessionStore.loadGameState(sessionId)

    suspend fun loadOrThrow(sessionId: String): GameState = load(sessionId) ?: throw SessionNotFoundException(sessionId)

    suspend fun save(state: GameState) {
        sessionStore.saveGameState(state)
    }

    suspend fun refresh(sessionId: String) {
        sessionStore.refreshTtl(sessionId)
    }

    suspend fun delete(sessionId: String) {
        sessionStore.deleteSession(sessionId)
    }
}
