package party.qwer.twentyq.rest

import io.ktor.client.HttpClient
import io.ktor.client.engine.cio.CIO
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.plugins.defaultRequest
import io.ktor.client.request.post
import io.ktor.client.request.unixSocket
import io.ktor.http.HttpStatusCode
import jakarta.annotation.PreDestroy
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.withTimeoutOrNull
import org.slf4j.LoggerFactory
import org.springframework.scheduling.annotation.Scheduled
import org.springframework.stereotype.Component
import party.qwer.twentyq.redis.RestartLock
import java.io.File
import java.util.concurrent.atomic.AtomicBoolean

fun interface CommandExecutor {
    fun execute(command: List<String>): Int
}

@Component
class ProcessCommandExecutor : CommandExecutor {
    override fun execute(command: List<String>): Int =
        ProcessBuilder(command)
            .directory(File("."))
            .start()
            .waitFor()
}

@Component
class LlmHealthMonitor(
    private val properties: LlmRestProperties,
    private val llmRestClient: LlmRestClient,
    private val commandExecutor: CommandExecutor,
    private val restartLock: RestartLock,
) {
    companion object {
        private val log = LoggerFactory.getLogger(LlmHealthMonitor::class.java)
        private const val DOCKER_TIMEOUT_MILLIS = 30_000L
        private const val HEALTH_RETRY_DELAY_MILLIS = 1_000L
    }

    private var consecutiveFailures: Int = 0

    private val scopeJob = SupervisorJob()
    private val scope = CoroutineScope(scopeJob + Dispatchers.IO)
    private val healthCheckInProgress = AtomicBoolean(false)
    private val dockerClient: HttpClient =
        HttpClient(CIO) {
            install(HttpTimeout) {
                requestTimeoutMillis = DOCKER_TIMEOUT_MILLIS
                connectTimeoutMillis = DOCKER_TIMEOUT_MILLIS
            }
            defaultRequest {
                if (properties.healthDockerSocket.isNotBlank()) {
                    unixSocket(properties.healthDockerSocket)
                }
            }
        }

    @PreDestroy
    fun cleanup() {
        scopeJob.cancel()
        runBlocking {
            withTimeoutOrNull(DOCKER_TIMEOUT_MILLIS) {
                scopeJob.join()
            }
        }
        dockerClient.close()
        log.info("LLM_DOCKER_CLIENT_CLOSED")
    }

    @Scheduled(fixedDelayString = "\${llm.rest.health-interval-millis:60000}")
    fun checkHealth() {
        if (!properties.healthEnabled) {
            return
        }

        if (!healthCheckInProgress.compareAndSet(false, true)) {
            log.debug("LLM_HEALTH_SKIP reason=in_progress")
            return
        }

        scope
            .launch {
                try {
                    checkHealthOnce()
                } catch (e: kotlin.coroutines.cancellation.CancellationException) {
                    throw e
                } catch (e: Throwable) {
                    log.warn("LLM_HEALTH_CYCLE_ERROR error={}", e.message, e)
                }
            }.invokeOnCompletion { healthCheckInProgress.set(false) }
    }

    internal suspend fun checkHealthOnce() {
        val threshold = maxOf(1, properties.healthFailureThreshold)

        consecutiveFailures = 0
        var lastUnhealthy: HealthCheckResult.Unhealthy? = null
        repeat(threshold) { attemptIndex ->
            val result = llmRestClient.checkHealth(properties.healthPath)
            if (result is HealthCheckResult.Healthy) {
                if (consecutiveFailures > 0) {
                    log.info("LLM_HEALTH_RECOVERED consecutive={}", consecutiveFailures)
                }
                consecutiveFailures = 0
                return
            }

            val unhealthy = result as HealthCheckResult.Unhealthy
            lastUnhealthy = unhealthy
            consecutiveFailures += 1

            if (attemptIndex < threshold - 1) {
                delay(HEALTH_RETRY_DELAY_MILLIS)
            }
        }

        val unhealthy = lastUnhealthy ?: return
        log.warn(
            "LLM_HEALTH_CYCLE_FAIL consecutive={} threshold={} reason={} message={}",
            consecutiveFailures,
            threshold,
            unhealthy.reason,
            unhealthy.message ?: "",
        )

        triggerRestart(unhealthy.reason)
        consecutiveFailures = 0
    }

    private suspend fun triggerRestart(reason: HealthFailureReason) {
        val lockKey = properties.healthRestartLockKey.trim()
        if (lockKey.isNotEmpty()) {
            val ttlSeconds = maxOf(1L, properties.healthRestartLockTtlSeconds)
            val acquired =
                runCatching { restartLock.tryAcquire(lockKey, ttlSeconds) }
                    .onFailure { ex ->
                        log.warn(
                            "LLM_RESTART_LOCK_ERROR key={} failureReason={} error={}",
                            lockKey,
                            reason,
                            ex.message,
                            ex,
                        )
                    }.getOrNull()

            if (acquired == null) {
                performRestart(reason)
                return
            }

            if (!acquired) {
                log.info("LLM_RESTART_LOCK_SKIP key={} failureReason={}", lockKey, reason)
                return
            }

            log.info("LLM_RESTART_LOCK_OK key={} ttlSeconds={} failureReason={}", lockKey, ttlSeconds, reason)
            try {
                performRestart(reason)
            } finally {
                runCatching { restartLock.release(lockKey) }
                    .onFailure { ex ->
                        log.warn(
                            "LLM_RESTART_LOCK_RELEASE_ERROR key={} failureReason={} error={}",
                            lockKey,
                            reason,
                            ex.message,
                            ex,
                        )
                    }
            }
            return
        }

        performRestart(reason)
    }

    private suspend fun performRestart(reason: HealthFailureReason) {
        val command = properties.healthRestartCommand.trim()
        val exitCode = executeRestartCommand(command, reason)
        if (exitCode != null && exitCode == 0) {
            return
        }

        val restarted = restartViaDocker(reason)
        if (restarted) {
            return
        }

        if (command.isEmpty()) {
            log.warn("LLM_RESTART_SKIP reason=command_missing failureReason={}", reason)
        } else {
            log.warn(
                "LLM_RESTART_SKIP reason=command_failed_or_unavailable failureReason={} exitCode={}",
                reason,
                exitCode ?: "null",
            )
        }
    }

    private fun executeRestartCommand(
        command: String,
        reason: HealthFailureReason,
    ): Int? {
        val tokens = command.split("\\s+".toRegex()).filter { it.isNotBlank() }
        if (tokens.isEmpty()) {
            return null
        }

        val first = tokens.first()
        val looksLikePath = first.startsWith("/") || first.startsWith(".") || first.contains("/")
        if (looksLikePath && !File(first).exists()) {
            log.warn("LLM_RESTART_CMD_SKIP reason=command_not_found command={} failureReason={}", command, reason)
            return null
        }

        val exitCode =
            runCatching { commandExecutor.execute(tokens) }
                .onFailure { log.error("LLM_RESTART_ERROR command={} reason={} error={}", command, reason, it.message) }
                .getOrNull()

        when {
            exitCode == 0 -> log.info("LLM_RESTART_CMD_OK command={} reason={} exitCode={}", command, reason, exitCode)
            exitCode != null ->
                log.warn(
                    "LLM_RESTART_CMD_FAIL command={} reason={} exitCode={}",
                    command,
                    reason,
                    exitCode,
                )
        }

        return exitCode
    }

    private suspend fun restartViaDocker(reason: HealthFailureReason): Boolean {
        val targets = properties.healthRestartContainers
        val socketPath = properties.healthDockerSocket
        if (!canRestartViaDocker(targets, socketPath, reason)) {
            return false
        }

        return restartDockerContainers(targets, reason)
    }

    private fun canRestartViaDocker(
        targets: List<String>,
        socketPath: String,
        reason: HealthFailureReason,
    ): Boolean {
        if (targets.isEmpty() || socketPath.isBlank()) {
            log.warn(
                "LLM_RESTART_DOCKER_SKIP reason=targets_or_socket_missing failureReason={}",
                reason,
            )
            return false
        }
        if (!File(socketPath).exists()) {
            log.warn(
                "LLM_RESTART_DOCKER_SKIP reason=socket_missing socket={} failureReason={}",
                socketPath,
                reason,
            )
            return false
        }

        return true
    }

    private suspend fun restartDockerContainers(
        targets: List<String>,
        reason: HealthFailureReason,
    ): Boolean {
        var success = false
        targets.forEach { container ->
            if (restartDockerContainer(container, reason)) {
                success = true
            }
        }

        return success
    }

    private suspend fun restartDockerContainer(
        container: String,
        reason: HealthFailureReason,
    ): Boolean {
        val response =
            runCatching {
                dockerClient.post("http://docker/containers/$container/restart")
            }.onFailure { throwable ->
                when (throwable) {
                    is kotlin.coroutines.cancellation.CancellationException -> throw throwable
                    is Error -> throw throwable
                }
                log.warn(
                    "LLM_RESTART_DOCKER_ERROR container={} failureReason={}",
                    container,
                    reason,
                    throwable,
                )
            }.getOrNull()

        if (response == null) {
            return false
        }

        if (response.status == HttpStatusCode.NoContent) {
            log.info(
                "LLM_RESTART_DOCKER_OK container={} failureReason={}",
                container,
                reason,
            )
            return true
        }

        log.warn(
            "LLM_RESTART_DOCKER_STATUS container={} failureReason={} status={}",
            container,
            reason,
            response.status.value,
        )
        return false
    }
}
