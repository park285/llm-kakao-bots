package io.github.kapu.turtlesoup.models

import io.github.kapu.turtlesoup.config.PuzzleConstants
import kotlinx.serialization.Serializable
import java.time.Instant

@Serializable
data class Puzzle(
    val title: String,
    val scenario: String,
    val solution: String,
    val category: PuzzleCategory = PuzzleCategory.MYSTERY,
    val difficulty: Int = 3,
    val hints: List<String> = emptyList(),
    @Serializable(with = InstantSerializer::class)
    val createdAt: Instant = Instant.now(),
) {
    init {
        require(difficulty in PuzzleConstants.MIN_DIFFICULTY..PuzzleConstants.MAX_DIFFICULTY) {
            "Difficulty must be between ${PuzzleConstants.MIN_DIFFICULTY} and ${PuzzleConstants.MAX_DIFFICULTY}"
        }
    }
}

@Serializable
data class PuzzleGenerationRequest(
    val category: PuzzleCategory? = null,
    val difficulty: Int? = null,
    val theme: String? = null,
) {
    init {
        difficulty?.let {
            require(it in PuzzleConstants.MIN_DIFFICULTY..PuzzleConstants.MAX_DIFFICULTY) {
                "Difficulty must be between ${PuzzleConstants.MIN_DIFFICULTY} and ${PuzzleConstants.MAX_DIFFICULTY}"
            }
        }
    }
}

@Serializable
data class PuzzleGenerationResponse(
    val title: String = "",
    val scenario: String = "",
    val solution: String = "",
    val category: String = "",
    val difficulty: Int = 0,
    val hints: List<String> = emptyList(),
) {
    fun toPuzzle(): Puzzle {
        val parsedCategory =
            PuzzleCategory.entries.find {
                it.name.equals(category, ignoreCase = true)
            } ?: PuzzleCategory.MYSTERY

        return Puzzle(
            title = title,
            scenario = scenario,
            solution = solution,
            category = parsedCategory,
            difficulty = difficulty.coerceIn(PuzzleConstants.MIN_DIFFICULTY, PuzzleConstants.MAX_DIFFICULTY),
            hints = hints,
        )
    }
}

/** 정답 판정 결과 */
@Serializable
enum class ValidationResult {
    YES, // 정답
    CLOSE, // 근접 (핵심 파악했으나 설명 부족)
    NO, // 오답
}

/** 힌트 응답 (구조화된 AI 출력) */
@Serializable
data class HintResponse(
    val hint: String,
)
