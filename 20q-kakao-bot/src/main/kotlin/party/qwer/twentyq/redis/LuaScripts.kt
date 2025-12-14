package party.qwer.twentyq.redis

import java.nio.charset.StandardCharsets

internal object LuaScripts {
    fun load(name: String): String =
        LuaScripts::class.java
            .getResource("/lua/$name")
            ?.readText(StandardCharsets.UTF_8)
            ?: error("Lua script not found: $name")
}
