package party.qwer.twentyq.rest

import org.springframework.boot.context.properties.ConfigurationProperties

/** LLM REST 설정 프로퍼티 */
@ConfigurationProperties(prefix = "llm.rest")
data class LlmRestProperties(
    var baseUrl: String = "http://localhost:40527",
    var timeoutSeconds: Long = 120,
    var connectTimeoutSeconds: Long = 10,
    var http2Enabled: Boolean = true,
    // Health check settings
    var healthEnabled: Boolean = true,
    var healthPath: String = "/health",
    var healthIntervalMillis: Long = 60_000,
    var healthFailureThreshold: Int = 5,
    var healthRestartCommand: String = "./bot-restart.sh",
    var healthRestartContainers: List<String> = emptyList(),
    var healthDockerSocket: String = "/var/run/docker.sock",
    var healthRestartLockKey: String = "",
    var healthRestartLockTtlSeconds: Long = 120,
)
