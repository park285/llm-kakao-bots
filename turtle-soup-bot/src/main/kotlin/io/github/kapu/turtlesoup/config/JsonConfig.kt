package io.github.kapu.turtlesoup.config

import kotlinx.serialization.json.Json

/** 공유 JSON 설정 */
object JsonConfig {
    val lenient: Json =
        Json {
            ignoreUnknownKeys = true
            isLenient = true
        }

    val prettyLenient: Json =
        Json {
            ignoreUnknownKeys = true
            isLenient = true
            prettyPrint = true
        }
}
