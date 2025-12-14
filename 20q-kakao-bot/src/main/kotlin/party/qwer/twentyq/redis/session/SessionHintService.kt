package party.qwer.twentyq.redis.session

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.redis.tracking.CandidateCountStore
import party.qwer.twentyq.redis.tracking.HintCountStore

/** 세션 힌트 관리 서비스 */
@Component
class SessionHintService(
    private val hintCountStore: HintCountStore,
    private val candidateCountStore: CandidateCountStore,
) : HintSessionOperations {
    companion object {
        private val log = LoggerFactory.getLogger(SessionHintService::class.java)
    }

    override suspend fun getHintCount(chatId: String): Int {
        val count = hintCountStore.getAsync(chatId)
        log.debugL { "VALKEY getHintCount chatId=$chatId count=$count" }
        return count
    }

    override suspend fun incrementHintCount(chatId: String) {
        hintCountStore.increment(chatId)
        log.debugL { "VALKEY incrementHintCount chatId=$chatId" }
    }

    override suspend fun saveCandidateCount(
        chatId: String,
        count: Int,
    ) {
        candidateCountStore.saveAsync(chatId, count)
        log.debugL { "VALKEY saveCandidateCount chatId=$chatId count=$count" }
    }

    override suspend fun getCandidateCount(chatId: String): Int = candidateCountStore.getAsync(chatId)
}
