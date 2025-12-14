package party.qwer.twentyq.config

import org.springframework.cache.annotation.EnableCaching
import org.springframework.context.annotation.Bean
import org.springframework.context.annotation.Configuration
import org.springframework.data.redis.cache.RedisCacheConfiguration
import org.springframework.data.redis.cache.RedisCacheManager
import org.springframework.data.redis.connection.RedisConnectionFactory
import org.springframework.data.redis.serializer.GenericJacksonJsonRedisSerializer
import org.springframework.data.redis.serializer.RedisSerializationContext
import org.springframework.data.redis.serializer.StringRedisSerializer
import party.qwer.twentyq.util.common.extensions.days
import tools.jackson.databind.json.JsonMapper

/**
 * 캐시 매니저 설정
 */
@Configuration
@EnableCaching
class CacheConfiguration {
    /**
     * 표준 JSON ObjectMapper (Kotlin 지원)
     * - data class, sealed class 직렬화/역직렬화
     */
    @Bean
    fun kotlinJsonMapper(): tools.jackson.databind.ObjectMapper =
        JsonMapper
            .builder()
            .addModule(
                tools.jackson.module.kotlin
                    .kotlinModule(),
            ).build()

    /**
     * 표준 YAML ObjectMapper (Kotlin 지원)
     * - 게임 메시지, 프롬프트 등 YAML 파싱
     */
    @Bean
    fun yamlObjectMapper(): tools.jackson.databind.ObjectMapper =
        tools.jackson.dataformat.yaml.YAMLMapper
            .builder()
            .addModule(
                tools.jackson.module.kotlin
                    .kotlinModule(),
            ).build()

    @Bean
    fun redisSerializerObjectMapper(): tools.jackson.databind.ObjectMapper = JsonMapper.builder().build()

    @Bean
    fun cacheManager(
        redisConnectionFactory: RedisConnectionFactory,
        redisSerializerObjectMapper: tools.jackson.databind.ObjectMapper,
    ): org.springframework.cache.CacheManager {
        val redisConfig =
            RedisCacheConfiguration.defaultCacheConfig().apply {
                entryTtl(30.days)
                serializeKeysWith(
                    RedisSerializationContext.SerializationPair
                        .fromSerializer(StringRedisSerializer()),
                )
                serializeValuesWith(
                    RedisSerializationContext.SerializationPair
                        .fromSerializer(GenericJacksonJsonRedisSerializer(redisSerializerObjectMapper)),
                )
                disableCachingNullValues()
            }

        return RedisCacheManager
            .builder(redisConnectionFactory)
            .cacheDefaults(redisConfig)
            .build()
    }
}
