package party.qwer.twentyq.service.riddle

import com.github.benmanes.caffeine.cache.Cache
import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.rest.GuardRestClient
import party.qwer.twentyq.util.cache.CacheBuilders
import party.qwer.twentyq.util.common.extensions.hours
import party.qwer.twentyq.util.game.constants.ValidationConstants
import party.qwer.twentyq.util.logging.LoggingConstants.LOG_TEXT_SHORT

/** 메타 질문 검증 서비스 (MCP Guard 기반) */
@Component
class MetaQuestionValidator(
    private val restClient: GuardRestClient,
    private val appProperties: AppProperties,
    private val questionPolicy: QuestionPolicy,
) {
    companion object {
        private val log = LoggerFactory.getLogger(MetaQuestionValidator::class.java)
    }

    private val cache: Cache<String, Boolean> =
        CacheBuilders
            .expireAfterWrite(
                ValidationConstants.META_QUESTION_CACHE_SIZE,
                1.hours,
                recordStats = true,
            )

    /** 사전 필터링: LLM 검증 필요 여부 판단 */
    suspend fun shouldValidate(question: String): Boolean {
        if (!appProperties.security.metaQuestionLlmEnabled) return false

        val policySuspicious =
            runCatching {
                questionPolicy.isAnswerLengthMetaQuestion(question) ||
                    questionPolicy.isAnswerIndexMetaQuestion(question) ||
                    questionPolicy.isAnswerBoundaryMetaQuestion(question)
            }.onFailure { ex ->
                log.warn("QuestionPolicy pre-filter failed: {}", ex.message)
            }.getOrDefault(false)

        if (policySuspicious) {
            log.debugL {
                "shouldValidate: SUSPICIOUS text='${question.take(LOG_TEXT_SHORT)}'"
            }
        }

        return policySuspicious
    }

    suspend fun refreshCache(): Boolean {
        log.info("MetaQuestionValidator cache REFRESH requested")
        cache.invalidateAll()
        return true
    }

    /** MCP Guard를 통한 메타 질문 검증 */
    suspend fun isMetaQuestion(question: String): Boolean {
        val key = question.trim()
        cache.getIfPresent(key)?.let { return it }

        val result = validateWithMcpGuard(question)
        cache.put(key, result)
        return result
    }

    private suspend fun validateWithMcpGuard(question: String): Boolean =
        runCatching {
            val isMalicious = restClient.isMalicious(question)
            log.info(
                "REST_GUARD question='{}', blocked={}",
                question.take(LOG_TEXT_SHORT),
                isMalicious,
            )
            isMalicious
        }.onFailure { e ->
            log.warn("REST Guard validation failed: {}, allowing question", e.message)
        }.getOrElse { false }

    fun getCacheStats(): String {
        val stats = cache.stats()
        return "hits=${stats.hitCount()}, misses=${stats.missCount()}, hitRate=${stats.hitRate()}"
    }
}
