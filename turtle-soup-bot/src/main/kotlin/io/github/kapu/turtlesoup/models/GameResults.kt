package io.github.kapu.turtlesoup.models

import kotlinx.serialization.Serializable

/** 정답 제출 결과 */
@Serializable
data class AnswerResult(
    val result: ValidationResult,
    val questionCount: Int,
    val hintCount: Int,
    val maxHints: Int,
    val hintsUsed: List<String> = emptyList(),
    val explanation: String = "",
) {
    val isCorrect: Boolean get() = result == ValidationResult.YES
    val isClose: Boolean get() = result == ValidationResult.CLOSE
}

/**
 * 포기 결과
 */
@Serializable
data class SurrenderResult(
    val solution: String,
    val hintsUsed: List<String> = emptyList(),
)

/**
 * 투표 시작/진행 결과
 */
@Serializable
data class VoteResult(
    val current: Int,
    val required: Int,
    val passed: Boolean = false,
)

/**
 * 힌트 결과
 */
@Serializable
data class HintResult(
    val hintNumber: Int,
    val content: String,
)
