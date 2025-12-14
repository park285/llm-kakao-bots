package party.qwer.twentyq.redis.tracking

import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.launch
import kotlinx.coroutines.reactor.awaitSingleOrNull
import org.redisson.api.RedissonReactiveClient
import org.slf4j.LoggerFactory
import org.springframework.boot.autoconfigure.condition.ConditionalOnProperty
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.redis.RedisKeys
import party.qwer.twentyq.redis.expireAsync
import party.qwer.twentyq.util.common.extensions.minutes
import java.time.Duration

@Component
@ConditionalOnProperty(
    prefix = "app.cache",
    name = ["enabled"],
    havingValue = "true",
    matchIfMissing = true,
)
class WrongGuessSetStore(
    private val redisson: RedissonReactiveClient,
    private val props: AppProperties,
) {
    companion object {
        private val log = LoggerFactory.getLogger(WrongGuessSetStore::class.java)
    }

    /**
     * 오답 추가 (Dual Storage: 방 전체 + 개인별 동시 저장)
     */
    suspend fun addAsync(
        roomId: String,
        guess: String,
        userId: String,
    ) {
        coroutineScope {
            // 병렬 저장: 방 전체 키 + 개인 키
            launch {
                val roomSet = redisson.getSet<String>(sessionKey(roomId))
                roomSet.add(guess).awaitSingleOrNull()
                roomSet.expireAsync(props.cache.sessionTtlMinutes.minutes)
            }
            launch {
                val userSet = redisson.getSet<String>(userKey(roomId, userId))
                userSet.add(guess).awaitSingleOrNull()
                userSet.expireAsync(props.cache.sessionTtlMinutes.minutes)
            }
        }
        log.debugL { "VALKEY WRONG_ADD room=$roomId userId=$userId guess=$guess (dual)" }
    }

    /**
     * 오답 전체 삭제 (방 전체 키 + 모든 개인별 키)
     */
    suspend fun deleteAsync(roomId: String) {
        coroutineScope {
            // 병렬 삭제: 방 전체 키 + 패턴 매칭으로 개인별 키 일괄 삭제
            launch {
                redisson.getSet<String>(sessionKey(roomId)).delete().awaitSingleOrNull()
            }
            launch {
                val userPattern = "${RedisKeys.WRONG_GUESSES}:$roomId:*"
                val deletedCount = redisson.keys.deleteByPattern(userPattern).awaitSingleOrNull() ?: 0L
                if (deletedCount > 0) {
                    log.debugL { "VALKEY WRONG_DELETE_USERS room=$roomId deletedKeys=$deletedCount" }
                }
            }
        }
        log.debugL { "VALKEY WRONG_DELETE room=$roomId (session+user keys deleted)" }
    }

    /**
     * TTL 갱신 (Dual Storage)
     */
    suspend fun setTtlAsync(
        roomId: String,
        ttl: Duration,
    ) {
        coroutineScope {
            // 방 전체 키 TTL 설정
            launch {
                redisson.getSet<String>(sessionKey(roomId)).expireAsync(ttl)
            }
            // 개별 사용자 키는 addAsync에서 자동 설정되므로 여기서는 방 전체만 처리
        }
        log.debugL { "VALKEY WRONG_TTL room=$roomId ttl=${ttl.toMillis()}ms (session key)" }
    }

    /**
     * 방 전체 오답 조회 (Status 표시용)
     */
    suspend fun getSessionWrongGuessesAsync(roomId: String): List<String> {
        val set = redisson.getSet<String>(sessionKey(roomId))
        return set.readAll().awaitSingleOrNull()?.toList() ?: emptyList()
    }

    /**
     * 개인별 오답 개수 (DB 통계용)
     */
    suspend fun getUserWrongGuessCountAsync(
        roomId: String,
        userId: String,
    ): Int {
        val set = redisson.getSet<String>(userKey(roomId, userId))
        return set.size().awaitSingleOrNull() ?: 0
    }

    /**
     * 개인별 오답 목록 (DB 통계용)
     */
    suspend fun getUserWrongGuessesAsync(
        roomId: String,
        userId: String,
    ): List<String> {
        val set = redisson.getSet<String>(userKey(roomId, userId))
        return set.readAll().awaitSingleOrNull()?.toList() ?: emptyList()
    }

    /**
     * 오답 중복 체크 (O(1) Set 연산)
     */
    suspend fun containsAsync(
        roomId: String,
        guess: String,
    ): Boolean {
        val set = redisson.getSet<String>(sessionKey(roomId))
        return set.contains(guess).awaitSingleOrNull() ?: false
    }

    // 방 전체 키 (세션 공유용)
    private fun sessionKey(roomId: String) = "${RedisKeys.WRONG_GUESSES}:$roomId"

    // 개인별 키 (DB 통계용)
    private fun userKey(
        roomId: String,
        userId: String,
    ) = "${RedisKeys.WRONG_GUESSES}:$roomId:$userId"
}
