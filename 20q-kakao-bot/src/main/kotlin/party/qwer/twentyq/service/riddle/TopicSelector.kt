package party.qwer.twentyq.service.riddle

import jakarta.annotation.PostConstruct
import org.slf4j.LoggerFactory
import org.springframework.core.io.ResourceLoader
import org.springframework.stereotype.Service
import party.qwer.twentyq.service.riddle.model.TopicEntry
import tools.jackson.core.type.TypeReference
import tools.jackson.databind.ObjectMapper
import tools.jackson.module.kotlin.readValue
import kotlin.random.Random

/** 카테고리별 주제 선택 서비스 */
@Service
class TopicSelector(
    private val objectMapper: ObjectMapper,
    private val resourceLoader: ResourceLoader,
) {
    companion object {
        private val log = LoggerFactory.getLogger(TopicSelector::class.java)
    }

    // 초기화 후 변경되지 않으므로 불변 Map 사용
    private lateinit var topics: Map<String, List<TopicEntry>>

    // Jackson 파싱용 중간 data class (타입 안전)
    private data class TopicItemRaw(
        val name: String,
        val details: Map<String, Any>,
    )

    @PostConstruct
    fun init() {
        log.info("TopicSelector initializing from classpath:topics/")

        val categoryFiles =
            mapOf(
                "object" to "object.json",
                "food" to "food.json",
                "place" to "place.json",
                "concept" to "concept.json",
                "movie" to "movie.json",
                "organism" to "organism.json",
            )

        val loadedTopics = mutableMapOf<String, List<TopicEntry>>()
        categoryFiles.forEach { (category, filename) ->
            loadCategory(category, filename, loadedTopics)
        }

        // 불변 Map으로 변환 (컨벤션: X ConcurrentHashMap)
        topics = loadedTopics.toMap()

        val totalCount = topics.values.sumOf { it.size }
        log.info("TopicSelector initialized with {} topics across {} categories", totalCount, topics.size)
    }

    private fun loadCategory(
        category: String,
        filename: String,
        targetMap: MutableMap<String, List<TopicEntry>>,
    ) {
        try {
            val resource = resourceLoader.getResource("classpath:topics/$filename")
            if (!resource.exists()) {
                log.warn("Topic file not found: classpath:topics/{}", filename)
                return
            }

            val root: Map<String, List<TopicItemRaw>> =
                resource.inputStream.use { stream ->
                    objectMapper.readValue(stream, object : TypeReference<Map<String, List<TopicItemRaw>>>() {})
                }
            val items = root[category] ?: emptyList()

            val entries =
                items.map { item ->
                    TopicEntry(
                        name = item.name.split("(")[0].trim(),
                        details = item.details,
                        category = category,
                    )
                }

            targetMap[category] = entries
            log.info("Loaded {} topics from {}", entries.size, filename)
        } catch (ex: tools.jackson.core.JacksonException) {
            log.error("Failed to parse topics from {}: {}", filename, ex.message, ex)
        } catch (ex: java.io.IOException) {
            log.error("I/O error while loading {}: {}", filename, ex.message, ex)
        }
    }

    fun selectTopic(
        category: String?,
        bannedTopics: List<String> = emptyList(),
    ): TopicEntry {
        val selectedCategory = category?.lowercase() ?: selectRandomCategory()
        log.info(
            "selectTopic CALLED category={}, selectedCategory={}, availableCategories={}",
            category,
            selectedCategory,
            topics.keys,
        )

        val categoryTopics = topics[selectedCategory]
        log.info("selectTopic categoryTopics found={}, count={}", categoryTopics != null, categoryTopics?.size ?: 0)

        val availableTopics =
            categoryTopics
                ?.filter { topic ->
                    !bannedTopics.any { banned ->
                        topic.name.equals(banned, ignoreCase = true)
                    }
                } ?: emptyList()

        log.info("selectTopic AFTER_FILTER availableTopics={}, bannedTopics={}", availableTopics.size, bannedTopics)

        if (availableTopics.isEmpty()) {
            log.warn(
                "No available topics for category={}, bannedCount={}, falling back to all categories",
                selectedCategory,
                bannedTopics.size,
            )
            return selectFromAllCategories(bannedTopics)
        }

        val selected = availableTopics.random()
        log.info(
            "Selected topic: '{}' from category '{}', available count: {}",
            selected.name,
            selectedCategory,
            availableTopics.size,
        )
        return selected
    }

    private fun selectRandomCategory(): String {
        val categories = topics.keys.toList()
        return categories[Random.nextInt(categories.size)]
    }

    private fun selectFromAllCategories(bannedTopics: List<String>): TopicEntry {
        val allTopics =
            topics.values
                .flatten()
                .filter { topic ->
                    !bannedTopics.any { banned ->
                        topic.name.equals(banned, ignoreCase = true)
                    }
                }

        check(allTopics.isNotEmpty()) { "No topics available after filtering banned topics" }

        return allTopics.random()
    }
}
