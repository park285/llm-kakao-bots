package party.qwer.twentyq.redis.session

import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty
import org.springframework.stereotype.Component
import party.qwer.twentyq.api.dto.QuestionHistory
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.redis.RedisKeys
import party.qwer.twentyq.redis.expireAsync
import party.qwer.twentyq.util.common.extensions.minutes
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.kotlinModule
import tools.jackson.module.kotlin.readValue
import java.time.Duration

/**
 * 질문 히스토리 저장소
 */
@Component
@ConditionalOnProperty(
    prefix = "app.cache",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class HistoryStore(
    private val redisson: RedissonReactiveClient,
    private val props: AppProperties,
    @param:org.springframework.beans.factory.annotation.Qualifier("kotlinJsonMapper")
    private val objectMapper: tools.jackson.databind.ObjectMapper,
) {
    companion object {
        private val log = LoggerFactory.getLogger(HistoryStore::class.java)
    }

    suspend fun getAsync(roomId: String): List<QuestionHistory> {
        val list = redisson.getList<String>(key(roomId))
        val range = list.readAll().awaitSingleOrNull() ?: emptyList()
        val history = range.mapNotNull { parseOrNull(it, roomId) }
        log.debugL { "GET room=$roomId, size=${history.size}" }
        return history
    }

    suspend fun add(
        roomId: String,
        questionNumber: Int,
        question: String,
        answer: String,
        isChain: Boolean = false,
        thoughtSignature: String? = null,
        userId: String? = null,
    ) {
        val history = QuestionHistory(questionNumber, question, answer, isChain, thoughtSignature, userId)
        val data = objectMapper.writeValueAsString(history)
        val list = redisson.getList<String>(key(roomId))
        list.add(data).awaitSingleOrNull()
        list.expireAsync(
            props.cache.sessionTtlMinutes.minutes,
        )
        log.debugL { "ADD room=$roomId, qNum=$questionNumber, answer=$answer, isChain=$isChain" }
    }

    suspend fun updateAt(
        roomId: String,
        index: Int,
        questionNumber: Int,
        question: String,
        answer: String,
    ) {
        val history = QuestionHistory(questionNumber, question, answer)
        val data = objectMapper.writeValueAsString(history)
        val list = redisson.getList<String>(key(roomId))
        list.fastSet(index, data).awaitSingleOrNull()
        list.expireAsync(
            props.cache.sessionTtlMinutes.minutes,
        )
        log.debugL { "UPDATE room=$roomId, index=$index, qNum=$questionNumber, answer=$answer" }
    }

    suspend fun clearAsync(roomId: String) {
        redisson.getList<String>(key(roomId)).delete().awaitSingleOrNull()
        log.debugL { "CLEAR room=$roomId" }
    }

    suspend fun setTtlAsync(
        roomId: String,
        ttl: Duration,
    ) {
        redisson.getList<String>(key(roomId)).expireAsync(ttl)
        log.debugL { "TTL room=$roomId, ttl=$ttl" }
    }

    private fun parseOrNull(
        data: String,
        roomId: String,
    ): QuestionHistory? =
        kotlin
            .runCatching { objectMapper.readValue<QuestionHistory>(data) }
            .onFailure { e -> log.warn("PARSE_ERROR room={}, error={}", roomId, e.message) }
            .getOrNull()

    private fun key(roomId: String) = "${RedisKeys.HISTORY}:$roomId"
}
