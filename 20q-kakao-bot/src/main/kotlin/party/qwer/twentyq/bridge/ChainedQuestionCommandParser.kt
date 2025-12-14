package party.qwer.twentyq.bridge

import org.slf4j.LoggerFactory
import party.qwer.twentyq.logging.debugL
import party.qwer.twentyq.model.ChainCondition
import party.qwer.twentyq.model.Command
import party.qwer.twentyq.rest.NlpRestClient

internal class ChainedQuestionCommandParser(
    private val nlpRestClient: NlpRestClient,
) {
    companion object {
        private val log = LoggerFactory.getLogger(ChainedQuestionCommandParser::class.java)
        private val CONDITIONAL_EC_FORMS = setOf("면", "으면")
    }

    suspend fun parse(
        t: String,
        p: String,
    ): Command.ChainedQuestion? =
        extractConditionAndBody(t, p)?.let { (condition, body) ->
            parseQuestionBody(body, t)?.let { questions ->
                val normalized = normalizeQuestions(questions)
                log.debugL {
                    "CHAIN_PARSE_SUCCESS count=${normalized.size}, " +
                        "questions=$normalized, condition=$condition"
                }
                Command.ChainedQuestion(normalized, condition)
            }
        }

    private fun parseQuestionBody(
        body: String?,
        input: String,
    ): List<String>? {
        if (body == null) {
            log.debugL { "CHAIN_PARSE_NO_BODY input='$input'" }
            return null
        }
        log.debugL { "CHAIN_PARSE_BODY body='$body'" }

        val rawQuestions = body.split(",").map { it.trim() }.filter { it.isNotBlank() }
        if (rawQuestions.isEmpty()) {
            log.debugL { "CHAIN_PARSE_EMPTY_QUESTIONS body='$body'" }
            return null
        }
        log.debugL { "CHAIN_PARSE_RAW_QUESTIONS count=${rawQuestions.size}, questions=$rawQuestions" }
        return rawQuestions
    }

    private fun extractConditionAndBody(
        t: String,
        p: String,
    ): Pair<ChainCondition, String?>? {
        val conditionalMatch = Regex("^$p\\s+if\\s+(.+,.+)$", RegexOption.IGNORE_CASE).find(t)
        return if (conditionalMatch != null) {
            val bodyContent = conditionalMatch.groups[1]?.value?.trim()
            log.debugL { "CHAIN_PARSE_CONDITIONAL condition=IF_TRUE" }
            ChainCondition.IF_TRUE to bodyContent
        } else {
            val regularMatch = Regex("^$p\\s+(.+,.+)$").find(t)
            if (regularMatch == null) {
                log.debugL { "CHAIN_PARSE_NO_MATCH input='$t'" }
                return null
            }
            ChainCondition.ALWAYS to regularMatch.groups[1]?.value?.trim()
        }
    }

    private suspend fun normalizeQuestions(rawQuestions: List<String>): List<String> =
        rawQuestions.mapIndexed { index, q ->
            if (index < rawQuestions.size - 1) {
                val normalized = normalizeWithNlp(q)
                log.debugL { "CHAIN_NORMALIZE index=$index, original='$q', normalized='$normalized'" }
                normalized
            } else {
                log.debugL { "CHAIN_LAST_QUESTION index=$index, question='$q'" }
                q
            }
        }

    private suspend fun normalizeWithNlp(question: String): String {
        val analysis = nlpRestClient.analyzeNlp(question)
        val tokenTags = analysis.tokens.zip(analysis.posTag)
        log.debugL { "NLP_ANALYZE question='$question', tokenTags=$tokenTags" }

        // 연결어미 "면", "으면" 감지
        val hasEC = tokenTags.any { (token, tag) -> tag == "EC" && token in CONDITIONAL_EC_FORMS }

        if (hasEC) {
            // 연결어미 제거 + "?" 추가
            val stem =
                tokenTags
                    .filterNot { (token, tag) -> tag == "EC" && token in CONDITIONAL_EC_FORMS }
                    .joinToString("") { (token) -> token }
            val result = "$stem?"
            log.debugL { "NLP_NORMALIZED question='$question', hasEC=true, result='$result'" }
            return result
        }

        log.debugL { "NLP_NO_CHANGE question='$question', hasEC=false" }
        return question
    }
}
