package party.qwer.twentyq.service

import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.toList
import kotlinx.coroutines.launch
import org.slf4j.LoggerFactory
import org.springframework.boot.context.event.ApplicationReadyEvent
import org.springframework.context.event.EventListener
import org.springframework.stereotype.Component
import party.qwer.twentyq.redis.session.UserStatsStore
import party.qwer.twentyq.repository.UserStatsRepository
import party.qwer.twentyq.util.common.extensions.days
import java.time.Instant

/**
 * 사용자 통계 워밍업 서비스
 */
@Component
class UserStatsWarmupService(
    private val userStatsRepository: UserStatsRepository,
    private val statsStore: UserStatsStore,
) {
    companion object {
        private val log = LoggerFactory.getLogger(UserStatsWarmupService::class.java)
        private const val WARMUP_DAYS = 7
    }

    private val warmupScope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    /**
     * 애플리케이션 시작 시 워밍업 실행
     */
    @EventListener(ApplicationReadyEvent::class)
    fun warmupOnStartup() {
        warmupScope.launch {
            runCatching {
                warmupRecentActiveUsers()
            }.onFailure { e ->
                log.error("WARMUP_FAILED error={}", e.message, e)
            }
        }
    }

    /**
     * 최근 활성 사용자 통계 로드
     */
    private suspend fun warmupRecentActiveUsers() {
        val startTime = System.currentTimeMillis()
        log.info("WARMUP_START days={}", WARMUP_DAYS)

        val cutoffDate = Instant.now().minus(WARMUP_DAYS.days)

        // 최근 업데이트된 통계만 조회
        val recentStats =
            userStatsRepository
                .findAll()
                .toList()
                .filter { it.updatedAt.isAfter(cutoffDate) }

        log.info("WARMUP_LOADED count={}", recentStats.size)

        // Valkey에 개별 저장
        var successCount = 0
        recentStats.forEach { entity ->
            val categoryStats =
                runCatching { UserStatsCalculator.parseCategoryStatsJson(entity.categoryStatsJson) }
                    .onFailure { e ->
                        log.warn(
                            "WARMUP_ITEM_CATEGORY_PARSE_FAILED chatId={}, userId={}, error={}",
                            entity.chatId,
                            entity.userId,
                            e.message,
                        )
                    }.getOrElse { emptyMap() }

            runCatching {
                val stats = entity.toUserStats(categoryStats)
                statsStore.set(entity.chatId, entity.userId, stats)
                successCount++
            }.onFailure { e ->
                log.warn(
                    "WARMUP_ITEM_STORE_FAILED chatId={}, userId={}, error={}",
                    entity.chatId,
                    entity.userId,
                    e.message,
                )
            }
        }

        val elapsed = System.currentTimeMillis() - startTime
        log.info("WARMUP_SUCCESS loaded={}, cached={}, elapsed={}ms", recentStats.size, successCount, elapsed)
    }
}
