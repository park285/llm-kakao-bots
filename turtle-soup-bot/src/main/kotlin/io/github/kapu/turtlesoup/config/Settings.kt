package io.github.kapu.turtlesoup.config

import com.typesafe.config.Config
import com.typesafe.config.ConfigFactory
import com.typesafe.config.ConfigValueType
import io.github.cdimascio.dotenv.dotenv

// 환경변수 우선 config 로딩 유틸리티 (dotenv -> System.getenv -> application.conf)
private object ConfigEnv {
    private val dotenv =
        dotenv {
            ignoreIfMissing = true
            systemProperties = false
        }

    fun env(key: String): String? = dotenv[key] ?: System.getenv(key)

    fun parseList(
        envVar: String?,
        fallback: List<String>,
    ): List<String> =
        envVar?.split(Regex("[,\\s]+"))
            ?.map { it.trim() }
            ?.filter { it.isNotEmpty() }
            ?: fallback

    fun Config.str(
        envKey: String,
        path: String,
    ): String = env(envKey) ?: getString(path)

    fun Config.int(
        envKey: String,
        path: String,
    ): Int = env(envKey)?.toIntOrNull() ?: getInt(path)

    fun Config.long(
        envKey: String,
        path: String,
    ): Long = env(envKey)?.toLongOrNull() ?: getLong(path)

    fun Config.double(
        envKey: String,
        path: String,
    ): Double = env(envKey)?.toDoubleOrNull() ?: getDouble(path)

    fun Config.bool(
        envKey: String,
        path: String,
    ): Boolean = env(envKey)?.toBooleanStrictOrNull() ?: getBoolean(path)

    fun Config.nullableStr(
        envKey: String,
        path: String,
    ): String? = (env(envKey) ?: getString(path)).takeIf { it.isNotBlank() }

    fun Config.strOrDefault(
        envKey: String,
        path: String,
        default: String,
    ): String = env(envKey) ?: runCatching { getString(path) }.getOrNull() ?: default
}

data class Settings(
    val server: ServerConfig,
    val llmRest: LlmRestConfig,
    val puzzle: PuzzleConfig,
    val redis: RedisConfig,
    val valkeyMq: ValkeyMQConfig,
    val access: AccessConfig,
) {
    companion object {
        fun load(): Settings {
            val config = ConfigFactory.load()
            return Settings(
                server = loadServerConfig(config),
                llmRest = LlmRestConfigLoader.load(config),
                puzzle = loadPuzzleConfig(config),
                redis = loadRedisConfig(config),
                valkeyMq = loadValkeyMQConfig(config),
                access = loadAccessConfig(config),
            )
        }

        private fun loadServerConfig(config: Config) =
            with(ConfigEnv) {
                ServerConfig(
                    host = config.str("SERVER_HOST", "server.host"),
                    port = config.int("SERVER_PORT", "server.port"),
                )
            }

        private fun loadPuzzleConfig(config: Config) =
            with(ConfigEnv) {
                PuzzleConfig(
                    rewriteEnabled = config.bool("PUZZLE_REWRITE_ENABLED", "puzzle.rewriteEnabled"),
                )
            }

        private fun loadRedisConfig(config: Config) =
            with(ConfigEnv) {
                RedisConfig(
                    host = config.str("REDIS_HOST", "redis.host"),
                    port = config.int("REDIS_PORT", "redis.port"),
                    password = config.nullableStr("REDIS_PASSWORD", "redis.password"),
                )
            }

        private fun loadValkeyMQConfig(config: Config) =
            with(ConfigEnv) {
                ValkeyMQConfig(
                    host = config.str("VALKEY_MQ_HOST", "valkeyMq.host"),
                    port = config.int("VALKEY_MQ_PORT", "valkeyMq.port"),
                    password = config.nullableStr("VALKEY_MQ_PASSWORD", "valkeyMq.password"),
                    timeout = config.int("VALKEY_MQ_TIMEOUT", "valkeyMq.timeout"),
                    connectionPoolSize = config.int("VALKEY_MQ_CONNECTION_POOL_SIZE", "valkeyMq.connectionPoolSize"),
                    connectionMinIdleSize =
                        config.int(
                            "VALKEY_MQ_CONNECTION_MIN_IDLE_SIZE",
                            "valkeyMq.connectionMinIdleSize",
                        ),
                    consumerGroup = config.str("VALKEY_MQ_CONSUMER_GROUP", "valkeyMq.consumerGroup"),
                    consumerName = config.str("VALKEY_MQ_CONSUMER_NAME", "valkeyMq.consumerName"),
                    streamKey = config.str("VALKEY_MQ_STREAM_KEY", "valkeyMq.streamKey"),
                    replyStreamKey = config.str("VALKEY_MQ_REPLY_STREAM_KEY", "valkeyMq.replyStreamKey"),
                )
            }

        private fun loadAccessConfig(config: Config) =
            with(ConfigEnv) {
                AccessConfig(
                    enabled = config.bool("ACCESS_ENABLED", "access.enabled"),
                    allowedChatIds =
                        parseList(
                            env("ACCESS_ALLOWED_CHAT_IDS"),
                            config.getStringList("access.allowedChatIds"),
                        ),
                    blockedChatIds =
                        parseList(
                            env("ACCESS_BLOCKED_CHAT_IDS"),
                            config.getStringList("access.blockedChatIds"),
                        ),
                    blockedUserIds =
                        parseList(
                            env("ACCESS_BLOCKED_USER_IDS"),
                            config.getStringList("access.blockedUserIds"),
                        ),
                    passthrough = config.bool("ACCESS_PASSTHROUGH", "access.passthrough"),
                )
            }
    }
}

data class ServerConfig(
    val host: String,
    val port: Int,
)

data class LlmRestConfig(
    val baseUrl: String,
    val timeoutSeconds: Long,
    val connectTimeoutSeconds: Long,
    val http2Enabled: Boolean = true,
    val healthEnabled: Boolean = true,
    val healthPath: String = "/health",
    val healthIntervalMillis: Long = 60_000,
    val healthFailureThreshold: Int = 5,
    val healthRestartCommand: String = "./bot-restart.sh",
    val healthRestartContainers: List<String> = emptyList(),
    val healthDockerSocket: String = "/var/run/docker.sock",
    val healthRestartLockKey: String = "",
    val healthRestartLockTtlSeconds: Long = RedisConstants.LOCK_TTL_SECONDS,
)

/** 퍼즐 프리셋 설정 */
data class PuzzleConfig(
    val rewriteEnabled: Boolean,
)

data class RedisConfig(
    val host: String,
    val port: Int,
    val password: String?,
)

data class ValkeyMQConfig(
    val host: String,
    val port: Int,
    val password: String?,
    val timeout: Int,
    val connectionPoolSize: Int,
    val connectionMinIdleSize: Int,
    val consumerGroup: String,
    val consumerName: String,
    val streamKey: String,
    val replyStreamKey: String,
)

/**
 * 접근 제어 설정
 */
data class AccessConfig(
    val enabled: Boolean,
    val allowedChatIds: List<String>,
    val blockedChatIds: List<String>,
    val blockedUserIds: List<String>,
    val passthrough: Boolean,
)

private object LlmRestConfigLoader {
    fun load(config: Config): LlmRestConfig =
        with(ConfigEnv) {
            LlmRestConfig(
                baseUrl = config.str("LLM_REST_BASE_URL", "llmRest.baseUrl"),
                timeoutSeconds =
                    config.long("LLM_REST_TIMEOUT_SECONDS", "llmRest.timeoutSeconds"),
                connectTimeoutSeconds =
                    config.long("LLM_REST_CONNECT_TIMEOUT_SECONDS", "llmRest.connectTimeoutSeconds"),
                http2Enabled = config.resolveHttp2Enabled(),
                healthEnabled = config.resolveHealthEnabled(),
                healthPath = config.resolveHealthPath(),
                healthIntervalMillis = config.resolveHealthInterval(),
                healthFailureThreshold = config.resolveHealthThreshold(),
                healthRestartCommand = config.resolveHealthRestartCommand(),
                healthRestartContainers = resolveRestartContainers(config),
                healthDockerSocket = config.resolveHealthDockerSocket(),
                healthRestartLockKey = config.resolveHealthRestartLockKey(),
                healthRestartLockTtlSeconds = config.resolveHealthRestartLockTtlSeconds(),
            )
        }

    private fun Config.resolveHttp2Enabled(): Boolean =
        with(ConfigEnv) { bool("LLM_REST_HTTP2_ENABLED", "llmRest.http2Enabled") }

    private fun Config.resolveHealthEnabled(): Boolean =
        with(ConfigEnv) { bool("LLM_REST_HEALTH_ENABLED", "llmRest.healthEnabled") }

    private fun Config.resolveHealthPath(): String =
        with(ConfigEnv) {
            strOrDefault(
                "LLM_REST_HEALTH_PATH",
                "llmRest.healthPath",
                "/health",
            )
        }

    private fun Config.resolveHealthInterval(): Long =
        with(ConfigEnv) {
            long(
                "LLM_REST_HEALTH_INTERVAL_MILLIS",
                "llmRest.healthIntervalMillis",
            )
        }

    private fun Config.resolveHealthThreshold(): Int =
        with(ConfigEnv) {
            int(
                "LLM_REST_HEALTH_FAILURE_THRESHOLD",
                "llmRest.healthFailureThreshold",
            )
        }

    private fun Config.resolveHealthRestartCommand(): String =
        with(ConfigEnv) {
            strOrDefault(
                "LLM_REST_HEALTH_RESTART_COMMAND",
                "llmRest.healthRestartCommand",
                "./bot-restart.sh",
            )
        }

    private fun resolveRestartContainers(config: Config): List<String> =
        with(ConfigEnv) {
            env("LLM_REST_HEALTH_RESTART_CONTAINERS")
                ?.takeIf { it.isNotBlank() }
                ?.let { parseList(it, emptyList()) }
                ?: runCatching { config.getValue("llmRest.healthRestartContainers") }
                    .mapCatching { value ->
                        when (value.valueType()) {
                            ConfigValueType.LIST -> config.getStringList("llmRest.healthRestartContainers")
                            ConfigValueType.STRING -> parseList(value.unwrapped().toString(), emptyList())
                            ConfigValueType.NULL -> emptyList()
                            else -> emptyList()
                        }
                    }
                    .getOrDefault(emptyList())
        }

    private fun Config.resolveHealthDockerSocket(): String =
        with(ConfigEnv) {
            strOrDefault(
                "LLM_REST_HEALTH_DOCKER_SOCKET",
                "llmRest.healthDockerSocket",
                "/var/run/docker.sock",
            )
        }

    private fun Config.resolveHealthRestartLockKey(): String =
        with(ConfigEnv) {
            strOrDefault(
                "LLM_REST_HEALTH_RESTART_LOCK_KEY",
                "llmRest.healthRestartLockKey",
                "",
            )
        }

    private fun Config.resolveHealthRestartLockTtlSeconds(): Long =
        with(ConfigEnv) {
            env("LLM_REST_HEALTH_RESTART_LOCK_TTL_SECONDS")?.toLongOrNull()
                ?: runCatching { getLong("llmRest.healthRestartLockTtlSeconds") }.getOrNull()
                ?: RedisConstants.LOCK_TTL_SECONDS
        }
}
