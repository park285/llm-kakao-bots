package party.qwer.twentyq.rest

import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.mockk
import io.mockk.verify
import kotlinx.coroutines.runBlocking
import org.junit.jupiter.api.Test
import party.qwer.twentyq.redis.RestartLock

class LlmHealthMonitorTest {
    private val properties =
        LlmRestProperties(
            baseUrl = "http://localhost:8080",
            timeoutSeconds = 1,
            connectTimeoutSeconds = 1,
            healthEnabled = true,
            healthPath = "/health",
            healthIntervalMillis = 10,
            healthFailureThreshold = 2,
            healthRestartCommand = "echo restart",
        )

    @Test
    fun `should trigger restart after threshold`() =
        runBlocking {
            val restClient = mockk<LlmRestClient>()
            val executor = mockk<CommandExecutor>()
            val restartLock = mockk<RestartLock>(relaxed = true)
            coEvery { restClient.checkHealth(any()) } returnsMany
                listOf(
                    HealthCheckResult.Unhealthy(HealthFailureReason.HTTP_ERROR, "status=500"),
                    HealthCheckResult.Unhealthy(HealthFailureReason.HTTP_ERROR, "status=500"),
                )
            every { executor.execute(any()) } returns 0

            val monitor = LlmHealthMonitor(properties, restClient, executor, restartLock)
            monitor.checkHealthOnce()

            verify(exactly = 1) { executor.execute(match { it.contains("echo") }) }
        }

    @Test
    fun `should reset failure count on success`() =
        runBlocking {
            val restClient = mockk<LlmRestClient>()
            val executor = mockk<CommandExecutor>()
            val restartLock = mockk<RestartLock>(relaxed = true)
            coEvery { restClient.checkHealth(any()) } returnsMany
                listOf(
                    HealthCheckResult.Unhealthy(HealthFailureReason.HTTP_ERROR, "status=500"),
                    HealthCheckResult.Healthy,
                    HealthCheckResult.Unhealthy(HealthFailureReason.HTTP_ERROR, "status=500"),
                    HealthCheckResult.Unhealthy(HealthFailureReason.HTTP_ERROR, "status=500"),
                )
            every { executor.execute(any()) } returns 0

            val monitor = LlmHealthMonitor(properties, restClient, executor, restartLock)
            monitor.checkHealthOnce() // 1st fail -> retry success
            monitor.checkHealthOnce() // fail -> retry fail -> restart

            verify(exactly = 1) { executor.execute(any()) }
        }

    @Test
    fun `should skip restart when lock contended`() =
        runBlocking {
            val restClient = mockk<LlmRestClient>()
            val executor = mockk<CommandExecutor>()
            val restartLock = mockk<RestartLock>()
            val propertiesWithLock =
                properties.copy(
                    healthFailureThreshold = 1,
                    healthRestartLockKey = "shared:watchdog:restart:mcp-llm-server",
                    healthRestartLockTtlSeconds = 120,
                )

            coEvery { restClient.checkHealth(any()) } returns
                HealthCheckResult.Unhealthy(HealthFailureReason.HTTP_ERROR, "status=500")
            coEvery { restartLock.tryAcquire(any(), any()) } returns false
            every { executor.execute(any()) } returns 0

            val monitor = LlmHealthMonitor(propertiesWithLock, restClient, executor, restartLock)
            monitor.checkHealthOnce()

            coVerify(exactly = 1) { restartLock.tryAcquire("shared:watchdog:restart:mcp-llm-server", 120) }
            coVerify(exactly = 0) { restartLock.release(any()) }
            verify(exactly = 0) { executor.execute(any()) }
        }
}
