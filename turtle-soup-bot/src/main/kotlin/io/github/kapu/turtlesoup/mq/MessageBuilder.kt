package io.github.kapu.turtlesoup.mq

import io.github.kapu.turtlesoup.config.GameConstants
import io.github.kapu.turtlesoup.models.GameState
import io.github.kapu.turtlesoup.models.HistoryEntry
import io.github.kapu.turtlesoup.utils.MessageKeys
import io.github.kapu.turtlesoup.utils.MessageProvider

/** 메시지 빌드 헬퍼 */
class MessageBuilder(
    private val messageProvider: MessageProvider,
) {
    /** 상태 헤더 빌드 */
    fun buildStatusHeader(state: GameState): String =
        messageProvider.get(
            MessageKeys.ANSWER_HISTORY_HEADER,
            "questionCount" to state.questionCount.toString(),
            "hintCount" to state.hintsUsed.toString(),
            "maxHints" to GameConstants.MAX_HINTS.toString(),
        )

    fun buildAnswerWithHistory(
        state: GameState,
        history: List<HistoryEntry>,
    ): String {
        val header = buildStatusHeader(state)
        val historyLines =
            history
                .mapIndexed { index, item ->
                    messageProvider.get(
                        MessageKeys.ANSWER_HISTORY_ITEM,
                        "number" to (index + 1).toString(),
                        "question" to item.question,
                        "answer" to item.answer,
                    )
                }.joinToString("\n")

        return "$header\n$historyLines"
    }

    fun buildSummary(history: List<HistoryEntry>): String {
        if (history.isEmpty()) {
            return messageProvider.get(MessageKeys.SUMMARY_EMPTY)
        }

        val header = messageProvider.get(MessageKeys.SUMMARY_HEADER, "count" to history.size.toString())
        val body =
            history.mapIndexed { index, item ->
                messageProvider.get(
                    MessageKeys.SUMMARY_ITEM,
                    "number" to (index + 1).toString(),
                    "question" to item.question,
                    "answer" to item.answer,
                )
            }.joinToString("\n")

        return "$header\n$body"
    }

    /** 힌트 블록 빌드 */
    fun buildHintBlock(hints: List<String>): String =
        if (hints.isNotEmpty()) {
            val items =
                hints.mapIndexed { index, hint ->
                    messageProvider.get(
                        MessageKeys.HINT_ITEM,
                        "number" to (index + 1).toString(),
                        "content" to hint,
                    )
                }.joinToString("\n")

            messageProvider.get(
                MessageKeys.HINT_SECTION_USED,
                "hintCount" to hints.size.toString(),
                "hintList" to items,
            )
        } else {
            messageProvider.get(MessageKeys.HINT_SECTION_NONE)
        }
}
