package party.qwer.twentyq.util.game

import jakarta.annotation.PostConstruct
import org.slf4j.LoggerFactory
import org.springframework.core.io.ClassPathResource
import org.springframework.stereotype.Component
import party.qwer.twentyq.config.PromptResourcePaths
import party.qwer.twentyq.service.exception.GameMessageKeys
import party.qwer.twentyq.util.logging.LoggingConstants.DEFAULT_EASTER_EGG_PROBABILITY
import party.qwer.twentyq.util.logging.LoggingConstants.PERCENT_MULTIPLIER
import tools.jackson.core.type.TypeReference
import tools.jackson.databind.ObjectMapper
import tools.jackson.dataformat.yaml.YAMLFactory

/**
 * 게임 메시지 제공자
 */
@Component
class GameMessageProvider(
    @param:org.springframework.beans.factory.annotation.Qualifier("yamlObjectMapper")
    private val objectMapper: tools.jackson.databind.ObjectMapper,
) {
    companion object {
        private val log = LoggerFactory.getLogger(GameMessageProvider::class.java)
        private const val MESSAGE_FILE_PATH = PromptResourcePaths.GAME_MESSAGES
    }

    private lateinit var messages: Map<String, Any>

    @PostConstruct
    fun init() {
        runCatching {
            val resource = ClassPathResource(MESSAGE_FILE_PATH)
            val tree = objectMapper.readTree(resource.inputStream)
            val rootNode = tree.get("toon") ?: tree
            messages = objectMapper.convertValue(rootNode, object : TypeReference<Map<String, Any>>() {})

            log.info("GameMessageProvider initialized from classpath:{}", MESSAGE_FILE_PATH)
        }.getOrElse { ex ->
            log.error("Failed to load game messages from {}: {}", MESSAGE_FILE_PATH, ex.message, ex)
            error("Cannot load game messages")
        }
    }

    fun get(
        key: String,
        vararg params: Pair<String, Any>,
    ): String {
        val template = getNestedValue(messages, key) ?: return key
        return params.fold(template) { acc, (k, v) ->
            acc.replace("{$k}", v.toString())
        }
    }

    fun getInvalidQuestionMessage(easterEggProbability: Int = DEFAULT_EASTER_EGG_PROBABILITY): String {
        val random = kotlin.random.Random.Default
        return if (random.nextInt(PERCENT_MULTIPLIER) < easterEggProbability) {
            // GameMessageKeys.INVALID_QUESTION_EASTER_EGGS 직접 조회
            val keys = GameMessageKeys.INVALID_QUESTION_EASTER_EGGS.split(".")
            var current: Any? = messages

            for (k in keys) {
                when (current) {
                    is Map<*, *> -> current = current[k]
                    else -> {
                        current = null
                        break
                    }
                }
            }

            val easterEggs =
                when (current) {
                    is List<*> -> current.mapNotNull { it as? String }
                    else -> emptyList()
                }

            easterEggs.randomOrNull() ?: get(GameMessageKeys.INVALID_QUESTION_DEFAULT)
        } else {
            get(GameMessageKeys.INVALID_QUESTION_DEFAULT)
        }
    }

    private fun getNestedValue(
        map: Map<String, Any>,
        path: String,
    ): String? {
        val keys = path.split(".")
        var current: Any? = map

        for (key in keys) {
            when (current) {
                is Map<*, *> -> current = current[key]
                else -> return null
            }
        }

        return when (current) {
            is String -> current
            is List<*> -> current.joinToString(", ") { it.toString() }
            else -> current?.toString()
        }
    }
}
