package party.qwer.twentyq.util.common.json

import com.vladsch.flexmark.ast.FencedCodeBlock
import com.vladsch.flexmark.parser.Parser
import com.vladsch.flexmark.util.ast.Node as FlexmarkNode

object MarkdownJsonExtractor {
    fun extractJsonFromMarkdown(text: String): String {
        val trimmed = text.trim()
        if (looksLikeJson(trimmed)) return trimmed

        return runCatching {
            val parser = Parser.builder().build()
            val document = parser.parse(trimmed)
            findFirstJsonCodeBlock(document) ?: trimmed
        }.getOrElse { trimmed }
    }

    private fun looksLikeJson(s: String): Boolean = s.startsWith("{") && s.endsWith("}")

    private fun findFirstJsonCodeBlock(document: FlexmarkNode): String? {
        var node: FlexmarkNode? = document.firstChild
        while (node != null) {
            if (node is FencedCodeBlock) {
                val code = node.contentChars.toString().trim()
                if (looksLikeJson(code)) return code
            }
            node = node.next
        }
        return null
    }
}
