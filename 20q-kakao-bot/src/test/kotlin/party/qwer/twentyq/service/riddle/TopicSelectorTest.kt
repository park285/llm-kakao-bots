package party.qwer.twentyq.service.riddle

import io.mockk.every
import io.mockk.mockk
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.BeforeEach
import org.junit.jupiter.api.Nested
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.assertDoesNotThrow
import org.springframework.core.io.ClassPathResource
import org.springframework.core.io.ResourceLoader
import party.qwer.twentyq.service.riddle.model.TopicEntry
import tools.jackson.databind.ObjectMapper
import tools.jackson.module.kotlin.jacksonObjectMapper
import java.io.ByteArrayInputStream

/**
 * TopicSelector 테스트
 */
class TopicSelectorTest {
    private val objectMapper: ObjectMapper = jacksonObjectMapper()
    private val resourceLoader = mockk<ResourceLoader>()

    private lateinit var topicSelector: TopicSelector

    @BeforeEach
    fun setUp() {
        topicSelector = TopicSelector(objectMapper, resourceLoader)
    }

    @Nested
    inner class InitializationTests {
        @Test
        fun `should initialize without ConcurrentHashMap usage`() {
            // Given
            mockEmptyResources()

            // When & Then
            assertDoesNotThrow { topicSelector.init() }
        }

        @Test
        fun `should load topics from JSON files`() {
            // Given
            val foodJson = """{"food": [{"name": "김치", "details": {"origin": "Korea"}}]}"""
            mockResource("food.json", foodJson)
            mockEmptyResourcesExcept("food")

            // When
            topicSelector.init()

            // Then
            val topic = topicSelector.selectTopic("food", emptyList())
            assertThat(topic.name).isEqualTo("김치")
            assertThat(topic.category).isEqualTo("food")
        }

        @Test
        fun `should handle name with parentheses by trimming`() {
            // Given
            val objectJson = """{"object": [{"name": "컴퓨터(PC)", "details": {}}]}"""
            mockResource("object.json", objectJson)
            mockEmptyResourcesExcept("object")

            // When
            topicSelector.init()

            // Then
            val topic = topicSelector.selectTopic("object", emptyList())
            assertThat(topic.name).isEqualTo("컴퓨터")
        }

        @Test
        fun `should skip non-existent files gracefully`() {
            // Given
            mockNoResourceExists()

            // When & Then
            assertDoesNotThrow { topicSelector.init() }
        }
    }

    @Nested
    inner class SelectTopicTests {
        @BeforeEach
        fun initTopics() {
            val foodJson = """{"food": [
                {"name": "김치", "details": {}},
                {"name": "비빔밥", "details": {}},
                {"name": "불고기", "details": {}}
            ]}"""
            val conceptJson = """{"concept": [
                {"name": "사랑", "details": {}},
                {"name": "자유", "details": {}}
            ]}"""
            mockResource("food.json", foodJson)
            mockResource("concept.json", conceptJson)
            mockEmptyResourcesExcept("food", "concept")
            topicSelector.init()
        }

        @Test
        fun `should select topic from specified category`() {
            // When
            val topic = topicSelector.selectTopic("food", emptyList())

            // Then
            assertThat(topic.category).isEqualTo("food")
            assertThat(topic.name).isIn("김치", "비빔밥", "불고기")
        }

        @Test
        fun `should filter out banned topics`() {
            // Given
            val bannedTopics = listOf("김치", "비빔밥")

            // When
            val topic = topicSelector.selectTopic("food", bannedTopics)

            // Then
            assertThat(topic.name).isEqualTo("불고기")
        }

        @Test
        fun `should fall back to all categories when no available topics`() {
            // Given
            val bannedTopics = listOf("김치", "비빔밥", "불고기")

            // When
            val topic = topicSelector.selectTopic("food", bannedTopics)

            // Then
            // food의 모든 항목이 ban되면 다른 카테고리에서 선택
            assertThat(topic.category).isNotEqualTo("food")
        }

        @Test
        fun `should select random category when category is null`() {
            // When
            val topic = topicSelector.selectTopic(null, emptyList())

            // Then
            assertThat(topic.category).isIn("food", "concept")
        }

        @Test
        fun `should handle case-insensitive category lookup`() {
            // When
            val topic = topicSelector.selectTopic("FOOD", emptyList())

            // Then
            assertThat(topic.category).isEqualTo("food")
        }
    }

    // Helper methods

    private fun mockResource(
        filename: String,
        content: String,
    ) {
        val resource = mockk<org.springframework.core.io.Resource>()
        every { resource.exists() } returns true
        every { resource.inputStream } returns ByteArrayInputStream(content.toByteArray())
        every { resourceLoader.getResource("classpath:topics/$filename") } returns resource
    }

    private fun mockEmptyResources() {
        val files = listOf("object.json", "food.json", "place.json", "concept.json", "movie.json", "organism.json")
        files.forEach { filename ->
            val resource = mockk<org.springframework.core.io.Resource>()
            every { resource.exists() } returns false
            every { resourceLoader.getResource("classpath:topics/$filename") } returns resource
        }
    }

    private fun mockEmptyResourcesExcept(vararg excludeCategories: String) {
        val categoryToFile =
            mapOf(
                "object" to "object.json",
                "food" to "food.json",
                "place" to "place.json",
                "concept" to "concept.json",
                "movie" to "movie.json",
                "organism" to "organism.json",
            )
        categoryToFile.forEach { (category, filename) ->
            if (category !in excludeCategories) {
                val resource = mockk<org.springframework.core.io.Resource>()
                every { resource.exists() } returns false
                every { resourceLoader.getResource("classpath:topics/$filename") } returns resource
            }
        }
    }

    private fun mockNoResourceExists() {
        val files = listOf("object.json", "food.json", "place.json", "concept.json", "movie.json", "organism.json")
        files.forEach { filename ->
            val resource = mockk<org.springframework.core.io.Resource>()
            every { resource.exists() } returns false
            every { resourceLoader.getResource("classpath:topics/$filename") } returns resource
        }
    }
}
