package party.qwer.twentyq.service

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.util.cache.CacheBuilders
import java.io.IOException
import java.net.HttpURLConnection
import java.net.URI
import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.time.Duration
import java.util.concurrent.CompletableFuture
import kotlin.time.Duration.Companion.hours
import kotlin.time.toJavaDuration
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

/**
 * 환율 서비스 - USD/KRW 환율을 동적으로 가져와서 캐싱
 * 
 * 사용 API: Frankfurter API (ECB 데이터 기반, 무료)  
 * 캐시 TTL: 1시간
 */
@Service
class ExchangeRateService {
    companion object {
        private val log = LoggerFactory.getLogger(ExchangeRateService::class.java)

        // 기본 환율 (API 실패 시 fallback)
        private const val DEFAULT_USD_KRW_RATE = 1400.0

        // Frankfurter API - USD/KRW만 조회 (최적화)
        private const val EXCHANGE_RATE_API_URL = "https://api.frankfurter.app/latest?from=USD&to=KRW"
        private val HTTP_TIMEOUT = Duration.ofSeconds(10)

        // 캐시 키
        private const val CACHE_KEY = "USD_KRW"
    }

    private suspend fun <T> CompletableFuture<T>.await(): T =
        suspendCancellableCoroutine { continuation ->
            whenComplete { result, error ->
                if (error != null) {
                    continuation.resumeWithException(error)
                } else {
                    continuation.resume(result)
                }
            }

            continuation.invokeOnCancellation { cancel(true) }
        }

    private val httpClient: HttpClient =
        HttpClient
            .newBuilder()
            .connectTimeout(HTTP_TIMEOUT)
            .build()

    // 환율 캐시 (1시간 TTL)
    private val rateCache =
        CacheBuilders.expireAfterWrite<String, Double>(
            maxSize = 10L,
            ttl = 1.hours.toJavaDuration(),
            recordStats = false,
        )

    private val fetchMutex = Mutex()

    /**
     * USD -> KRW 환율 조회 (캐시 우선, 없으면 API 호출)
     */
    suspend fun getUsdKrwRate(): Double {
        // 캐시 확인
        rateCache.getIfPresent(CACHE_KEY)?.let { return it }

        // 동시 요청 방지를 위한 뮤텍스
        return fetchMutex.withLock {
            // 다른 스레드가 이미 가져왔는지 재확인
            rateCache.getIfPresent(CACHE_KEY)?.let { return@withLock it }

            // API 호출
            val rate = fetchExchangeRateFromApi()
            rateCache.put(CACHE_KEY, rate)
            log.info("EXCHANGE_RATE_FETCHED rate={}", rate)
            rate
        }
    }

    /**
     * USD 금액을 KRW로 변환
     */
    suspend fun usdToKrw(usdAmount: Double): Double {
        val rate = getUsdKrwRate()
        return usdAmount * rate
    }

    /**
     * 현재 환율 정보 (표시용)
     */
    suspend fun getRateInfo(): String {
        val rate = getUsdKrwRate()
        return "1 USD = ${String.format("%,.0f", rate)} KRW"
    }

    /**
     * Frankfurter API 호출 - USD/KRW만 조회
     * 응답 예시: {"amount":1,"base":"USD","date":"2025-12-06","rates":{"KRW":1420.5}}
     */
    private suspend fun fetchExchangeRateFromApi(): Double =
        try {
            val request =
                HttpRequest
                    .newBuilder()
                    .uri(URI.create(EXCHANGE_RATE_API_URL))
                    .timeout(HTTP_TIMEOUT)
                    .GET()
                    .build()

            val response =
                httpClient
                    .sendAsync(request, HttpResponse.BodyHandlers.ofString())
                    .await()

            if (response.statusCode() == HttpURLConnection.HTTP_OK) {
                parseKrwRate(response.body())
            } else {
                log.warn("EXCHANGE_RATE_API_FAILED status={}", response.statusCode())
                DEFAULT_USD_KRW_RATE
            }
        } catch (e: CancellationException) {
            throw e
        } catch (e: Exception) {
            val rootCause = e.cause ?: e
            when (rootCause) {
                is IOException -> {
                    log.warn("EXCHANGE_RATE_FETCH_ERROR error={}", rootCause.message)
                    DEFAULT_USD_KRW_RATE
                }

                is InterruptedException -> {
                    Thread.currentThread().interrupt()
                    log.warn("EXCHANGE_RATE_FETCH_INTERRUPTED error={}", rootCause.message)
                    DEFAULT_USD_KRW_RATE
                }

                else -> {
                    log.warn("EXCHANGE_RATE_FETCH_UNEXPECTED error={}", rootCause.message)
                    DEFAULT_USD_KRW_RATE
                }
            }
        }

    /**
     * Frankfurter API 응답에서 KRW 환율 파싱
     * 응답 형식: {"rates":{"KRW":1420.5}}
     */
    private fun parseKrwRate(jsonBody: String): Double {
        // 간단한 정규식 파싱 (Jackson 의존성 회피)
        val regex = """"KRW"\s*:\s*([\d.]+)""".toRegex()
        val match = regex.find(jsonBody)
        val rate = match?.groupValues?.get(1)?.toDoubleOrNull()
        if (rate == null) {
            log.warn("EXCHANGE_RATE_PARSE_FAILED")
            return DEFAULT_USD_KRW_RATE
        }
        return rate
    }

    /**
     * 캐시 강제 갱신
     */
    suspend fun refreshRate(): Double {
        rateCache.invalidate(CACHE_KEY)
        return getUsdKrwRate()
    }
}
