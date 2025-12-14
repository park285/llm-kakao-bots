package io.github.kapu.turtlesoup.models

import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.collections.shouldContain
import io.kotest.matchers.shouldBe
import io.kotest.matchers.shouldNotBe

class GameStateTest : StringSpec({

    "should create GameState with default values" {
        val state =
            GameState(
                sessionId = "session-1",
                userId = "user-1",
                chatId = "chat-1",
            )

        state.sessionId shouldBe "session-1"
        state.userId shouldBe "user-1"
        state.chatId shouldBe "chat-1"
        state.hintsUsed shouldBe 0
        state.isSolved shouldBe false
        state.questionCount shouldBe 0
    }

    "should use hint immutably" {
        val state =
            GameState(
                sessionId = "session-1",
                userId = "user-1",
                chatId = "chat-1",
            )

        val newState = state.useHint("Test hint content")

        state.hintsUsed shouldBe 0
        newState.hintsUsed shouldBe 1
        newState.hintContents shouldContain "Test hint content"
    }

    "should mark solved immutably" {
        val state =
            GameState(
                sessionId = "session-1",
                userId = "user-1",
                chatId = "chat-1",
                isSolved = false,
            )

        val newState = state.markSolved()

        state.isSolved shouldBe false
        newState.isSolved shouldBe true
    }

    "should calculate elapsed seconds" {
        val state =
            GameState(
                sessionId = "session-1",
                userId = "user-1",
                chatId = "chat-1",
            )

        state.elapsedSeconds shouldNotBe null
        state.elapsedSeconds shouldBe 0L
    }
})
