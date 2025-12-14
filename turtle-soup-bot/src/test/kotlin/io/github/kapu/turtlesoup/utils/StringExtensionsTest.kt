package io.github.kapu.turtlesoup.utils

import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.shouldBe

class StringExtensionsTest : StringSpec({

    "isValidQuestion should validate question length" {
        "질문".isValidQuestion() shouldBe true
        "이것은 유효한 질문입니다".isValidQuestion() shouldBe true
        "".isValidQuestion() shouldBe false
        " ".isValidQuestion() shouldBe false
        "a".isValidQuestion() shouldBe false // MIN_QUESTION_LENGTH = 2
    }
})
