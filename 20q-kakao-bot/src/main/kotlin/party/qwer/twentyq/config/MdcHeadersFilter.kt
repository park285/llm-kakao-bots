package party.qwer.twentyq.config

import org.slf4j.MDC
import org.springframework.stereotype.Component
import org.springframework.web.server.ServerWebExchange
import org.springframework.web.server.WebFilter
import org.springframework.web.server.WebFilterChain
import reactor.core.publisher.Mono

/**
 * MDC 헤더 추출 필터
 */
@Component
class MdcHeadersFilter : WebFilter {
    override fun filter(
        exchange: ServerWebExchange,
        chain: WebFilterChain,
    ): Mono<Void> {
        val path = exchange.request.uri.path

        // actuator/error 경로는 필터링 제외
        if (path.startsWith("/actuator") || path.startsWith("/error")) {
            return chain.filter(exchange)
        }

        val sessionId = exchange.request.headers.getFirst("X-Session-Id")
        val userId = exchange.request.headers.getFirst("X-User-Id")
        val userEmail = exchange.request.headers.getFirst("X-User-Email")

        return chain
            .filter(exchange)
            .doOnEach { signal ->
                if (!signal.isOnComplete && !signal.isOnError) {
                    if (!sessionId.isNullOrBlank()) MDC.put("sessionId", sessionId)
                    if (!userId.isNullOrBlank()) MDC.put("userId", userId)
                    if (!userEmail.isNullOrBlank()) MDC.put("userEmail", userEmail)
                }
            }.doFinally {
                MDC.remove("sessionId")
                MDC.remove("userId")
                MDC.remove("userEmail")
            }
    }
}
