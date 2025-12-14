package io.github.kapu.turtlesoup.api

import io.github.kapu.turtlesoup.config.Settings
import io.github.kapu.turtlesoup.rest.LlmRestClient
import io.github.oshai.kotlinlogging.KotlinLogging
import io.ktor.http.HttpStatusCode
import io.ktor.server.application.Application
import io.ktor.server.response.respond
import io.ktor.server.routing.get
import io.ktor.server.routing.routing
import kotlinx.serialization.Serializable
import org.koin.ktor.ext.inject

private val logger = KotlinLogging.logger {}

@Serializable
data class LlmDebugTransport(
    val baseUrl: String,
    val http2Enabled: Boolean,
    val timeoutSeconds: Long,
    val connectTimeoutSeconds: Long,
    val healthEnabled: Boolean,
    val healthPath: String,
    val healthIntervalMillis: Long,
    val healthFailureThreshold: Int,
    val healthRestartCommand: String,
    val healthRestartLockKey: String,
    val healthRestartLockTtlSeconds: Long,
)

@Serializable
data class LlmDebugResponse(
    val llmRest: LlmDebugTransport,
    val modelConfig: LlmRestClient.ModelConfigResponse?,
    val modelConfigStatus: String,
)

fun Application.configureDebugRoutes() {
    val settings: Settings by inject()
    val llmRestClient: LlmRestClient by inject()

    routing {
        get("/debug/models") {
            val llm = settings.llmRest
            val transport =
                LlmDebugTransport(
                    baseUrl = llm.baseUrl,
                    http2Enabled = llm.http2Enabled,
                    timeoutSeconds = llm.timeoutSeconds,
                    connectTimeoutSeconds = llm.connectTimeoutSeconds,
                    healthEnabled = llm.healthEnabled,
                    healthPath = llm.healthPath,
                    healthIntervalMillis = llm.healthIntervalMillis,
                    healthFailureThreshold = llm.healthFailureThreshold,
                    healthRestartCommand = llm.healthRestartCommand,
                    healthRestartLockKey = llm.healthRestartLockKey,
                    healthRestartLockTtlSeconds = llm.healthRestartLockTtlSeconds,
                )
            val modelConfig = llmRestClient.getModelConfig()
            val response =
                LlmDebugResponse(
                    llmRest = transport,
                    modelConfig = modelConfig,
                    modelConfigStatus = if (modelConfig == null) "unavailable" else "ok",
                )
            call.respond(HttpStatusCode.OK, response)
        }
    }

    logger.info { "debug_routes_configured" }
}
