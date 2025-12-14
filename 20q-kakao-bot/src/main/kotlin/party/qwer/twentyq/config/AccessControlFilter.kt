package party.qwer.twentyq.config

import org.springframework.http.HttpStatus
import org.springframework.http.MediaType
import org.springframework.stereotype.Component
import org.springframework.web.server.ServerWebExchange
import org.springframework.web.server.WebFilter
import org.springframework.web.server.WebFilterChain
import party.qwer.twentyq.util.common.security.resolveAccessDenialReason
import party.qwer.twentyq.util.game.GameMessageProvider
import reactor.core.publisher.Mono

/**
 * API 접근 제어 필터
 */
@Component
class AccessControlFilter(
    private val appProperties: AppProperties,
    private val messageProvider: GameMessageProvider,
) : WebFilter {
    companion object {
        private val ACCESS_DENIED_STATUS = HttpStatus.FORBIDDEN
        private const val ACCESS_DENIED_CODE = "FORBIDDEN"
    }

    override fun filter(
        exchange: ServerWebExchange,
        chain: WebFilterChain,
    ): Mono<Void> =
        when {
            !shouldFilter(exchange.request.uri.path) -> chain.filter(exchange)
            else -> {
                val denialReason = getAccessDenialReason(exchange)
                if (denialReason != null) {
                    writeAccessDeniedError(exchange, denialReason)
                } else {
                    chain.filter(exchange)
                }
            }
        }

    private fun shouldFilter(path: String): Boolean =
        path.startsWith("/api/twentyq/riddles") &&
            !path.startsWith("/actuator") &&
            !path.startsWith("/error")

    private fun getAccessDenialReason(exchange: ServerWebExchange): String? {
        val userId = exchange.request.headers.getFirst("X-User-Id")
        val chatId = exchange.request.headers.getFirst("X-Session-Id")

        return resolveAccessDenialReason(
            access = appProperties.access,
            userId = userId,
            chatId = chatId,
        )
    }

    private fun writeAccessDeniedError(
        exchange: ServerWebExchange,
        messageKey: String,
    ): Mono<Void> {
        val msg = messageProvider.get(messageKey)
        return writeError(exchange, msg)
    }

    private fun writeError(
        exchange: ServerWebExchange,
        message: String,
    ): Mono<Void> {
        val response = exchange.response
        response.statusCode = ACCESS_DENIED_STATUS
        response.headers.contentType = MediaType.APPLICATION_JSON

        val errorJson = """{"error":"$ACCESS_DENIED_CODE","message":"$message"}"""
        val buffer = response.bufferFactory().wrap(errorJson.toByteArray())

        return response.writeWith(Mono.just(buffer))
    }
}
