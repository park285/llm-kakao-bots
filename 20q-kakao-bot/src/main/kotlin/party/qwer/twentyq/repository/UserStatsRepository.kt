package party.qwer.twentyq.repository

import org.springframework.data.repository.kotlin.CoroutineCrudRepository
import org.springframework.stereotype.Repository
import party.qwer.twentyq.model.UserStatsEntity

/**
 * 사용자 스탯 Repository (R2DBC Coroutine)
 * 방별 독립 스탯 (복합 ID: "chatId:userId")
 */
@Repository
interface UserStatsRepository : CoroutineCrudRepository<UserStatsEntity, String>
