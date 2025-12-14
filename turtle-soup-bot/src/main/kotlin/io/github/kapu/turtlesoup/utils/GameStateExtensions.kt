package io.github.kapu.turtlesoup.utils

import io.github.kapu.turtlesoup.config.GameConstants
import io.github.kapu.turtlesoup.models.GameState

/** 힌트 사용 가능 여부 (최대 3개) */
fun GameState.canUseHint(): Boolean = hintsUsed < GameConstants.MAX_HINTS

/** 남은 힌트 수 */
fun GameState.remainingHints(): Int = (GameConstants.MAX_HINTS - hintsUsed).coerceAtLeast(0)
