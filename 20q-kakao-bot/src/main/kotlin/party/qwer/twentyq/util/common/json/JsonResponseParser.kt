package party.qwer.twentyq.util.common.json

import com.networknt.schema.InputFormat
import com.networknt.schema.SchemaRegistry
import com.networknt.schema.dialect.Dialects
import org.slf4j.LoggerFactory
import tools.jackson.core.JacksonException
import tools.jackson.databind.JsonNode
import tools.jackson.databind.ObjectMapper

object JsonResponseParser {
    @PublishedApi
    internal val log = LoggerFactory.getLogger(JsonResponseParser::class.java)

    fun parseAndValidate(
        response: String,
        schema: Map<String, Any>,
        objectMapper: ObjectMapper,
    ): Result<JsonNode> =
        try {
            val cleaned = MarkdownJsonExtractor.extractJsonFromMarkdown(response)
            val jsonNode = objectMapper.readTree(cleaned)

            val schemaRegistry = SchemaRegistry.withDialect(Dialects.getDraft202012())
            val schemaJson = objectMapper.writeValueAsString(schema)
            val jsonSchema = schemaRegistry.getSchema(schemaJson, InputFormat.JSON)
            val inputJson = objectMapper.writeValueAsString(jsonNode)
            val errors = jsonSchema.validate(inputJson, InputFormat.JSON)

            if (errors.isEmpty()) {
                Result.success(jsonNode)
            } else {
                val errorMsg = errors.firstOrNull()?.message ?: "Unknown validation error"
                log.warn("JSON schema validation failed: {}", errorMsg)
                Result.failure(IllegalArgumentException("Schema validation failed: $errorMsg"))
            }
        } catch (e: JacksonException) {
            log.warn("JSON parsing/validation failed: {}", e.message)
            Result.failure(e)
        }

    inline fun <reified T> parseAndValidateToType(
        response: String,
        schema: Map<String, Any>,
        objectMapper: ObjectMapper,
    ): Result<T> =
        parseAndValidate(response, schema, objectMapper).mapCatching { jsonNode ->
            objectMapper.treeToValue(jsonNode, T::class.java)
        }

    fun <T> parseJsonField(
        response: String,
        fieldName: String,
        extractor: (JsonNode) -> T?,
        fallback: T?,
        objectMapper: ObjectMapper,
    ): T? =
        try {
            val cleaned = MarkdownJsonExtractor.extractJsonFromMarkdown(response)
            val jsonNode = objectMapper.readTree(cleaned)
            runCatching { extractor(jsonNode) }.getOrNull() ?: fallback
        } catch (e: JacksonException) {
            log.warn("Failed to parse JSON field '{}': {}", fieldName, e.message)
            fallback
        }
}
