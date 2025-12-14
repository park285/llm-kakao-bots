package party.qwer.twentyq.bridge.handlers

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.model.RiddleCategory
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.util.game.GameMessageProvider

@Component
class StartHandler(
    private val riddleService: RiddleService,
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private val log = LoggerFactory.getLogger(StartHandler::class.java)
        private const val MSG_PARAM_CATEGORY = "category"
        private const val EASTER_EGG_1_PROBABILITY = 0.05
        private const val EASTER_EGG_2_PROBABILITY = 0.10
    }

    suspend fun handle(
        chatId: String,
        categories: List<String>? = null,
    ): String {
        log.info("HANDLE_START chatId={}, categories={}", chatId, categories)

        // 기존 세션이 있으면 resume
        if (riddleService.hasSession(chatId)) {
            log.info("EXISTING_SESSION_FOUND chatId={}, resuming game", chatId)
            val status = riddleService.getStatus(chatId)
            return messageProvider.get(
                "start.resume",
                "questionCount" to status.questionCount,
                "hintCount" to status.hintCount,
            )
        }

        // 새 게임 시작: 복수 카테고리 중 랜덤 선택
        val parsedCategory = parseCategories(categories, chatId)
        riddleService.createRiddle(chatId, parsedCategory)

        val selectedCategory = riddleService.getStatus(chatId).selectedCategory
        return buildStartMessage(selectedCategory, parsedCategory, chatId)
    }

    suspend fun hasExistingSession(chatId: String): Boolean = riddleService.hasSession(chatId)

    /**
     * 복수 카테고리 중 유효한 것을 필터링하고 랜덤 선택
     */
    private fun parseCategories(
        categories: List<String>?,
        chatId: String,
    ): String? {
        if (categories.isNullOrEmpty()) return null

        val allowedCategories =
            RiddleCategory.entries
                .filter { it != RiddleCategory.ANY }
                .map { it.koreanName }
                .toSet()

        val validCategories = categories.filter { allowedCategories.contains(it) }

        return when {
            validCategories.isEmpty() -> {
                log.warn("ALL_CATEGORIES_INVALID chatId={}, requested={}", chatId, categories)
                null
            }
            validCategories.size == 1 -> {
                log.info("SINGLE_CATEGORY_SELECTED chatId={}, category={}", chatId, validCategories.first())
                validCategories.first()
            }
            else -> {
                val selected = validCategories.random()
                log.info(
                    "RANDOM_CATEGORY_SELECTED chatId={}, candidates={}, selected={}",
                    chatId,
                    validCategories,
                    selected,
                )
                selected
            }
        }
    }

    private fun buildStartMessage(
        selectedCategory: String?,
        requestedCategory: String?,
        chatId: String,
    ): String {
        val categoryText =
            selectedCategory?.let {
                val categoryEnum = RiddleCategory.fromString(it)
                messageProvider.get("start.category_prefix", "category" to categoryEnum.koreanName)
            } ?: ""

        val invalidWarning =
            if (requestedCategory != null && selectedCategory == null) {
                messageProvider.get("start.invalid_category_warning")
            } else {
                ""
            }

        val rand = Math.random()
        val baseMessage =
            when {
                rand < EASTER_EGG_1_PROBABILITY -> {
                    log.info("EASTER_EGG_1_TRIGGERED chatId={}", chatId)
                    messageProvider.get("start.easter_egg_1", MSG_PARAM_CATEGORY to categoryText)
                }
                rand < EASTER_EGG_2_PROBABILITY -> {
                    log.info("EASTER_EGG_2_TRIGGERED chatId={}", chatId)
                    messageProvider.get("start.easter_egg_2", MSG_PARAM_CATEGORY to categoryText)
                }
                selectedCategory != null -> {
                    messageProvider.get("start.ready_with_category", MSG_PARAM_CATEGORY to categoryText)
                }
                else -> {
                    messageProvider.get("start.ready")
                }
            }

        return invalidWarning + baseMessage
    }
}
