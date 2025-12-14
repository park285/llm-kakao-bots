package party.qwer.twentyq.service.riddle

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.service.NormalizeService

@Component
class AnswerValidator(
    private val normalizeService: NormalizeService,
) {
    companion object {
        private val log = LoggerFactory.getLogger(AnswerValidator::class.java)

        // 정답 패턴: "정답 [답변]인가요/입니까/이에요/이야"
        private val ANSWER_PATTERN = Regex("^정답\\s+(.+?)(?:인가요|입니까|이에요|이야)?\\s*$", RegexOption.IGNORE_CASE)

        // 한국어 어미 제거 패턴
        private val KOREAN_ENDINGS_PATTERN =
            Regex(
                "(?:야|이야|예요|이에요|입니까|인가요|니|죠|지|거야|거니|거죠|거지)\\s*\\??\\s*$",
                RegexOption.IGNORE_CASE,
            )

        private val WHITESPACE_PUNCT_PATTERN = Regex("[\\p{Z}\\s\\p{Punct}]")
    }

    suspend fun normalize(text: String): String =
        try {
            normalizeService.normalize(text).normalized
        } catch (e: tools.jackson.core.JacksonException) {
            log.debugL { "Normalize failed (JSON): ${e.message}" }
            text
        } catch (e: IllegalArgumentException) {
            log.debugL { "Normalize failed (arg): ${e.message}" }
            text
        } catch (e: IllegalStateException) {
            log.debugL { "Normalize failed (state): ${e.message}" }
            text
        } catch (e: NoSuchElementException) {
            log.debugL { "Normalize failed (element): ${e.message}" }
            text
        }

    fun matchExplicitAnswer(text: String): MatchResult? = ANSWER_PATTERN.find(text.trim())

    fun normalizeForEquality(text: String): String {
        var normalized =
            java.text.Normalizer
                .normalize(text, java.text.Normalizer.Form.NFKC)
                .lowercase()
        normalized = stripKoreanEndings(normalized)
        normalized = normalized.replace(WHITESPACE_PUNCT_PATTERN, "")
        return normalized
    }

    private fun stripKoreanEndings(text: String): String = text.replace(KOREAN_ENDINGS_PATTERN, "").trim()
}
