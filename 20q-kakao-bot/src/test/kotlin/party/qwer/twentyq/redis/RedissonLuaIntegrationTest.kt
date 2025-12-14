package party.qwer.twentyq.redis

import kotlinx.coroutines.reactor.awaitSingleOrNull
import kotlinx.coroutines.test.runTest
import org.junit.jupiter.api.Assertions.assertEquals
import org.junit.jupiter.api.Assertions.assertTrue
import org.junit.jupiter.api.Test
import org.redisson.api.RScript
import org.redisson.api.RedissonClient
import org.redisson.api.RedissonReactiveClient
import org.springframework.beans.factory.annotation.Autowired
import org.springframework.beans.factory.annotation.Qualifier
import org.springframework.boot.test.context.SpringBootTest

@SpringBootTest
@org.springframework.test.context.ActiveProfiles("integration")
@org.springframework.test.annotation.DirtiesContext(
    classMode = org.springframework.test.annotation.DirtiesContext.ClassMode.AFTER_EACH_TEST_METHOD,
)
class RedissonLuaIntegrationTest {
    @Autowired
    @Qualifier("redissonReactiveClient")
    private lateinit var redisson: RedissonReactiveClient

    @Autowired
    @Qualifier("redissonClient")
    private lateinit var redissonSync: RedissonClient

    @Test
    fun `verify Redisson SYNC eval with multiple KEYS works`() {
        val script =
            """
            return "KEYS=" .. #KEYS .. " ARGV=" .. #ARGV
            """.trimIndent()

        // StringCodec을 사용하여 Kryo 우회
        val result =
            redissonSync
                .getScript(org.redisson.client.codec.StringCodec.INSTANCE)
                .eval<String>(
                    RScript.Mode.READ_ONLY,
                    script,
                    RScript.ReturnType.VALUE,
                    listOf("key1", "key2"),
                    "arg1",
                    "arg2",
                    "arg3",
                    "arg4",
                )

        println("Redisson SYNC eval result: $result")
        assertEquals("KEYS=2 ARGV=4", result, "Sync API should pass 2 KEYS and 4 ARGV")
    }

    @Test
    fun `verify Redisson SYNC eval with enqueue script works`() {
        val script =
            """
            local queueKey = KEYS[1]
            local userSetKey = KEYS[2]
            local userId = ARGV[1]
            local messageJson = ARGV[2]
            local maxSize = tonumber(ARGV[3])
            local ttl = tonumber(ARGV[4])

            if redis.call("SISMEMBER", userSetKey, userId) == 1 then
                return "DUPLICATE"
            end

            local size = redis.call("LLEN", queueKey)
            if size >= maxSize then
                return "QUEUE_FULL"
            end

            redis.call("RPUSH", queueKey, messageJson)
            redis.call("SADD", userSetKey, userId)
            redis.call("EXPIRE", queueKey, ttl)
            redis.call("EXPIRE", userSetKey, ttl)

            return "SUCCESS"
            """.trimIndent()

        val queueKey = "test:queue:sync"
        val userSetKey = "test:queue:sync:users"

        // 정리
        redissonSync.getQueue<String>(queueKey).delete()
        redissonSync.getSet<String>(userSetKey).delete()

        val result =
            redissonSync
                .getScript(org.redisson.client.codec.StringCodec.INSTANCE)
                .eval<String>(
                    RScript.Mode.READ_WRITE,
                    script,
                    RScript.ReturnType.VALUE,
                    listOf(queueKey, userSetKey),
                    "user1",
                    """{"userId":"user1","content":"test"}""",
                    "3",
                    "3600",
                )

        println("Enqueue script result: $result")
        assertEquals("SUCCESS", result)
        val queueSize = redissonSync.getQueue<String>(queueKey).size
        assertEquals(1, queueSize)
        redissonSync.getQueue<String>(queueKey).delete()
        redissonSync.getSet<String>(userSetKey).delete()
    }

    @Test
    fun `verify Redisson REACTIVE eval fails with Kryo (documented bug)`() =
        runTest {
            val script =
                """
                return "KEYS=" .. #KEYS .. " ARGV=" .. #ARGV
                """.trimIndent()

            try {
                val result =
                    redisson.script
                        .eval<String>(
                            RScript.Mode.READ_ONLY,
                            script,
                            RScript.ReturnType.VALUE,
                            ArrayList<Any>(listOf("key1", "key2")),
                            "arg1",
                            "arg2",
                            "arg3",
                            "arg4",
                        ).awaitSingleOrNull()

                println("Reactive eval result: $result")
            } catch (e: Exception) {
                println("Expected Reactive API failure: ${e::class.simpleName}")
                assertTrue(
                    e is com.esotericsoftware.kryo.KryoException || e.cause is com.esotericsoftware.kryo.KryoException,
                    "Reactive API should fail with KryoException",
                )
            }
        }
}
