package party.qwer.twentyq.redis.session

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.api.dto.QuestionHistory
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.RiddleSecret

/**
 * 게임 핵심 데이터 관리: 퀴즈/비밀/히스토리/카테고리
 */
@Component
class SessionGameService(
    private val sessionStore: SessionStore,
    private val secretStore: SecretStore,
    private val historyStore: HistoryStore,
    private val categoryStore: CategoryStore,
) : GameSessionOperations {
    companion object {
        private val log = LoggerFactory.getLogger(SessionGameService::class.java)
    }

    override suspend fun getQuiz(chatId: String): String? {
        val quiz = sessionStore.getAsync(chatId)
        log.debugL {
            "VALKEY getQuiz chatId=$chatId exists=${quiz != null} size=${quiz?.length ?: 0}"
        }
        return quiz
    }

    override suspend fun getSecret(chatId: String): RiddleSecret? = secretStore.getAsync(chatId)

    override suspend fun saveSecret(
        chatId: String,
        secret: RiddleSecret,
    ) {
        secretStore.saveAsync(chatId, secret)
        log.debugL {
            "VALKEY saveSecret chatId=$chatId target='${secret.target}', category='${secret.category}'"
        }
    }

    override suspend fun getHistory(chatId: String): List<QuestionHistory> {
        val history = historyStore.getAsync(chatId)
        log.debugL { "VALKEY getHistory chatId=$chatId size=${history.size}" }
        return history
    }

    override suspend fun addHistory(
        chatId: String,
        questionNumber: Int,
        question: String,
        answer: String,
        isChain: Boolean,
        thoughtSignature: String?,
        userId: String?,
    ) {
        historyStore.add(chatId, questionNumber, question, answer, isChain, thoughtSignature, userId)
        log.debugL {
            "VALKEY addHistory chatId=$chatId questionNumber=$questionNumber answer=$answer isChain=$isChain"
        }
    }

    override suspend fun updateHistoryAt(
        chatId: String,
        index: Int,
        questionNumber: Int,
        question: String,
        answer: String,
    ) {
        historyStore.updateAt(chatId, index, questionNumber, question, answer)
        log.debugL {
            "VALKEY updateHistoryAt chatId=$chatId index=$index questionNumber=$questionNumber answer=$answer"
        }
    }

    override suspend fun getSelectedCategory(chatId: String): String? {
        val category = categoryStore.getAsync(chatId)
        log.debugL { "VALKEY getSelectedCategory chatId=$chatId category=$category" }
        return category
    }

    override suspend fun saveSelectedCategory(
        chatId: String,
        category: String?,
    ) {
        categoryStore.saveAsync(chatId, category)
        log.debugL { "VALKEY saveSelectedCategory chatId=$chatId category=$category" }
    }
}
