package party.qwer.twentyq.config

import org.redisson.Redisson
import org.redisson.api.RedissonClient
import org.redisson.api.RedissonReactiveClient
import org.redisson.client.codec.StringCodec
import org.redisson.config.Config
import org.slf4j.LoggerFactory
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration

/**
 * Valkey MQ 스트림 설정
 */
@Configuration
class ValkeyMQConfig(
    private val appProperties: AppProperties,
) {
    @Bean
    fun redissonMQClient(): RedissonClient {
        val mq = appProperties.mq
        val log = LoggerFactory.getLogger(ValkeyMQConfig::class.java)
        val address = "redis://${mq.host}:${mq.port}"

        log.info(
            "VALKEY_MQ_CONFIG host={}, port={}, password='{}', consumerGroup={}, consumerName={}, streamKey={}",
            mq.host,
            mq.port,
            mq.password ?: "null",
            mq.consumerGroup,
            mq.consumerName,
            mq.streamKey,
        )

        return Config()
            .apply {
                setCodec(StringCodec.INSTANCE)
                useSingleServer().apply {
                    setAddress(address)
                    setConnectionPoolSize(mq.connectionPoolSize)
                    setConnectionMinimumIdleSize(mq.connectionMinIdleSize)
                    setConnectTimeout(mq.timeout)
                    mq.password
                        ?.takeIf { it.isNotBlank() }
                        ?.let(::setPassword)
                }
            }.let(Redisson::create)
    }

    @Bean
    fun redissonMQReactiveClient(redissonMQClient: RedissonClient): RedissonReactiveClient = redissonMQClient.reactive()
}
