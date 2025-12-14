package io.github.kapu.turtlesoup.redis

import java.nio.charset.StandardCharsets

/** Lua 스크립트 로더 */
internal object LuaScripts {
    fun load(name: String): String =
        LuaScripts::class.java
            .getResource("/lua/$name")
            ?.readText(StandardCharsets.UTF_8)
            ?: error("Lua script not found: $name")
}
