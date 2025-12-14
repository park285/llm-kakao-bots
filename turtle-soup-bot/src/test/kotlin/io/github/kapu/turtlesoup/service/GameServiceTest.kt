package io.github.kapu.turtlesoup.service

import io.github.kapu.turtlesoup.models.GameState
import io.github.kapu.turtlesoup.models.HistoryEntry
import io.github.kapu.turtlesoup.models.Puzzle
import io.github.kapu.turtlesoup.rest.AnswerQuestionResult
import io.github.kapu.turtlesoup.rest.LlmRestClient
import io.github.kapu.turtlesoup.rest.QuestionHistoryItem
import io.github.kapu.turtlesoup.security.McpInjectionGuard
import io.github.kapu.turtlesoup.utils.SessionNotFoundException
import io.kotest.assertions.throwables.shouldThrow
import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.collections.shouldContainExactly
import io.kotest.matchers.shouldBe
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import kotlinx.coroutines.runBlocking

class GameServiceTest : StringSpec({
    "askQuestion uses LLM-provided history and question count" {
        val restClient = mockk<LlmRestClient>()
        val sessionManager = mockk<GameSessionManager>()
        val setupService = mockk<GameSetupService>()
        val injectionGuard = mockk<McpInjectionGuard>()
        val service = GameService(restClient, sessionManager, setupService, injectionGuard)

        val puzzle =
            Puzzle(
                title = "title",
                scenario = "scenario",
                solution = "solution",
            )
        val existingState =
            GameState(
                sessionId = "chat",
                userId = "u1",
                chatId = "chat",
                puzzle = puzzle,
                history = listOf(HistoryEntry("oldQ", "oldA")),
                questionCount = 1,
            )

        val llmHistory =
            listOf(
                QuestionHistoryItem("q1", "a1"),
                QuestionHistoryItem("q2", "a2"),
            )
        val llmResult =
            AnswerQuestionResult(
                answer = "yes",
                questionCount = 5,
                history = llmHistory,
            )

        coEvery { injectionGuard.validateOrThrow(any()) } answers { firstArg() }
        coEvery { sessionManager.loadOrThrow("chat") } returns existingState
        coEvery {
            restClient.answerQuestion(
                "chat",
                puzzle.scenario,
                puzzle.solution,
                "new question",
            )
        } returns llmResult
        coEvery { sessionManager.save(any()) } just runs
        coEvery { sessionManager.refresh("chat") } just runs
        coEvery { sessionManager.withOwnerLock<Pair<GameState, AnswerQuestionResult>>("chat", any()) } coAnswers {
            val block = args[1] as suspend () -> Pair<GameState, AnswerQuestionResult>
            block.invoke()
        }

        val (newState, result) =
            runBlocking {
                service.askQuestion("chat", "new question")
            }

        result shouldBe llmResult
        newState.questionCount shouldBe 5
        newState.history.map { it.question } shouldContainExactly listOf("q1", "q2")
        newState.history.map { it.answer } shouldContainExactly listOf("a1", "a2")

        // 기존 히스토리가 그대로 누적되지 않는지 확인
        newState.history.shouldContainExactly(
            HistoryEntry("q1", "a1"),
            HistoryEntry("q2", "a2"),
        )

        coVerify(exactly = 1) { sessionManager.save(any()) }
    }

    "askQuestion fails if session missing" {
        val restClient = mockk<LlmRestClient>()
        val sessionManager = mockk<GameSessionManager>()
        val setupService = mockk<GameSetupService>()
        val injectionGuard = mockk<McpInjectionGuard>()
        val service = GameService(restClient, sessionManager, setupService, injectionGuard)

        coEvery { injectionGuard.validateOrThrow(any()) } answers { firstArg() }
        coEvery { sessionManager.loadOrThrow("missing") } throws SessionNotFoundException("missing")
        coEvery { sessionManager.withOwnerLock<Any?>("missing", any()) } coAnswers {
            val block = args[1] as suspend () -> Any?
            block.invoke()
        }

        shouldThrow<SessionNotFoundException> {
            runBlocking {
                service.askQuestion("missing", "이건 질문인가요?")
            }
        }
    }
})
