package party.qwer.twentyq

import org.springframework.boot.autoconfigure.SpringBootApplication
import org.springframework.boot.context.properties.EnableConfigurationProperties
import org.springframework.boot.runApplication
import org.springframework.scheduling.annotation.EnableScheduling
import party.qwer.twentyq.config.AppProperties

@SpringBootApplication(
    exclude = [org.redisson.spring.starter.RedissonAutoConfigurationV2::class],
)
@EnableConfigurationProperties(AppProperties::class)
@EnableScheduling
class TwentyQKakaoBotApplication

fun main() {
    runApplication<TwentyQKakaoBotApplication>()
}
