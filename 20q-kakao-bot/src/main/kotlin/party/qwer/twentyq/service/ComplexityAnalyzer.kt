package party.qwer.twentyq.service

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Service
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.mcp.NlpAnalysis
import party.qwer.twentyq.rest.NlpRestClient

/** 텍스트 복잡도 분석 서비스 */
@Service
class ComplexityAnalyzer(
    private val nlpRestClient: NlpRestClient,
) {
    companion object {
        private val log = LoggerFactory.getLogger(ComplexityAnalyzer::class.java)
        private const val MIN_TEXT_LENGTH = 3
        private const val MIN_TEXT_LENGTH_FOR_TOKEN_CHECK = 5
        private const val MIN_TOKENS_FOR_CONTENT_CHECK = 3
        private const val UNKNOWN_RATIO_HIGH = 0.4
        private const val UNKNOWN_RATIO_MEDIUM = 0.2
        private const val AVG_TOKEN_LENGTH_VERY_LOW = 0.8
        private const val AVG_TOKEN_LENGTH_LOW = 1.0
        private const val HANGUL_RATIO_VERY_LOW = 0.2
        private const val HANGUL_RATIO_LOW = 0.4
        private const val CONTENT_RATIO_VERY_LOW = 0.1
        private const val COMPLEXITY_THRESHOLD = 4
        private const val SCORE_HIGH_UNKNOWN = 3
        private const val SCORE_MEDIUM_UNKNOWN = 1
        private const val SCORE_VERY_LOW_TOKEN_LEN = 3
        private const val SCORE_LOW_TOKEN_LEN = 1
        private const val SCORE_VERY_LOW_HANGUL = 3
        private const val SCORE_LOW_HANGUL = 2
        private const val SCORE_SINGLE_JAMO = 2
        private const val SCORE_JAMO_WITH_SPECIAL = 2
        private const val SCORE_VERY_LOW_CONTENT = 2

        // 사전 컴파일된 Regex 패턴 (성능 최적화)
        private val INCOMPLETE_JAMO_PATTERN = Regex("[ㄱ-ㅎㅏ-ㅣ]{2,}")
        private val LAUGHTER_PATTERN = Regex(".*[ㅋㅎ]{2,}.*")
        private val SINGLE_JAMO_PATTERN = Regex("[ㄱ-ㅎㅏ-ㅣ]")
        private val ONLY_LAUGHTER_PATTERN = Regex("^[ㅋㅎ]+$")
        private val LAUGHTER_WITH_SPACES_PATTERN = Regex(".*\\s[ㅋㅎ]+\\s.*")
        private val SPECIAL_CHARS_PATTERN = Regex("[!?.,;:]+")
    }

    suspend fun hasComplexTypo(text: String): Boolean {
        if (text.length < MIN_TEXT_LENGTH) {
            return false
        }

        val analysis = analyzeTokens(text)
        val tokenTags = analysis?.tokens?.zip(analysis.posTag).orEmpty()
        val result =
            when {
                analysis == null -> true
                tokenTags.isEmpty() -> {
                    log.debugL { "hasComplexTypo: EMPTY tokens for text='$text'" }
                    true
                }
                else -> evaluateComplexity(text, tokenTags)
            }

        return result
    }

    private suspend fun analyzeTokens(text: String): NlpAnalysis? =
        kotlin
            .runCatching { nlpRestClient.analyzeNlp(text) }
            .onFailure { e -> log.warn("hasComplexTypo: NLP analysis failed for text='{}': {}", text, e.message) }
            .getOrNull()

    private fun evaluateComplexity(
        text: String,
        tokenTags: List<Pair<String, String>>,
    ): Boolean {
        val stats = buildComplexityStats(text, tokenTags)
        val acc = ComplexityAccumulator()

        applyUnknownScore(stats, acc)
        applyTokenLengthScore(stats, acc)
        applyJamoScores(stats, acc)
        applyHangulScore(stats, acc)
        applyContentScore(stats, acc)

        val isComplex = acc.score >= COMPLEXITY_THRESHOLD
        logComplexityResult(text, isComplex, acc)
        return isComplex
    }

    private fun buildComplexityStats(
        text: String,
        tokenTags: List<Pair<String, String>>,
    ): ComplexityStats {
        val unknownCount = tokenTags.count { (_, tag) -> tag == "UN" || tag.startsWith("UNK") }
        val unknownRatio = unknownCount.toDouble() / tokenTags.size
        val avgTokenLength = tokenTags.map { (token, _) -> token.length }.average()
        val hasIncompleteHangul =
            text.contains(INCOMPLETE_JAMO_PATTERN) &&
                !text.matches(LAUGHTER_PATTERN)
        val hasSingleJamo =
            text.contains(SINGLE_JAMO_PATTERN) &&
                !text.matches(ONLY_LAUGHTER_PATTERN) &&
                !text.matches(LAUGHTER_WITH_SPACES_PATTERN)
        val hasJamoWithSpecialChars =
            text.contains(SINGLE_JAMO_PATTERN) &&
                text.contains(SPECIAL_CHARS_PATTERN)
        val hangulCount = text.count { it in '가'..'힣' }
        val hangulRatio = hangulCount.toDouble() / text.length
        val contentWords =
            tokenTags.count { (_, tag) ->
                tag.startsWith("NN") ||
                    tag.startsWith("VV") ||
                    tag.startsWith("VA") ||
                    tag == "NR"
            }
        val contentRatio = contentWords.toDouble() / tokenTags.size

        return ComplexityStats(
            unknownRatio = unknownRatio,
            avgTokenLength = avgTokenLength,
            hasIncompleteHangul = hasIncompleteHangul,
            hasSingleJamo = hasSingleJamo,
            hasJamoWithSpecialChars = hasJamoWithSpecialChars,
            hangulRatio = hangulRatio,
            contentRatio = contentRatio,
            tokenCount = tokenTags.size,
            textLength = text.length,
        )
    }

    private fun applyUnknownScore(
        stats: ComplexityStats,
        acc: ComplexityAccumulator,
    ) {
        when {
            stats.unknownRatio > UNKNOWN_RATIO_HIGH ->
                acc.add(SCORE_HIGH_UNKNOWN, "high_unknown=${stats.unknownRatio}")
            stats.unknownRatio > UNKNOWN_RATIO_MEDIUM ->
                acc.add(SCORE_MEDIUM_UNKNOWN, "medium_unknown=${stats.unknownRatio}")
        }
    }

    private fun applyTokenLengthScore(
        stats: ComplexityStats,
        acc: ComplexityAccumulator,
    ) {
        if (stats.textLength > MIN_TEXT_LENGTH_FOR_TOKEN_CHECK) {
            when {
                stats.avgTokenLength < AVG_TOKEN_LENGTH_VERY_LOW ->
                    acc.add(SCORE_VERY_LOW_TOKEN_LEN, "very_low_avg_len=${stats.avgTokenLength}")
                stats.avgTokenLength < AVG_TOKEN_LENGTH_LOW ->
                    acc.add(SCORE_LOW_TOKEN_LEN, "low_avg_len=${stats.avgTokenLength}")
            }
        }
    }

    private fun applyJamoScores(
        stats: ComplexityStats,
        acc: ComplexityAccumulator,
    ) {
        if (stats.hasSingleJamo && stats.textLength > MIN_TEXT_LENGTH) {
            acc.add(SCORE_SINGLE_JAMO, "single_jamo")
        }
        if (stats.hasJamoWithSpecialChars) {
            acc.add(SCORE_JAMO_WITH_SPECIAL, "jamo_with_special_chars")
        }
    }

    private fun applyHangulScore(
        stats: ComplexityStats,
        acc: ComplexityAccumulator,
    ) {
        if (!stats.hasIncompleteHangul) return

        when {
            stats.hangulRatio < HANGUL_RATIO_VERY_LOW ->
                acc.add(SCORE_VERY_LOW_HANGUL, "very_low_hangul=${stats.hangulRatio}")
            stats.hangulRatio < HANGUL_RATIO_LOW ->
                acc.add(SCORE_LOW_HANGUL, "low_hangul=${stats.hangulRatio}")
        }
    }

    private fun applyContentScore(
        stats: ComplexityStats,
        acc: ComplexityAccumulator,
    ) {
        if (stats.tokenCount > MIN_TOKENS_FOR_CONTENT_CHECK && stats.contentRatio < CONTENT_RATIO_VERY_LOW) {
            acc.add(SCORE_VERY_LOW_CONTENT, "very_low_content=${stats.contentRatio}")
        }
    }

    private fun logComplexityResult(
        text: String,
        isComplex: Boolean,
        acc: ComplexityAccumulator,
    ) {
        if (isComplex) {
            log.info(
                "hasComplexTypo: COMPLEX score={} reasons={} text='{}'",
                acc.score,
                acc.reasons.joinToString(", "),
                text,
            )
        } else {
            log.debugL {
                "hasComplexTypo: NORMAL " +
                    "score=${acc.score} reasons=${acc.reasons.joinToString(", ")} text='$text'"
            }
        }
    }
}
