package io.github.kapu.turtlesoup.bridge

import io.github.kapu.turtlesoup.models.Command
import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.shouldBe
import io.kotest.matchers.types.shouldBeInstanceOf

class CommandParserTest : StringSpec({

    val parser = CommandParser()

    // 시작 명령어 테스트
    "should parse start command without difficulty" {
        val result = parser.parse("/스프 시작")

        result.shouldBeInstanceOf<Command.Start>()
        (result as Command.Start).difficulty shouldBe null
    }

    "should parse start command with difficulty 1" {
        val result = parser.parse("/스프 시작 1")

        result.shouldBeInstanceOf<Command.Start>()
        (result as Command.Start).difficulty shouldBe 1
    }

    "should parse start command with difficulty 3" {
        val result = parser.parse("/스프 시작 3")

        result.shouldBeInstanceOf<Command.Start>()
        (result as Command.Start).difficulty shouldBe 3
    }

    "should parse start command with difficulty 5" {
        val result = parser.parse("/스프 시작 5")

        result.shouldBeInstanceOf<Command.Start>()
        (result as Command.Start).difficulty shouldBe 5
    }

    "should parse out-of-range difficulty (0) as Start with difficulty 0" {
        // 범위 검증은 GameCommandHandler에서 수행
        val result = parser.parse("/스프 시작 0")

        result.shouldBeInstanceOf<Command.Start>().difficulty shouldBe 0
    }

    "should parse out-of-range difficulty (6) as Start with difficulty 6" {
        // 범위 검증은 GameCommandHandler에서 수행
        val result = parser.parse("/스프 시작 6")

        result.shouldBeInstanceOf<Command.Start>().difficulty shouldBe 6
    }

    "should parse large difficulty (99) as Start with difficulty 99" {
        val result = parser.parse("/스프 시작 99")

        result.shouldBeInstanceOf<Command.Start>().apply {
            difficulty shouldBe 99
            hasInvalidInput shouldBe false
        }
    }

    "should parse text difficulty as Start with hasInvalidInput true" {
        val result = parser.parse("/스프 시작 쉬움")

        result.shouldBeInstanceOf<Command.Start>().apply {
            difficulty shouldBe null
            hasInvalidInput shouldBe true
        }
    }

    "should parse english text difficulty as Start with hasInvalidInput true" {
        val result = parser.parse("/스프 시작 easy")

        result.shouldBeInstanceOf<Command.Start>().apply {
            difficulty shouldBe null
            hasInvalidInput shouldBe true
        }
    }

    // 기존 명령어 테스트
    "should parse help command" {
        val result = parser.parse("/스프")

        result shouldBe Command.Help
    }

    "should parse help command with keyword" {
        val result = parser.parse("/스프 도움")

        result shouldBe Command.Help
    }

    "should parse hint command" {
        val result = parser.parse("/스프 힌트")

        result shouldBe Command.Hint
    }

    "should parse problem command" {
        val result = parser.parse("/스프 문제")

        result shouldBe Command.Problem
    }

    "should parse surrender command" {
        val result = parser.parse("/스프 포기")

        result shouldBe Command.Surrender
    }

    "should parse agree command" {
        val result = parser.parse("/스프 동의")

        result shouldBe Command.Agree
    }

    "should parse summary command" {
        val result = parser.parse("/스프 정리")

        result shouldBe Command.Summary
    }

    "should parse ask command" {
        val result = parser.parse("/스프 이것은 질문입니다")

        result.shouldBeInstanceOf<Command.Ask>().question shouldBe "이것은 질문입니다"
    }

    "should parse answer command" {
        val result = parser.parse("/스프 정답 이것이 답입니다")

        result.shouldBeInstanceOf<Command.Answer>().answer shouldBe "이것이 답입니다"
    }

    "should return null for non-command message" {
        val result = parser.parse("일반 메시지입니다")

        result shouldBe null
    }

    "should return null for empty message" {
        val result = parser.parse("")

        result shouldBe null
    }

    "should return null for null message" {
        val result = parser.parse(null)

        result shouldBe null
    }
})
