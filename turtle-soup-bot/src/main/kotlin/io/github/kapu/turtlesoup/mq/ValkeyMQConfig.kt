package io.github.kapu.turtlesoup.mq

import io.github.oshai.kotlinlogging.KotlinLogging
import org.redisson.Redisson
import org.redisson.api.RedissonReactiveClient
import org.redisson.codec.JsonJacksonCodec
import org.redisson.config.Config
import io.github.kapu.turtlesoup.config.ValkeyMQConfig as ValkeyMQSettings

private val logger = KotlinLogging.logger {}

/**
 * Redisson Reactive 클라이언트 생성
 */
fun createRedissonReactiveClient(settings: ValkeyMQSettings): RedissonReactiveClient {
    val config =
        Config().apply {
            useSingleServer().apply {
                address = "redis://${settings.host}:${settings.port}"
                settings.password?.let { password = it }
                connectionPoolSize = settings.connectionPoolSize
                connectionMinimumIdleSize = settings.connectionMinIdleSize
                timeout = settings.timeout
            }
            codec = JsonJacksonCodec()
        }

    logger.info {
        "redisson_reactive_client_created " +
            "host=${settings.host} " +
            "port=${settings.port} " +
            "pool_size=${settings.connectionPoolSize}"
    }

    return Redisson.create(config).reactive()
}
