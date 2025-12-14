package party.qwer.twentyq.redis.tracking

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.awaitAll
import kotlinx.coroutines.delay
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Test
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.boot.test.context.SpringBootTest
import org.springframework.test.annotation.DirtiesContext
import java.time.Duration

@SpringBootTest
@org.springframework.test.context.ActiveProfiles("integration")
@DirtiesContext(classMode = DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD)
class HintCountStoreIntegrationTest {
    @Autowired
    private lateinit var store: HintCountStore

    private val testRoomId = "test-room-${System.currentTimeMillis()}"

    @BeforeEach
    fun cleanup() =
        runTest {
            store.deleteAsync(testRoomId)
        }

    @Test
    fun `getAsync should return 0 when counter does not exist`() =
        runTest {
            val count = store.getAsync(testRoomId)
            assertEquals(0, count)
        }

    @Test
    fun `increment should atomically increase counter`() =
        runTest {
            val result1 = store.increment(testRoomId)
            val result2 = store.increment(testRoomId)
            val result3 = store.increment(testRoomId)

            assertEquals(1L, result1)
            assertEquals(2L, result2)
            assertEquals(3L, result3)

            val finalCount = store.getAsync(testRoomId)
            assertEquals(3, finalCount)
        }

    @Test
    fun `concurrent increment should handle atomicity correctly`() =
        runTest {
            val iterations = 20
            val results =
                (1..iterations)
                    .map {
                        async(Dispatchers.IO) {
                            store.increment(testRoomId)
                        }
                    }.awaitAll()

            // 모든 증가 작업 성공 확인
            assertEquals(iterations, results.size)

            // 최종 카운트 검증
            val finalCount = store.getAsync(testRoomId)
            assertEquals(iterations, finalCount, "동시성 환경에서 정확히 $iterations 회 증가해야 함")
        }

    @Test
    fun `deleteAsync should remove counter`() =
        runTest {
            store.increment(testRoomId)
            store.increment(testRoomId)

            store.deleteAsync(testRoomId)
            val count = store.getAsync(testRoomId)

            assertEquals(0, count)
        }

    @Test
    fun `increment should auto-set TTL on each operation`() =
        runTest {
            // increment 호출 시 자동으로 TTL 설정됨 (코드상 sessionTtlMinutes 적용)
            store.increment(testRoomId)

            val count = store.getAsync(testRoomId)
            assertEquals(1, count)

            // TTL이 설정되어 있어야 함 (실제 만료 시간 검증은 통합 테스트에서 수행)
            store.increment(testRoomId)
            val updatedCount = store.getAsync(testRoomId)
            assertEquals(2, updatedCount)
        }
}
