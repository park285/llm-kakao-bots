package party.qwer.twentyq.config

import io.mockk.every
import io.mockk.mockk
import org.redisson.api.RQueueReactive
import org.redisson.api.RedissonClient
import org.redisson.api.RedissonReactiveClient
import org.springframework.beans.factory.annotation.Qualifier
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.context.annotation.Primary
import reactor.core.publisher.Mono

/**
 * 테스트용 Redisson 설정 (통합 테스트 제외)
 */
@Configuration
@org.springframework.context.annotation.Profile("!integration")
class TestRedissonConfig {
    @Bean
    @Primary
    fun redissonClient(): RedissonClient = mockk(relaxed = true)

    @Bean
    @Primary
    @Qualifier("redissonReactiveClient")
    fun redissonReactiveClient(redissonClient: RedissonClient): RedissonReactiveClient {
        val mockClient = mockk<RedissonReactiveClient>(relaxed = true)
        val mockQueue = mockk<RQueueReactive<String>>(relaxed = true)

        // getQueue().readAll()이 빈 Mono를 반환하도록 설정
        every { mockQueue.readAll() } returns Mono.just(emptyList())
        every { mockClient.getQueue<String>(any(), any()) } returns mockQueue

        return mockClient
    }
}
