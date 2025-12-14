package io.github.kapu.turtlesoup.bridge

import io.github.kapu.turtlesoup.models.Command
import io.github.oshai.kotlinlogging.KotlinLogging

/** /스프 명령어 파서 */
class CommandParser(
    private val prefix: String = "/스프",
) {
    private val escapedPrefix = Regex.escape(prefix)

    // 파서 함수 목록 - 복잡도 감소를 위해 리스트 사용
    private val parsers: List<(String) -> Command?> by lazy {
        listOf(
            ::parseHelp,
            ::parseStart,
            ::parseHint,
            ::parseProblem,
            ::parseSurrender,
            ::parseAgree,
            ::parseSummary,
            ::parseAnswer,
            ::parseAsk,
        )
    }

    /** 메시지를 Command로 파싱 */
    fun parse(message: String?): Command? {
        if (message.isNullOrBlank()) return null
        val text = message.trim()
        if (!text.startsWith(prefix)) return null

        logger.debug { "parsing message='$text'" }

        return parsers.firstNotNullOfOrNull { it(text) } ?: Command.Unknown
    }

    /** /스프 또는 /스프 도움 */
    private fun parseHelp(text: String): Command? {
        val helpRe = Regex("^$escapedPrefix\\s*(?:도움|help)?$", RegexOption.IGNORE_CASE)
        return if (helpRe.containsMatchIn(text)) Command.Help else null
    }

    /** /스프 시작 [난이도] */
    private fun parseStart(text: String): Command? {
        val startRe = Regex("""^$escapedPrefix\s*(?:시작|start)(?:\s+(\S+))?$""", RegexOption.IGNORE_CASE)
        val match = startRe.find(text) ?: return null
        val rawInput = match.groups[1]?.value
        val difficulty = rawInput?.toIntOrNull()
        val hasInvalidInput = rawInput != null && difficulty == null
        return Command.Start(difficulty, hasInvalidInput)
    }

    /** /스프 힌트 */
    private fun parseHint(text: String): Command? {
        val hintRe = Regex("^$escapedPrefix\\s*(?:힌트|hint)$", RegexOption.IGNORE_CASE)
        return if (hintRe.containsMatchIn(text)) Command.Hint else null
    }

    /** /스프 문제 */
    private fun parseProblem(text: String): Command? {
        val problemRe = Regex("^$escapedPrefix\\s*(?:문제|제시문|problem)$", RegexOption.IGNORE_CASE)
        return if (problemRe.containsMatchIn(text)) Command.Problem else null
    }

    /** /스프 포기 */
    private fun parseSurrender(text: String): Command? {
        val surrenderRe = Regex("^$escapedPrefix\\s*(?:포기|surrender)$", RegexOption.IGNORE_CASE)
        return if (surrenderRe.containsMatchIn(text)) Command.Surrender else null
    }

    /** /스프 동의 */
    private fun parseAgree(text: String): Command? {
        val agreeRe = Regex("^$escapedPrefix\\s*(?:동의|agree)$", RegexOption.IGNORE_CASE)
        return if (agreeRe.containsMatchIn(text)) Command.Agree else null
    }

    /** /스프 정리 */
    private fun parseSummary(text: String): Command? {
        val summaryRe = Regex("^$escapedPrefix\\s*(?:정리|summary)$", RegexOption.IGNORE_CASE)
        return if (summaryRe.containsMatchIn(text)) Command.Summary else null
    }

    /** /스프 정답 [답변] */
    private fun parseAnswer(text: String): Command? {
        val answerRe = Regex("^$escapedPrefix\\s*(?:정답|answer)\\s+(.+)$", RegexOption.IGNORE_CASE)
        val answer = answerRe.find(text)?.groups?.get(1)?.value?.trim()
        return answer?.let { Command.Answer(it) }
    }

    /** /스프 [질문] - 일반 질문 */
    private fun parseAsk(text: String): Command? {
        val askRe = Regex("^$escapedPrefix\\s+(.+)$")
        val question = askRe.find(text)?.groups?.get(1)?.value?.trim()?.takeIf { it.isNotBlank() }
        return question?.let { Command.Ask(it) }
    }

    companion object {
        private val logger = KotlinLogging.logger {}
    }
}
