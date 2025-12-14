package io.github.kapu.turtlesoup

import io.github.kapu.turtlesoup.api.configureDebugRoutes
import io.github.kapu.turtlesoup.api.configureGameRoutes
import io.github.kapu.turtlesoup.config.JsonConfig
import io.github.kapu.turtlesoup.config.Settings
import io.github.kapu.turtlesoup.config.appModule
import io.github.kapu.turtlesoup.mq.ValkeyMQStreamConsumer
import io.github.kapu.turtlesoup.rest.LlmHealthMonitor
import io.github.oshai.kotlinlogging.KotlinLogging
import io.ktor.http.HttpHeaders
import io.ktor.http.HttpMethod
import io.ktor.http.HttpStatusCode
import io.ktor.serialization.kotlinx.json.json
import io.ktor.server.application.Application
import io.ktor.server.application.ApplicationStopping
import io.ktor.server.application.install
import io.ktor.server.engine.embeddedServer
import io.ktor.server.netty.Netty
import io.ktor.server.plugins.calllogging.CallLogging
import io.ktor.server.plugins.contentnegotiation.ContentNegotiation
import io.ktor.server.plugins.cors.routing.CORS
import io.ktor.server.plugins.statuspages.StatusPages
import io.ktor.server.request.uri
import io.ktor.server.response.respond
import io.ktor.server.routing.get
import io.ktor.server.routing.routing
import org.koin.ktor.ext.inject
import org.koin.ktor.plugin.Koin
import org.koin.logger.slf4jLogger
import org.slf4j.event.Level

private val logger = KotlinLogging.logger {}

fun main() {
    val settings = Settings.load()

    embeddedServer(
        Netty,
        port = settings.server.port,
        host = settings.server.host,
        module = Application::module,
    ).start(wait = true)
}

fun Application.module() {
    installKoin()
    val settings: Settings by inject()
    val mqConsumer: ValkeyMQStreamConsumer by inject()
    val llmHealthMonitor: LlmHealthMonitor by inject()

    configureSerialization()
    configureCors()
    configureCallLogging()
    configureStatusPages()
    configureDebugRoutes()
    configureGameRoutes()
    configureHealthEndpoint()
    startMqConsumer(mqConsumer)
    startHealthMonitor(llmHealthMonitor)

    logger.info { "turtle_soup_bot_started host=${settings.server.host} port=${settings.server.port}" }
}

private fun Application.installKoin() {
    install(Koin) {
        slf4jLogger()
        modules(appModule)
    }
}

private fun Application.configureSerialization() {
    install(ContentNegotiation) {
        json(JsonConfig.prettyLenient)
    }
}

private fun Application.configureCors() {
    install(CORS) {
        allowMethod(HttpMethod.Options)
        allowMethod(HttpMethod.Get)
        allowMethod(HttpMethod.Post)
        allowMethod(HttpMethod.Put)
        allowMethod(HttpMethod.Delete)
        allowHeader(HttpHeaders.Authorization)
        allowHeader(HttpHeaders.ContentType)
        anyHost()
    }
}

private fun Application.configureCallLogging() {
    install(CallLogging) {
        level = Level.INFO
        filter { call -> call.request.uri.startsWith("/api") }
    }
}

private fun Application.configureStatusPages() {
    install(StatusPages) {
        exception<Throwable> { call, cause ->
            logger.error(cause) { "unhandled_exception path=${call.request.uri}" }
            call.respond(
                HttpStatusCode.InternalServerError,
                mapOf("error" to "INTERNAL_ERROR", "message" to (cause.message ?: "Unknown error")),
            )
        }
    }
}

private fun Application.configureHealthEndpoint() {
    routing {
        get("/health") {
            call.respond(HttpStatusCode.OK, mapOf("status" to "UP"))
        }
    }
}

private fun Application.startMqConsumer(mqConsumer: ValkeyMQStreamConsumer) {
    mqConsumer.start()
    logger.info { "mq_consumer_started" }

    monitor.subscribe(ApplicationStopping) {
        logger.info { "application_stopping" }
        mqConsumer.stop()
    }
}

private fun Application.startHealthMonitor(monitorService: LlmHealthMonitor) {
    monitorService.start()
    monitor.subscribe(ApplicationStopping) {
        monitorService.stop()
    }
}
