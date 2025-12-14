package party.qwer.twentyq.api

import org.slf4j.LoggerFactory
import org.springframework.http.ResponseEntity
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController
import party.qwer.twentyq.model.UserStats
import party.qwer.twentyq.service.UserStatsService

/**
 * 사용자 스탯 API 컨트롤러
 */
@RestController
@RequestMapping("/api/twentyq/stats")
class UserStatsController(
    private val userStatsService: UserStatsService,
) {
    companion object {
        private val log = LoggerFactory.getLogger(UserStatsController::class.java)
    }

    /**
     * 방별 사용자 스탯 조회
     */
    @GetMapping("/rooms/{chatId}/users/{userId}")
    suspend fun getUserStats(
        @PathVariable chatId: String,
        @PathVariable userId: String,
    ): ResponseEntity<UserStats> {
        log.info("USER_STATS_REQUEST chatId={}, userId={}", chatId, userId)

        val stats = userStatsService.getUserStats(chatId, userId)

        return if (stats != null) {
            log.info(
                "USER_STATS_SUCCESS userId={}, totalStarted={}, totalCompleted={}",
                userId,
                stats.totalGamesStarted,
                stats.totalGamesCompleted,
            )
            ResponseEntity.ok(stats)
        } else {
            log.info("USER_STATS_NOT_FOUND userId={}", userId)
            ResponseEntity.notFound().build()
        }
    }
}
