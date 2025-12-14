package io.github.kapu.turtlesoup.utils

import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.shouldBe

class MessageProviderTest : StringSpec({
    "중첩 키를 파싱하고 변수 치환을 수행한다" {
        val yaml =
            """
            root:
              child: "value {name}"
            plain: "hello"
            """.trimIndent()

        val provider = MessageProvider(yaml)

        provider.get("root.child", "name" to "world") shouldBe "value world"
        provider.get("plain") shouldBe "hello"
    }

    "없는 키는 키 문자열을 반환한다" {
        val yaml = """root: "value""""
        val provider = MessageProvider(yaml)

        provider.get("missing") shouldBe "missing"
    }
})
