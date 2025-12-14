package io.github.kapu.turtlesoup.service

import io.github.kapu.turtlesoup.redis.SessionStore
import io.github.kapu.turtlesoup.utils.SessionNotFoundException

class SessionValidator(
    private val sessionStore: SessionStore,
) {
    suspend fun requireSession(chatId: String) {
        if (!sessionStore.sessionExists(chatId)) {
            throw SessionNotFoundException(chatId)
        }
    }
}
