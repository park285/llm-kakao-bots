package party.qwer.twentyq.config

import org.redisson.api.RedissonClient
import org.redisson.client.RedisException
import org.slf4j.LoggerFactory
import org.springframework.boot.health.contributor.Health
import org.springframework.boot.health.contributor.HealthIndicator
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.errorL

/**
 * Redis 헬스 체크 지표
 */
@Component
class RedisHealthIndicator(
    private val redissonClient: RedissonClient,
) : HealthIndicator {
    private val log = LoggerFactory.getLogger(javaClass)

    override fun health(): Health =
        try {
            // ping 테스트
            redissonClient.keys.count()

            Health
                .up()
                .withDetail("status", "connected")
                .build()
        } catch (e: RedisException) {
            log.errorL { "Redis health check failed: ${e.message}" }
            Health
                .down()
                .withDetail("error", e.message ?: "unknown")
                .build()
        }
}
