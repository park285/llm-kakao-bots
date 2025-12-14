package party.qwer.twentyq.bridge.handlers

import org.slf4j.LoggerFactory
import org.springframework.stereotype.Component
import party.qwer.twentyq.api.dto.RiddleStatusResponse
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.util.game.GameMessageProvider

@Component
class StatusHandler(
    private val riddleService: RiddleService,
    private val messageProvider: GameMessageProvider,
) {
    companion object {
        private val log = LoggerFactory.getLogger(StatusHandler::class.java)
    }

    suspend fun handle(chatId: String): String = handleSeparated(chatId).joinToString("\n")

    suspend fun handleSeparated(chatId: String): List<String> {
        val status = riddleService.getStatus(chatId)
        val remaining = (status.maxHints - status.hintCount).coerceAtLeast(0)

        val header = buildHeader(status.selectedCategory, remaining)
        val wrongGuessLine = buildWrongGuessLine(chatId)
        val hintLines = buildHintLines(status)
        val qnaLines = buildQnaLines(status)

        log.info(
            "STATUS_SEPARATED chatId={}, questionCount={}, hintCount={}, hasHints={}",
            chatId,
            status.questionCount,
            status.hintCount,
            hintLines.isNotBlank(),
        )

        val messages = mutableListOf<String>()

        // 첫 번째 메시지: 현황 (헤더 + 오답 + Q&A)
        val mainParts = mutableListOf<String>()
        mainParts.add(header)
        if (wrongGuessLine.isNotBlank()) mainParts.add(wrongGuessLine)
        if (qnaLines.isNotBlank()) mainParts.add(qnaLines)
        messages.add(mainParts.joinToString("\n"))

        // 두 번째 메시지: 힌트 (있는 경우만)
        if (hintLines.isNotBlank()) {
            messages.add(hintLines)
        }

        return messages
    }

    private fun buildHeader(
        category: String?,
        remaining: Int,
    ): String =
        category?.let {
            messageProvider.get(
                "status.header_with_category",
                "category" to it,
                "remaining" to remaining,
            )
        } ?: messageProvider.get(
            "status.header_no_category",
            "remaining" to remaining,
        )

    private suspend fun buildWrongGuessLine(chatId: String): String {
        val wrongGuesses = runCatching { riddleService.getWrongGuesses(chatId) }.getOrElse { emptyList() }
        if (wrongGuesses.isEmpty()) return ""
        return messageProvider.get(
            "status.wrong_guesses",
            "guesses" to wrongGuesses.joinToString(", "),
        )
    }

    private fun buildHintLines(status: RiddleStatusResponse): String {
        if (status.hints.isEmpty()) return ""
        val first = status.hints.first()
        return messageProvider.get(
            "status.hint_line",
            "number" to first.hintNumber,
            "content" to first.content,
        )
    }

    private fun buildQnaLines(status: RiddleStatusResponse): String {
        if (status.questions.isEmpty()) return ""
        return status.questions
            .mapIndexed { idx, h ->
                val numberText =
                    if (h.isChain) {
                        "${idx + 1}${messageProvider.get("status.chain_suffix")}"
                    } else {
                        "${idx + 1}"
                    }
                messageProvider.get(
                    "status.question_answer",
                    "number" to numberText,
                    "question" to h.question,
                    "answer" to h.answer,
                )
            }.joinToString("\n")
    }
}
