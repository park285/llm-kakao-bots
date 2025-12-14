package party.qwer.twentyq.config

import org.springframework.boot.context.properties.ConfigurationProperties
import party.qwer.twentyq.config.properties.Access
import party.qwer.twentyq.config.properties.Admin
import party.qwer.twentyq.config.properties.Commands
import party.qwer.twentyq.config.properties.Pricing
import party.qwer.twentyq.config.properties.RedisDefaults
import party.qwer.twentyq.config.properties.RiddleConfig
import party.qwer.twentyq.config.properties.Security
import party.qwer.twentyq.config.properties.Topics
import party.qwer.twentyq.config.properties.ValkeyMQ

@ConfigurationProperties(prefix = "app")
data class AppProperties(
    val cache: RedisDefaults,
    val mq: ValkeyMQ = ValkeyMQ(),
    val topics: Topics = Topics(),
    val riddle: RiddleConfig = RiddleConfig(),
    val security: Security = Security(),
    val access: Access = Access(),
    val commands: Commands = Commands(),
    val admin: Admin = Admin(),
    val pricing: Pricing = Pricing(),
)
