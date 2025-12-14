package party.qwer.twentyq.service.riddle

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.model.RiddleCategory
import party.qwer.twentyq.model.RiddleSecret
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.rest.TwentyQRestClient
import party.qwer.twentyq.util.game.GameMessageProvider
import tools.jackson.databind.ObjectMapper

/** 수수께끼 생성 및 초기화 서비스 */
@Service
class RiddleCreator(
    private val sessionRepo: RiddleSessionRepository,
    private val topicSelector: TopicSelector,
    private val restClient: TwentyQRestClient,
    private val appProperties: AppProperties,
    private val messageProvider: GameMessageProvider,
    private val objectMapper: ObjectMapper,
) {
    companion object {
        private val log = LoggerFactory.getLogger(RiddleCreator::class.java)
    }

    suspend fun createRiddle(
        chatId: String,
        category: String? = null,
    ): String {
        val parsedCategory = RiddleCategory.fromString(category)
        log.info("createRiddle START chatId={}, category={}", chatId, parsedCategory)

        resumeIfExisting(chatId)?.let { return it }

        val bannedTopics = collectBannedTopics(chatId, parsedCategory)
        val secret = selectSecret(parsedCategory, bannedTopics)

        sessionRepo.saveSecret(chatId = chatId, secret = secret)

        val finalCategory = RiddleCategory.fromString(secret.category)
        if (finalCategory != RiddleCategory.ANY) {
            sessionRepo.saveSelectedCategory(chatId = chatId, category = finalCategory.name)
        }

        // LLM 세션 생성 및 저장
        createLlmSession(chatId)

        log.info(
            "createRiddle SUCCESS chatId={}, target='{}', category='{}', bannedCount={}",
            chatId,
            secret.target,
            secret.category,
            bannedTopics.size,
        )

        return secret.intro.ifBlank { messageProvider.get("start.intro") }
    }

    private suspend fun collectBannedTopics(
        chatId: String,
        requestedCategory: RiddleCategory,
    ): List<String> {
        val limit = appProperties.riddle.game.recentTopicsLimit
        val categoryName = requestedCategory.takeIf { it != RiddleCategory.ANY }?.name
        return sessionRepo.getBannedTopics(chatId, categoryName, limit)
    }

    private suspend fun selectSecret(
        category: RiddleCategory,
        bannedTopics: List<String>,
    ): RiddleSecret {
        val categoryName = if (category == RiddleCategory.ANY) null else category.name.lowercase()
        log.info(
            "selectSecret BEFORE_SELECT category={}, categoryName={}, bannedCount={}",
            category,
            categoryName,
            bannedTopics.size,
        )
        val selectedTopic = topicSelector.selectTopic(categoryName, bannedTopics)

        log.info(
            "selectSecret SELECTED target='{}', category='{}'",
            selectedTopic.name,
            selectedTopic.category,
        )

        return RiddleSecret(
            target = selectedTopic.name,
            category = selectedTopic.category,
            intro = messageProvider.get("start.intro"),
            description = objectMapper.writeValueAsString(selectedTopic.details),
        )
    }

    private suspend fun resumeIfExisting(chatId: String): String? {
        sessionRepo.getQuiz(chatId) ?: return null
        val history = sessionRepo.getHistory(chatId)
        val hintCount = sessionRepo.getHintCount(chatId)

        log.info(
            "createRiddle RESUME_GAME chatId={}, questionCount={}, hintCount={}",
            chatId,
            history.size,
            hintCount,
        )

        return messageProvider.get(
            "start.resume",
            "questionCount" to history.size,
            "hintCount" to hintCount,
        )
    }

    private suspend fun createLlmSession(chatId: String) {
        val response = restClient.createSession(chatId)
        if (response.isError) {
            log.warn("createRiddle LLM_SESSION_FAILED chatId={}, error={}", chatId, response.errorMessage)
        }
    }
}
