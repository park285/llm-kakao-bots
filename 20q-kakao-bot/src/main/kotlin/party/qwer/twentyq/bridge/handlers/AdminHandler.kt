package party.qwer.twentyq.bridge.handlers

import jakarta.annotation.PreDestroy
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.redis.RiddleSessionRepository
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.service.riddle.MetaQuestionValidator
import party.qwer.twentyq.util.common.security.requireAdminOrThrow
import party.qwer.twentyq.util.common.security.requireSessionOrThrow
import party.qwer.twentyq.util.game.GameMessageProvider

@Component
class AdminHandler(
    private val appProperties: AppProperties,
    private val riddleService: RiddleService,
    private val sessionRepo: RiddleSessionRepository,
    private val metaQuestionValidator: MetaQuestionValidator,
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private val log = LoggerFactory.getLogger(AdminHandler::class.java)
        private const val RESTART_DELAY_MS = 3000L
    }

    private val adminScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    @PreDestroy
    fun cleanup() {
        adminScope.cancel()
    }

    suspend fun forceEnd(
        chatId: String,
        userId: String,
    ): String {
        log.info("HANDLE_ADMIN_FORCE_END chatId={}, userId={}", chatId, userId)
        requireAdminOrThrow(
            adminUserIds = appProperties.admin.userIds,
            userId = userId,
            chatId = chatId,
            logger = log,
            warnMessage = "ADMIN_PERMISSION_DENIED userId={}, chatId={}",
            warnArgs = arrayOf(userId, chatId),
            errorMessage = messageProvider.get("error.no_permission"),
        )
        riddleService.requireSessionOrThrow(chatId)

        val result = riddleService.surrender(chatId)
        log.info("ADMIN_FORCE_END_SUCCESS chatId={}, adminId={}", chatId, userId)

        return messageProvider.get("admin.force_end_prefix") + result
    }

    suspend fun clearAll(
        chatId: String,
        userId: String,
    ): String {
        log.info("HANDLE_ADMIN_CLEAR_ALL chatId={}, userId={}", chatId, userId)

        requireAdminOrThrow(
            adminUserIds = appProperties.admin.userIds,
            userId = userId,
            chatId = chatId,
            logger = log,
            warnMessage = "ADMIN_PERMISSION_DENIED userId={}, chatId={}",
            warnArgs = arrayOf(userId, chatId),
            errorMessage = messageProvider.get("error.no_permission"),
        )

        sessionRepo.clearAllData(chatId)
        log.info("ADMIN_CLEAR_ALL_SUCCESS chatId={}, adminId={}", chatId, userId)

        return messageProvider.get("admin.clear_all_success")
    }

    suspend fun refreshCache(
        chatId: String,
        userId: String,
    ): String {
        log.info("HANDLE_ADMIN_REFRESH_CACHE chatId={}, userId={}", chatId, userId)
        requireAdminOrThrow(
            adminUserIds = appProperties.admin.userIds,
            userId = userId,
            chatId = chatId,
            logger = log,
            warnMessage = "ADMIN_PERMISSION_DENIED userId={}, chatId={}",
            warnArgs = arrayOf(userId, chatId),
            errorMessage = messageProvider.get("error.no_permission"),
        )

        val guardSuccess = metaQuestionValidator.refreshCache()

        log.info(
            "ADMIN_REFRESH_CACHE_SUCCESS adminId={}, guard={}",
            userId,
            guardSuccess,
        )

        return if (guardSuccess) {
            messageProvider.get("admin.cache_refresh_success")
        } else {
            messageProvider.get("admin.cache_refresh_failed")
        }
    }

    suspend fun restartAllBots(
        chatId: String,
        userId: String,
    ): String {
        log.info("HANDLE_ADMIN_RESTART_ALL chatId={}, userId={}", chatId, userId)
        requireAdminOrThrow(
            adminUserIds = appProperties.admin.userIds,
            userId = userId,
            chatId = chatId,
            logger = log,
            warnMessage = "ADMIN_PERMISSION_DENIED userId={}, chatId={}",
            warnArgs = arrayOf(userId, chatId),
            errorMessage = messageProvider.get("error.no_permission"),
        )

        // 응답 전송 후 재시작 (3초 딜레이)
        adminScope.launch {
            delay(RESTART_DELAY_MS)
            ProcessBuilder()
                .command("bash", "-c", "/home/kapu/gemini/scripts/restart_all_bots.sh")
                .start()
        }

        log.info("ADMIN_RESTART_ALL_INITIATED adminId={}", userId)
        return messageProvider.get("admin.restart_all_initiated")
    }
}
