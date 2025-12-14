package party.qwer.twentyq.service

import org.slf4j.LoggerFactory
import org.springframework.cache.get
import org.springframework.stereotype.Service
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.rest.TwentyQRestClient
import party.qwer.twentyq.util.logging.LoggingConstants

/** 답변 텍스트 정규화 서비스 */
@Service
class NormalizeService(
    private val restClient: TwentyQRestClient,
    private val complexityAnalyzer: ComplexityAnalyzer,
    private val cacheManager: org.springframework.cache.CacheManager,
    private val normalizationConfig: NormalizationConfig,
) {
    private val cacheEnabled = normalizationConfig.cacheEnabled

    companion object {
        private val log = LoggerFactory.getLogger(NormalizeService::class.java)
        private const val CACHE_NAME = "normalize"

        private val EMOJI_PATTERN = Regex("[\uD800-\uDFFF]|[\u2600-\u27BF]|[\uD83C-\uDBFF\uDC00-\uDFFF]+")

        // 정규화 패턴 (사전 컴파일 - 성능 최적화)
        private val REPEATED_QUESTION = Regex("\\?{2,}")
        private val REPEATED_EXCLAIM = Regex("!{2,}")
        private val REPEATED_DOTS = Regex("\\.{3,}")
        private val REPEATED_KIEUK = Regex("ㅋ{3,}")
        private val REPEATED_HIEUH = Regex("ㅎ{3,}")
        private val ENDING_YEO = Regex("([가-힣])ㅕ")
        private val ENDING_YO = Regex("([가-힣])ㅛ")
        private val TYPO_IT = Regex("잇([나요|어요|습니다|는지|을까])")
        private val SPACE_SU_IT = Regex("([을를])수(있)")
        private val PAST_TENSE_TYPO = Regex("([가-힣])엇([어요|습니다|는지])")
        private val FUTURE_TENSE_TYPO = Regex("([가-힣])겟([어요|습니다|는지])")
        private val TYPO_GWENCHAN = Regex("괜찬([아요은])")
        private val TYPO_WAT = Regex("왓([어요었])")
        private val TYPO_DWAESS = Regex("됬([어요었])")
        private val TYPO_DOEYO = Regex("되요")
        private val TYPO_HARYEOGU = Regex("하려구")
        private val TYPO_L_KKE = Regex("([가-힣])ㄹ께")
        private val REPEATED_SPACES = Regex("\\s{2,}")
    }

    suspend fun normalize(text: String): NormalizeResponse {
        val result =
            if (!cacheEnabled) {
                normalizeInternal(text)
            } else {
                val cached = getCachedValue(text)
                if (cached != null) {
                    log.debugL { "CACHE_HIT text='${text.take(LoggingConstants.LOG_TEXT_SHORT)}'" }
                    cached
                } else {
                    log.info("CACHE_MISS text='{}'", text.take(LoggingConstants.LOG_TEXT_SHORT))
                    val computed = normalizeInternal(text)
                    saveToCacheWithTtl(text, computed)
                    computed
                }
            }

        return result
    }

    private fun getCachedValue(text: String): NormalizeResponse? =
        kotlin
            .runCatching { cacheManager.getCache(CACHE_NAME)?.get<NormalizeResponse>(text) }
            .onFailure { e -> log.warn("Cache read failed: {}", e.message) }
            .getOrNull()

    private fun saveToCacheWithTtl(
        text: String,
        response: NormalizeResponse,
    ) {
        // 이모지 포함 시 캐시 저장 skip (StackOverflowError 방지)
        if (text.contains(EMOJI_PATTERN)) {
            log.debugL { "CACHE_SKIP text='${text.take(LoggingConstants.LOG_TEXT_SHORT)}' (contains emoji)" }
            return
        }

        kotlin
            .runCatching {
                val cache = cacheManager.getCache(CACHE_NAME)
                if (cache == null) {
                    log.warn("Cache UNAVAILABLE: name={} (CacheManager returned null)", CACHE_NAME)
                    return@runCatching
                }

                cache.put(text, response)
                log.debugL { "CACHE_WRITE_SUCCESS text='${text.take(LoggingConstants.LOG_TEXT_SHORT)}'" }
            }.onFailure { e ->
                if (e is StackOverflowError) {
                    log.debugL {
                        "CACHE_SKIP text='${text.take(LoggingConstants.LOG_TEXT_SHORT)}' " +
                            "(StackOverflowError, likely emoji)"
                    }
                } else {
                    log.warn(
                        "Cache write failed: text='{}', error='{}'",
                        text.take(LoggingConstants.LOG_TEXT_SHORT),
                        e.message ?: e::class.simpleName,
                    )
                }
            }
    }

    private suspend fun normalizeInternal(text: String): NormalizeResponse {
        val normalized =
            if (text.isBlank()) {
                NormalizeResponse(normalized = text.trim())
            } else {
                val afterRegex = normalizeByRegex(text)
                if (complexityAnalyzer.hasComplexTypo(afterRegex)) {
                    log.info("normalize FALLBACK_TO_LLM (complex typo detected): '{}'", afterRegex)
                    normalizeWithLLM(afterRegex)
                } else {
                    log.debugL {
                        "normalize DONE (no LLM): '${text.take(LoggingConstants.LOG_TEXT_SHORT)}' -> '$afterRegex'"
                    }
                    NormalizeResponse(normalized = afterRegex.trim())
                }
            }

        return normalized
    }

    private suspend fun normalizeWithLLM(afterRegex: String): NormalizeResponse {
        val response =
            runCatching { restClient.normalizeQuestion(afterRegex) }
                .onFailure { throwable ->
                    when (throwable) {
                        is kotlin.coroutines.cancellation.CancellationException -> throw throwable
                        is Error -> throw throwable
                    }
                    log.info(
                        "normalize LLM failed, using preprocessed text: {}",
                        throwable.message,
                    )
                }.getOrNull()
                ?: return NormalizeResponse(normalized = afterRegex.trim())

        return when {
            response.isError -> {
                log.warn("normalize MCP_ERROR: {}", response.errorMessage)
                NormalizeResponse(normalized = afterRegex.trim())
            }
            response.normalized.isNotBlank() -> {
                log.info("normalize MCP_SUCCESS: '{}' -> '{}'", afterRegex, response.normalized)
                NormalizeResponse(normalized = response.normalized)
            }
            else -> {
                log.warn("normalize MCP_EMPTY, using preprocessed text")
                NormalizeResponse(normalized = afterRegex.trim())
            }
        }
    }

    private fun normalizeByRegex(text: String): String =
        kotlin
            .runCatching {
                // 정규식 패턴으로 일반적인 오타 교정 (사전 컴파일된 패턴 사용)
                var normalized = text

                // 1) 연속 특수문자 정규화
                normalized = normalized.replace(REPEATED_QUESTION, "?")
                normalized = normalized.replace(REPEATED_EXCLAIM, "!")
                normalized = normalized.replace(REPEATED_DOTS, " ")

                // 2) 이모티콘 정규화
                normalized = normalized.replace(REPEATED_KIEUK, "ㅋㅋ")
                normalized = normalized.replace(REPEATED_HIEUH, "ㅎㅎ")

                // 3) 종결어미 오타: ㅕ, ㅛ → ?
                normalized = normalized.replace(ENDING_YEO, "$1?")
                normalized = normalized.replace(ENDING_YO, "$1?")

                // 4) "있다" 오타: 잇 → 있
                normalized = normalized.replace(TYPO_IT, "있$1")

                // 5) "~을 수 있다" 띄어쓰기
                normalized = normalized.replace(SPACE_SU_IT, "$1 수 $2")
                normalized = normalized.replace("수잇", "수 있")

                // 6) 과거형 오타: 엇 → 었
                normalized = normalized.replace(PAST_TENSE_TYPO, "$1었$2")

                // 7) 미래형 오타: 겟 → 겠
                normalized = normalized.replace(FUTURE_TENSE_TYPO, "$1겠$2")

                // 8) 네이버 커버리지 보완 (고빈도 오타)
                normalized = normalized.replace(TYPO_GWENCHAN, "괜찮$1")
                normalized = normalized.replace(TYPO_WAT, "왔$1")
                normalized = normalized.replace(TYPO_DWAESS, "됐$1")
                normalized = normalized.replace("않되", "안돼")

                // 9) 최고빈도 오타 패턴
                normalized = normalized.replace(TYPO_DOEYO, "돼요")
                normalized = normalized.replace(TYPO_HARYEOGU, "하려고")
                normalized = normalized.replace(TYPO_L_KKE, "$1ㄹ게")

                // 10) 공백 정규화
                normalized = normalized.replace(REPEATED_SPACES, " ")

                if (normalized != text) {
                    log.info("normalizeByRegex SUCCESS: '{}' -> '{}'", text, normalized)
                }

                normalized
            }.onFailure { e -> log.warn("Regex normalization failed for text='{}': {}", text, e.message) }
            .getOrElse { text }
}
