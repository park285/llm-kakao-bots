package io.github.kapu.turtlesoup.models

import kotlinx.serialization.Serializable

@Serializable
enum class PuzzleCategory {
    MYSTERY,
    HORROR,
    LOGIC,
    ABSURD,
    DAILY,
}
