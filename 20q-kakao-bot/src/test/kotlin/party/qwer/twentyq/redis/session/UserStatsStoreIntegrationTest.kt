package party.qwer.twentyq.redis.session

import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.test.annotation.DirtiesContext
import party.qwer.twentyq.model.BestScore
import party.qwer.twentyq.model.CategoryStat
import party.qwer.twentyq.model.UserStats
import java.time.Duration
import java.time.Instant

@SpringBootTest
@org.springframework.test.context.ActiveProfiles("integration")
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
class UserStatsStoreIntegrationTest {
    @Autowired
    private lateinit var store: UserStatsStore

    private val testChatId = "test-chat-${System.currentTimeMillis()}"
    private val testUserId = "test-user-${System.currentTimeMillis()}"

    @BeforeEach
    fun cleanup() =
        runTest {
            store.invalidate(testChatId, testUserId)
        }

    @Test
    fun `should return null when stats do not exist`() =
        runTest {
            val result = store.get("nonexistent-chat", "nonexistent-user")
            assertNull(result)
        }

    @Test
    fun `should store and retrieve user stats`() =
        runTest {
            val stats =
                UserStats(
                    userId = testUserId,
                    totalGamesStarted = 12,
                    totalGamesCompleted = 10,
                    totalSurrenders = 2,
                    totalQuestionsAsked = 150,
                    totalHintsUsed = 5,
                    bestScore =
                        BestScore(
                            questionCount = 8,
                            target = "사과",
                            category = "과일",
                            achievedAt = Instant.now(),
                        ),
                    categoryStats =
                        mapOf(
                            "과일" to CategoryStat(gamesCompleted = 5, surrenders = 1),
                            "동물" to CategoryStat(gamesCompleted = 5, surrenders = 1),
                        ),
                )

            store.set(testChatId, testUserId, stats)
            val retrieved = store.get(testChatId, testUserId)

            assertNotNull(retrieved)
            assertEquals(stats.userId, retrieved?.userId)
            assertEquals(stats.totalGamesCompleted, retrieved?.totalGamesCompleted)
            assertEquals(stats.bestScore?.questionCount, retrieved?.bestScore?.questionCount)
            assertEquals(2, retrieved?.categoryStats?.size)
        }

    @Test
    fun `should overwrite existing stats on set`() =
        runTest {
            val stats1 =
                UserStats(
                    userId = testUserId,
                    totalGamesCompleted = 5,
                )
            val stats2 =
                UserStats(
                    userId = testUserId,
                    totalGamesCompleted = 10,
                )

            store.set(testChatId, testUserId, stats1)
            store.set(testChatId, testUserId, stats2)

            val retrieved = store.get(testChatId, testUserId)
            assertEquals(10, retrieved?.totalGamesCompleted)
        }

    @Test
    fun `should invalidate stats and return null on get`() =
        runTest {
            val stats =
                UserStats(
                    userId = testUserId,
                    totalGamesCompleted = 5,
                )

            store.set(testChatId, testUserId, stats)
            store.invalidate(testChatId, testUserId)

            val retrieved = store.get(testChatId, testUserId)
            assertNull(retrieved)
        }

    @Test
    fun `should set TTL successfully for existing stats`() =
        runTest {
            val stats =
                UserStats(
                    userId = testUserId,
                    totalGamesCompleted = 5,
                )

            store.set(testChatId, testUserId, stats)
            val result = store.setTtl(testChatId, testUserId, Duration.ofMinutes(10))

            // TTL 설정은 성공해야 함
            // 실제 만료는 시간이 걸리므로 여기서는 설정만 검증
            assertNotNull(result)
        }

    @Test
    fun `should handle multiple users in same chat independently`() =
        runTest {
            val user1Id = "user1-${System.currentTimeMillis()}"
            val user2Id = "user2-${System.currentTimeMillis()}"

            val stats1 =
                UserStats(
                    userId = user1Id,
                    totalGamesCompleted = 5,
                )
            val stats2 =
                UserStats(
                    userId = user2Id,
                    totalGamesCompleted = 10,
                )

            store.set(testChatId, user1Id, stats1)
            store.set(testChatId, user2Id, stats2)

            val retrieved1 = store.get(testChatId, user1Id)
            val retrieved2 = store.get(testChatId, user2Id)

            assertEquals(5, retrieved1?.totalGamesCompleted)
            assertEquals(10, retrieved2?.totalGamesCompleted)

            // cleanup
            store.invalidate(testChatId, user1Id)
            store.invalidate(testChatId, user2Id)
        }
}
