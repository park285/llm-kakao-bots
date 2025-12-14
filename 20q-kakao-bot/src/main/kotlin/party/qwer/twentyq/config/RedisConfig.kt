package party.qwer.twentyq.config

import org.redisson.api.RedissonClient
import org.redisson.spring.data.connection.RedissonConnectionFactory
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.data.redis.connection.RedisConnectionFactory

/**
 * Redis 연결 설정
 */
@Configuration
class RedisConfig(
    private val redissonClient: RedissonClient,
) {
    @Bean
    fun redissonConnectionFactory(): RedisConnectionFactory = RedissonConnectionFactory(redissonClient)
}
