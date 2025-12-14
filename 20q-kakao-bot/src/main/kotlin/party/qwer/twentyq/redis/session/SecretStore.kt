package party.qwer.twentyq.redis.session

import org.slf4j.LoggerFactory
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.RiddleSecret
import tools.jackson.databind.ObjectMapper
import tools.jackson.module.kotlin.readValue

/**
 * 정답 데이터 저장소
 */
@Component
@ConditionalOnProperty(
    prefix = "app.redis",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class SecretStore(
    private val sessionStore: SessionStore,
    private val objectMapper: ObjectMapper,
) {
    companion object {
        private val log = LoggerFactory.getLogger(SecretStore::class.java)
    }

    suspend fun getAsync(roomId: String): RiddleSecret? {
        val data = sessionStore.getAsync(roomId) ?: return null
        return kotlin
            .runCatching { objectMapper.readValue<RiddleSecret>(data) }
            .onFailure { e -> log.error("PARSE_ERROR room={}, error={}", roomId, e.message, e) }
            .getOrNull()
    }

    suspend fun saveAsync(
        roomId: String,
        secret: RiddleSecret,
    ) {
        val data = objectMapper.writeValueAsString(secret)
        sessionStore.saveAsync(roomId, data)
        log.debugL { "SAVE room=$roomId, target='${secret.target}', category='${secret.category}'" }
    }
}
