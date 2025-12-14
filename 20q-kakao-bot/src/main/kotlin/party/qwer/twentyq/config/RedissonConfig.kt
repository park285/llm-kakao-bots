package party.qwer.twentyq.config

import org.redisson.Redisson
import org.redisson.api.RedissonClient
import org.redisson.api.RedissonReactiveClient
import org.redisson.client.codec.StringCodec
import org.redisson.config.Config
import org.slf4j.LoggerFactory
import org.springframework.boot.data.redis.autoconfigure.DataRedisProperties
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.context.annotation.Primary
import org.springframework.context.annotation.Profile

/**
 * Redisson 클라이언트 연결 설정 상수
 */
private object RedissonDefaults {
    const val CONNECTION_POOL_SIZE = 64
    const val CONNECTION_MIN_IDLE_SIZE = 10
    const val DEFAULT_CONNECT_TIMEOUT_MS = 2000
    const val LOCK_WATCHDOG_TIMEOUT_MS = 120_000L
}

/**
 * Redisson 클라이언트 설정
 */
@Configuration
@Profile("!test")
class RedissonConfig(
    private val redisProperties: DataRedisProperties,
) {
    @Bean
    fun redissonClient(): RedissonClient {
        val log = LoggerFactory.getLogger(RedissonConfig::class.java)
        val address = "redis://${redisProperties.host}:${redisProperties.port}"

        log.info(
            "REDISSON_CONFIG host={}, port={}, password='{}'",
            redisProperties.host,
            redisProperties.port,
            redisProperties.password ?: "null",
        )

        return Config()
            .apply {
                setLockWatchdogTimeout(RedissonDefaults.LOCK_WATCHDOG_TIMEOUT_MS)
                setCodec(StringCodec.INSTANCE)
                useSingleServer().apply {
                    setAddress(address)
                    setConnectionPoolSize(RedissonDefaults.CONNECTION_POOL_SIZE)
                    setConnectionMinimumIdleSize(RedissonDefaults.CONNECTION_MIN_IDLE_SIZE)
                    setConnectTimeout(
                        redisProperties.timeout?.toMillis()?.toInt() ?: RedissonDefaults.DEFAULT_CONNECT_TIMEOUT_MS,
                    )
                    redisProperties.password
                        ?.takeIf { it.isNotBlank() }
                        ?.let(::setPassword)
                }
            }.let(Redisson::create)
    }

    @Bean
    @Primary
    fun redissonReactiveClient(redissonClient: RedissonClient): RedissonReactiveClient = redissonClient.reactive()
}
