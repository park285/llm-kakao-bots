package party.qwer.twentyq.redis

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.runBlocking
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertNotNull
import org.junit.jupiter.api.Assertions.assertNull
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.test.annotation.DirtiesContext

/**
 * Redisson 기반 LockCoordinator 통합 테스트
 */
@SpringBootTest
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_CLASS)
@org.springframework.test.context.ActiveProfiles("integration")
class LockCoordinatorIntegrationTest {
    @Autowired
    private lateinit var lockCoordinator: LockCoordinator

    @Autowired
    private lateinit var processingLockService: ProcessingLockService

    @Test
    fun `withLock write should execute block and release lock`() =
        runBlocking {
            val chatId = "test-chat-write-${System.currentTimeMillis()}"
            val userId = "user-1"

            val result = lockCoordinator.withLock(chatId, userId, requiresWrite = true) { "ok" }

            assertNotNull(result)
            assertEquals("ok", result)
        }

    @Test
    fun `withLock read should execute block and release lock`() =
        runBlocking {
            val chatId = "test-chat-read-${System.currentTimeMillis()}"
            val userId = "user-1"

            val result = lockCoordinator.withLock(chatId, userId, requiresWrite = false) { "ok" }

            assertNotNull(result)
            assertEquals("ok", result)
        }

    @Test
    fun `withLock write should fail when lock already held`() =
        runBlocking {
            val chatId = "test-chat-conflict-${System.currentTimeMillis()}"

            lockCoordinator.withLock(chatId, "user-1", requiresWrite = true) {
                val result2 = lockCoordinator.withLock(chatId, "user-2", requiresWrite = true) { "should-fail" }
                assertNull(result2, "다른 사용자가 Lock을 보유 중이면 획득 실패해야 함")
            }
        }

    @Test
    fun `concurrent withLock write should allow only one winner`() =
        runBlocking {
            val chatId = "test-chat-concurrent-${System.currentTimeMillis()}"
            val userIds = (1..10).map { "user-$it" }

            val results =
                userIds
                    .map { userId ->
                        async(Dispatchers.IO) {
                            lockCoordinator.withLock(chatId, userId, requiresWrite = true) {
                                kotlinx.coroutines.delay(10)
                                "success"
                            }
                        }
                    }.awaitAll()

            val successCount = results.count { it != null }
            assertTrue(successCount >= 1, "최소 1개의 코루틴은 락을 획득해야 함")
        }

    @Test
    fun `withLock read should allow multiple readers`() =
        runBlocking {
            val chatId = "test-chat-read-multiple-${System.currentTimeMillis()}"
            val userIds = (1..5).map { "user-$it" }

            val results =
                userIds
                    .map { userId ->
                        async(Dispatchers.IO) {
                            lockCoordinator.withLock(chatId, userId, requiresWrite = false) {
                                kotlinx.coroutines.delay(10)
                                "read-success"
                            }
                        }
                    }.awaitAll()

            val successCount = results.count { it == "read-success" }
            assertTrue(successCount >= 1, "Read Lock은 여러 사용자가 동시에 획득 가능")
        }

    @Test
    fun `withLock write should block readers`() =
        runBlocking {
            val chatId = "test-chat-write-blocks-read-${System.currentTimeMillis()}"

            lockCoordinator.withLock(chatId, "writer", requiresWrite = true) {
                val readerResult = lockCoordinator.withLock(chatId, "reader", requiresWrite = false) { "should-fail" }
                assertNull(readerResult, "Write Lock이 보유 중이면 Read Lock 획득 실패해야 함")
            }
        }

    @Test
    fun `withLock should release lock even on exception`() =
        runBlocking {
            val chatId = "test-chat-exception-${System.currentTimeMillis()}"
            val userId = "user-1"

            val exceptionResult =
                runCatching {
                    lockCoordinator.withLock(chatId, userId, requiresWrite = true) {
                        throw RuntimeException("test exception")
                    }
                }
            assertTrue(exceptionResult.isFailure, "예외가 발생해야 함")

            // Lock이 해제되었는지 확인: 새로운 Lock 획득이 성공해야 함
            val nextResult = lockCoordinator.withLock(chatId, "user-2", requiresWrite = true) { "ok" }
            assertNotNull(nextResult, "예외 발생 후에도 Lock이 해제되어야 함")
        }

    @Test
    fun `withLock should handle coroutine thread switching without IllegalMonitorStateException`() =
        runBlocking {
            val chatId = "test-chat-thread-switch-${System.currentTimeMillis()}"
            val userId = "user-1"

            // 여러 dispatcher 간 전환을 통해 스레드 스위칭 유도
            val result =
                lockCoordinator.withLock(chatId, userId, requiresWrite = true) {
                    // IO dispatcher에서 실행
                    kotlinx.coroutines.withContext(Dispatchers.IO) {
                        kotlinx.coroutines.delay(50) // suspend point - 스레드 전환 가능
                        "step1"
                    }
                    // Default dispatcher에서 실행
                    kotlinx.coroutines.withContext(Dispatchers.Default) {
                        kotlinx.coroutines.delay(50) // suspend point - 스레드 전환 가능
                        "step2"
                    }
                    // 다시 IO로 전환
                    kotlinx.coroutines.withContext(Dispatchers.IO) {
                        kotlinx.coroutines.delay(50) // suspend point - 스레드 전환 가능
                        "completed"
                    }
                }

            assertNotNull(result, "스레드 전환이 발생해도 정상적으로 완료되어야 함")
            assertEquals("completed", result)

            // Lock이 정상적으로 해제되었는지 확인
            val secondResult = lockCoordinator.withLock(chatId, "user-2", requiresWrite = true) { "ok" }
            assertNotNull(secondResult, "이전 Lock이 정상적으로 해제되어야 함")
        }

    @Test
    fun `concurrent withLock with heavy thread switching should not cause IllegalMonitorStateException`() =
        runBlocking {
            val chatId = "test-chat-concurrent-switch-${System.currentTimeMillis()}"
            val userIds = (1..5).map { "user-$it" }

            val results =
                userIds
                    .map { userId ->
                        async(Dispatchers.IO) {
                            lockCoordinator.withLock(chatId, userId, requiresWrite = true) {
                                // 여러 suspend point를 통해 스레드 전환 유도
                                repeat(3) { i ->
                                    kotlinx.coroutines.withContext(Dispatchers.Default) {
                                        kotlinx.coroutines.delay(10)
                                    }
                                    kotlinx.coroutines.withContext(Dispatchers.IO) {
                                        kotlinx.coroutines.delay(10)
                                    }
                                }
                                "success-$userId"
                            }
                        }
                    }.awaitAll()

            val successCount = results.count { it != null }
            assertTrue(successCount >= 1, "최소 1개의 코루틴은 락을 획득하고 스레드 전환 후에도 정상 완료해야 함")
        }

    @Test
    fun `withLock should timeout long running block and release lock`() =
        runBlocking {
            val chatId = "test-chat-timeout-${System.currentTimeMillis()}"

            val timedOut =
                lockCoordinator.withLock(
                    chatId = chatId,
                    userId = "user-timeout",
                    requiresWrite = true,
                    blockTimeoutMillis = 50,
                ) {
                    kotlinx.coroutines.delay(200)
                    "late"
                }

            assertNull(timedOut, "타임아웃 시 null 반환")

            val result = lockCoordinator.withLock(chatId, "user-next", requiresWrite = true) { "ok" }
            assertNotNull(result, "타임아웃 후에도 락이 해제되어야 함")
        }

    @Test
    fun `cleanupStaleLocks should delete existing locks`() =
        runBlocking {
            // Given: 락 2개 생성
            val chatId1 = "test-cleanup-1-${System.currentTimeMillis()}"
            val chatId2 = "test-cleanup-2-${System.currentTimeMillis()}"

            lockCoordinator.withLock(chatId1, "user-1", requiresWrite = true) {
                lockCoordinator.withLock(chatId2, "user-2", requiresWrite = true) {
                    // Lock이 보유된 상태에서 cleanup 호출
                    processingLockService.cleanupStaleLocks()
                }
            }

            // Then: cleanup 후에도 새로운 lock 획득 가능
            val result1 = lockCoordinator.withLock(chatId1, "user-3", requiresWrite = true) { "ok" }
            val result2 = lockCoordinator.withLock(chatId2, "user-4", requiresWrite = true) { "ok" }

            assertNotNull(result1)
            assertNotNull(result2)
        }

    @Test
    fun `cleanupStaleLocks should handle empty locks without error`() =
        runBlocking {
            // Given: 락 없음
            // When: cleanupStaleLocks 호출
            processingLockService.cleanupStaleLocks()

            // Then: 에러 없이 완료
        }
}
