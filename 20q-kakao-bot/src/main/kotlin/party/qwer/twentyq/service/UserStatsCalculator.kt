package party.qwer.twentyq.service

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.model.BestScore
import party.qwer.twentyq.model.CategoryStat
import tools.jackson.core.JacksonException
import tools.jackson.core.type.TypeReference
import tools.jackson.databind.json.JsonMapper
import tools.jackson.module.kotlin.kotlinModule
import java.time.Instant

/**
 * 사용자 통계 계산 담당 클래스
 */
@Component
class UserStatsCalculator {
    companion object {
        private val log = LoggerFactory.getLogger(UserStatsCalculator::class.java)
        private val objectMapper = JsonMapper.builder().addModule(kotlinModule()).build()

        /**
         * Composite ID 생성
         */
        fun compositeId(
            chatId: String,
            userId: String,
        ): String = "$chatId:$userId"

        /**
         * Category Stats JSON 직렬화
         */
        fun serializeCategoryStats(categoryStats: Map<String, CategoryStat>): String =
            try {
                objectMapper.writeValueAsString(categoryStats)
            } catch (e: JacksonException) {
                log.error("CATEGORY_STATS_SERIALIZE_FAILED error={}", e.message, e)
                "{}"
            }

        /**
         * Category Stats JSON 역직렬화
         */
        fun parseCategoryStatsJson(json: String?): Map<String, CategoryStat> {
            if (json.isNullOrBlank()) return emptyMap()
            return try {
                objectMapper.readValue(
                    json,
                    object : TypeReference<Map<String, CategoryStat>>() {},
                )
            } catch (e: JacksonException) {
                log.error("CATEGORY_STATS_PARSE_FAILED json={}, error={}", json, e.message, e)
                emptyMap()
            }
        }

        /**
         * 최고 기록 계산 (총 시도 횟수 = 질문 + 오답)
         */
        fun computeBestScore(
            currentBest: Int?,
            currentBestWrongGuess: Int?,
            questionCount: Int,
            wrongGuessCount: Int,
            result: GameResult,
            target: String?,
            category: String,
        ): BestScoreUpdate? {
            if (result != GameResult.CORRECT || target == null) return null

            val totalAttempts = questionCount + wrongGuessCount
            val currentTotal = (currentBest ?: Int.MAX_VALUE) + (currentBestWrongGuess ?: 0)

            // 총 시도 횟수가 더 적을 때만 갱신
            if (totalAttempts >= currentTotal) return null

            return BestScoreUpdate(questionCount, wrongGuessCount, target, category)
        }

        /**
         * 전체 통계 계산
         */
        fun computeTotalStats(
            existingSurrenders: Int,
            result: GameResult,
        ): Int =
            if (result == GameResult.SURRENDER) {
                existingSurrenders + 1
            } else {
                existingSurrenders
            }

        /**
         * 카테고리별 베스트 스코어 계산 (총 시도 횟수 = 질문 + 오답)
         */
        fun computeCategoryBestScore(
            current: CategoryStat,
            questionCount: Int,
            wrongGuessCount: Int,
            result: GameResult,
            target: String?,
        ): CategoryBestScoreUpdate? {
            if (result != GameResult.CORRECT || target == null) return null

            val totalAttempts = questionCount + wrongGuessCount
            val currentTotal = (current.bestQuestionCount ?: Int.MAX_VALUE) + (current.bestWrongGuessCount ?: 0)

            // 총 시도 횟수가 더 적을 때만 갱신
            if (totalAttempts >= currentTotal) return null

            return CategoryBestScoreUpdate(questionCount, wrongGuessCount, target, Instant.now())
        }
    }

    // 베스트 스코어 업데이트 결과
    data class BestScoreUpdate(
        val questionCount: Int,
        val wrongGuessCount: Int,
        val target: String,
        val category: String,
    )

    // 카테고리별 베스트 스코어 업데이트 결과
    data class CategoryBestScoreUpdate(
        val questionCount: Int,
        val wrongGuessCount: Int,
        val target: String,
        val achievedAt: Instant,
    )
}
