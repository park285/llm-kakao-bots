package io.github.kapu.turtlesoup.rest

import io.github.kapu.turtlesoup.config.Settings
import io.github.kapu.turtlesoup.redis.LockManager
import io.github.oshai.kotlinlogging.KotlinLogging
import io.ktor.client.HttpClient
import io.ktor.client.engine.cio.CIO
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.defaultRequest
import io.ktor.client.request.post
import io.ktor.client.request.unixSocket
import io.ktor.http.HttpStatusCode
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import java.io.File

interface CommandExecutor {
    fun execute(command: List<String>): Int
}

class ProcessCommandExecutor : CommandExecutor {
    override fun execute(command: List<String>): Int =
        ProcessBuilder(command)
            .directory(File("."))
            .start()
            .waitFor()
}

class LlmHealthMonitor(
    private val settings: Settings,
    private val restClient: LlmRestClient,
    private val lockManager: LockManager,
    private val commandExecutor: CommandExecutor = ProcessCommandExecutor(),
    private val scope: CoroutineScope = CoroutineScope(SupervisorJob() + Dispatchers.Default),
) {
    private companion object {
        private const val DOCKER_TIMEOUT_MILLIS = 30_000L
        private const val HEALTH_RETRY_DELAY_MILLIS = 1_000L
        private val logger = KotlinLogging.logger {}
    }

    private var job: Job? = null
    private val startupAtMillis: Long = System.currentTimeMillis()
    private var consecutiveFailures: Int = 0
    private val dockerClient: HttpClient =
        HttpClient(CIO) {
            install(HttpTimeout) {
                requestTimeoutMillis = DOCKER_TIMEOUT_MILLIS
                connectTimeoutMillis = DOCKER_TIMEOUT_MILLIS
            }
            defaultRequest {
                if (settings.llmRest.healthDockerSocket.isNotBlank()) {
                    unixSocket(settings.llmRest.healthDockerSocket)
                }
            }
        }

    fun start() {
        if (!settings.llmRest.healthEnabled || job != null) {
            return
        }

        job =
            scope.launch {
                while (isActive) {
                    checkHealth()
                    delay(settings.llmRest.healthIntervalMillis)
                }
            }
        val intervalMillis = settings.llmRest.healthIntervalMillis
        val threshold = settings.llmRest.healthFailureThreshold
        logger.info { "llm_health_monitor_started interval=${intervalMillis}ms threshold=$threshold" }
    }

    fun stop() {
        val oldJob = job
        job = null
        oldJob?.cancel()

        runBlocking {
            kotlinx.coroutines.withTimeoutOrNull(DOCKER_TIMEOUT_MILLIS) {
                oldJob?.join()
            }
        }

        dockerClient.close()
        logger.info { "llm_health_monitor_stopped" }
    }

    suspend fun checkHealth() {
        val threshold = maxOf(1, settings.llmRest.healthFailureThreshold)
        consecutiveFailures = 0
        repeat(threshold) { attemptIndex ->
            val healthy = restClient.isHealthy()
            if (healthy) {
                if (consecutiveFailures > 0) {
                    logger.info { "llm_health_recovered consecutive=$consecutiveFailures" }
                }
                consecutiveFailures = 0
                return
            }

            consecutiveFailures += 1
            if (attemptIndex < threshold - 1) {
                delay(HEALTH_RETRY_DELAY_MILLIS)
            }
        }

        logger.warn { "llm_health_cycle_failed consecutive=$consecutiveFailures threshold=$threshold" }
        triggerRestart()
        consecutiveFailures = 0
    }

    private suspend fun triggerRestart() {
        val lockKey = settings.llmRest.healthRestartLockKey.trim()
        if (lockKey.isNotEmpty()) {
            val ttlSeconds = maxOf(1L, settings.llmRest.healthRestartLockTtlSeconds)
            val acquired =
                runCatching { lockManager.tryAcquireSharedLock(lockKey, ttlSeconds) }
                    .onFailure { error ->
                        logger.warn(error) { "llm_restart_lock_error key=$lockKey" }
                    }.getOrNull()

            if (acquired == null) {
                performRestart()
                return
            }

            if (!acquired) {
                logger.info { "llm_restart_lock_skip key=$lockKey" }
                return
            }

            logger.info { "llm_restart_lock_acquired key=$lockKey ttl_seconds=$ttlSeconds" }
            try {
                performRestart()
            } finally {
                runCatching { lockManager.releaseSharedLock(lockKey) }
                    .onFailure { error ->
                        logger.warn(error) { "llm_restart_lock_release_error key=$lockKey" }
                    }
            }
            return
        }

        performRestart()
    }

    private suspend fun performRestart() {
        val command = settings.llmRest.healthRestartCommand.trim()
        if (command.isNotEmpty()) {
            val tokens = command.split("\\s+".toRegex())
            val exitCode =
                runCatching { commandExecutor.execute(tokens) }
                    .onFailure { logger.error { "llm_restart_error command=$command error=${it.message}" } }
                    .getOrNull()

            if (exitCode != null && exitCode == 0) {
                logger.info { "llm_restart_executed command=$command exit_code=$exitCode" }
                return
            }
            logger.warn { "llm_restart_cmd_failed command=$command exit_code=${exitCode ?: "null"}" }
        }

        val restarted = restartViaDocker()
        if (!restarted) {
            logger.warn { "llm_restart_skip command_missing_or_failed" }
        }
    }

    private suspend fun restartViaDocker(): Boolean {
        val targets = settings.llmRest.healthRestartContainers
        val socketPath = settings.llmRest.healthDockerSocket
        if (targets.isEmpty() || socketPath.isBlank()) {
            logger.warn { "llm_restart_docker_skip reason=targets_or_socket_missing" }
            return false
        }
        if (!File(socketPath).exists()) {
            logger.warn { "llm_restart_docker_skip reason=socket_missing socket=$socketPath" }
            return false
        }

        var success = false
        targets.forEach { container ->
            val result =
                runCatching {
                    dockerClient.post("http://docker/containers/$container/restart")
                }.onFailure { error ->
                    logger.warn(error) { "llm_restart_docker_fail container=$container" }
                }.getOrNull()

            if (result != null && result.status == HttpStatusCode.NoContent) {
                success = true
                logger.info { "llm_restart_docker_ok container=$container" }
            } else if (result != null) {
                logger.warn { "llm_restart_docker_status container=$container status=${result.status.value}" }
            }
        }

        return success
    }
}
