package io.github.kapu.turtlesoup.utils

import com.charleskorn.kaml.Yaml
import com.charleskorn.kaml.YamlMap
import com.charleskorn.kaml.YamlScalar
import io.github.oshai.kotlinlogging.KotlinLogging

/** YAML 기반 메시지 템플릿 관리자 */
class MessageProvider(yamlContent: String) {
    private val messages: Map<String, Any?>

    init {
        messages = parseYaml(yamlContent)
        logger.info { "message_provider_initialized keys=${messages.keys.size}" }
    }

    /**
     * 메시지 템플릿 조회 및 변수 치환
     */
    fun get(
        key: String,
        vararg params: Pair<String, Any>,
    ): String {
        val template =
            getNestedValue(messages, key) ?: run {
                logger.warn { "message_key_not_found key=$key" }
                return key
            }

        return params.fold(template) { acc, (k, v) ->
            acc.replace("{$k}", v.toString())
        }
    }

    /**
     * 중첩된 키로 값 조회 (예: "error.no_session")
     */
    private fun getNestedValue(
        map: Map<String, Any?>,
        key: String,
    ): String? {
        val parts = key.split('.')
        var current: Any? = map

        for (part in parts) {
            current = (current as? Map<*, *>)?.get(part) ?: return null
        }

        return when (current) {
            is String -> current
            else -> current?.toString()
        }
    }

    /**
     * YamlMap을 Kotlin Map으로 변환
     */
    private fun parseYaml(content: String): Map<String, Any?> {
        val yaml = Yaml.default
        val node = yaml.parseToYamlNode(content)

        return when (node) {
            is YamlMap -> yamlMapToMap(node)
            else -> emptyMap()
        }
    }

    private fun yamlMapToMap(yamlMap: YamlMap): Map<String, Any?> {
        val result = mutableMapOf<String, Any?>()

        yamlMap.entries.forEach { (key, value) ->
            val keyStr = key.content
            val valueAny =
                when (value) {
                    is YamlScalar -> value.content
                    is YamlMap -> yamlMapToMap(value)
                    else -> value.toString()
                }
            result[keyStr] = valueAny
        }

        return result
    }

    companion object {
        private val logger = KotlinLogging.logger {}

        /** 클래스패스에서 YAML 로드 */
        fun fromClasspath(path: String): MessageProvider {
            val content =
                MessageProvider::class.java
                    .getResourceAsStream(path)
                    ?.bufferedReader()
                    ?.use { it.readText() }
                    ?: throw IllegalArgumentException("YAML file not found: $path")

            return MessageProvider(content)
        }
    }
}
