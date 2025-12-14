package io.github.kapu.turtlesoup.bridge

import io.github.kapu.turtlesoup.models.GameState
import io.github.kapu.turtlesoup.models.SurrenderResult
import io.github.kapu.turtlesoup.models.SurrenderVote
import io.github.kapu.turtlesoup.redis.SessionStore
import io.github.kapu.turtlesoup.redis.SurrenderVoteStore
import io.github.kapu.turtlesoup.service.GameService
import io.github.kapu.turtlesoup.service.SessionValidator
import io.github.kapu.turtlesoup.service.SurrenderVoteService
import io.github.kapu.turtlesoup.utils.MessageProvider
import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.collections.shouldContain
import io.kotest.matchers.shouldBe
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.just
import io.mockk.mockk
import io.mockk.runs
import kotlinx.coroutines.runBlocking

class SurrenderHandlerTest : StringSpec({

    "single player surrenders immediately" {
        val gameService = mockk<GameService>()
        val sessionStore = mockk<SessionStore>()
        val voteStore = mockk<SurrenderVoteStore>()
        val voteService = SurrenderVoteService(sessionStore, voteStore, SessionValidator(sessionStore))
        val messageProvider =
            MessageProvider(
                """
                surrender:
                  result: surrendered
                """.trimIndent(),
            )

        val handler = SurrenderHandler(gameService, voteService, messageProvider)

        coEvery { sessionStore.sessionExists("chat") } returns true
        coEvery {
            sessionStore.loadGameState("chat")
        } returns
            GameState(
                sessionId = "chat",
                userId = "u1",
                chatId = "chat",
            )
        coEvery { voteStore.get("chat") } returns null
        coEvery { gameService.surrender("chat") } returns SurrenderResult(solution = "sol")

        val result =
            runBlocking {
                handler.handleConsensus("chat", "u1")
            }

        result shouldBe "surrendered"
        coVerify(exactly = 1) { gameService.surrender("chat") }
        coVerify(exactly = 0) { voteStore.save(any(), any()) }
    }

    "three players start a vote requiring three approvals" {
        val gameService = mockk<GameService>()
        val sessionStore = mockk<SessionStore>()
        val voteStore = mockk<SurrenderVoteStore>()
        val voteService = SurrenderVoteService(sessionStore, voteStore, SessionValidator(sessionStore))
        val messageProvider =
            MessageProvider(
                """
                vote:
                  start: vote_start
                """.trimIndent(),
            )

        val handler = SurrenderHandler(gameService, voteService, messageProvider)

        val players = setOf("u1", "u2", "u3")
        coEvery { sessionStore.sessionExists("chat") } returns true
        coEvery {
            sessionStore.loadGameState("chat")
        } returns
            GameState(
                sessionId = "chat",
                userId = "u1",
                chatId = "chat",
                players = players,
            )
        coEvery { voteStore.get("chat") } returns null
        coEvery { voteStore.save(any(), any()) } just runs

        val result =
            runBlocking {
                handler.handleConsensus("chat", "u1")
            }

        result shouldBe "vote_start"
        coVerify(exactly = 0) { gameService.surrender(any()) }
        coVerify {
            voteStore.save(
                "chat",
                withArg { vote ->
                    vote.requiredApprovals() shouldBe 3
                    vote.approvals shouldContain "u1"
                },
            )
        }
    }

    "non player cannot agree on vote" {
        val gameService = mockk<GameService>()
        val sessionStore = mockk<SessionStore>()
        val voteStore = mockk<SurrenderVoteStore>()
        val voteService = SurrenderVoteService(sessionStore, voteStore, SessionValidator(sessionStore))
        val messageProvider =
            MessageProvider(
                """
                vote:
                  not_found: not_found
                """.trimIndent(),
            )
        val handler = SurrenderHandler(gameService, voteService, messageProvider)

        coEvery { sessionStore.sessionExists("chat") } returns true
        coEvery {
            voteStore.get("chat")
        } returns SurrenderVote(initiator = "u1", eligiblePlayers = setOf("u1"), approvals = setOf("u1"))

        val result = runBlocking { handler.handleAgree("chat", "u2") }

        result shouldBe "not_found"
        coVerify(exactly = 0) { voteStore.approve(any(), any()) }
    }

    "duplicate vote returns already voted" {
        val gameService = mockk<GameService>()
        val sessionStore = mockk<SessionStore>()
        val voteStore = mockk<SurrenderVoteStore>()
        val voteService = SurrenderVoteService(sessionStore, voteStore, SessionValidator(sessionStore))
        val messageProvider =
            MessageProvider(
                """
                vote:
                  already_voted: already_voted
                """.trimIndent(),
            )
        val handler = SurrenderHandler(gameService, voteService, messageProvider)

        coEvery { sessionStore.sessionExists("chat") } returns true
        coEvery {
            voteStore.get("chat")
        } returns SurrenderVote(initiator = "u1", eligiblePlayers = setOf("u1", "u2"), approvals = setOf("u1", "u2"))

        val result = runBlocking { handler.handleAgree("chat", "u2") }

        result shouldBe "already_voted"
        coVerify(exactly = 0) { voteStore.approve(any(), any()) }
    }

    "active vote shows in-progress message" {
        val gameService = mockk<GameService>()
        val sessionStore = mockk<SessionStore>()
        val voteStore = mockk<SurrenderVoteStore>()
        val voteService = SurrenderVoteService(sessionStore, voteStore, SessionValidator(sessionStore))
        val messageProvider =
            MessageProvider(
                """
                vote:
                  in_progress: progress current={current} required={required} remain={remain}
                """.trimIndent(),
            )
        val handler = SurrenderHandler(gameService, voteService, messageProvider)

        val players = setOf("u1", "u2", "u3")
        val vote = SurrenderVote(initiator = "u1", eligiblePlayers = players, approvals = setOf("u1"))
        coEvery { sessionStore.sessionExists("chat") } returns true
        coEvery {
            sessionStore.loadGameState("chat")
        } returns
            GameState(
                sessionId = "chat",
                userId = "u1",
                chatId = "chat",
                players = players,
            )
        coEvery { voteStore.get("chat") } returns vote

        val result = runBlocking { handler.handleConsensus("chat", "u2") }

        result shouldBe "progress current=1 required=3 remain=2"
        coVerify(exactly = 0) { gameService.surrender(any()) }
    }

    "vote completion triggers surrender and clears vote" {
        val gameService = mockk<GameService>()
        val sessionStore = mockk<SessionStore>()
        val voteStore = mockk<SurrenderVoteStore>()
        val voteService = SurrenderVoteService(sessionStore, voteStore, SessionValidator(sessionStore))
        val messageProvider =
            MessageProvider(
                """
                vote:
                  passed: passed
                surrender:
                  result: solved {solution}
                """.trimIndent(),
            )
        val handler = SurrenderHandler(gameService, voteService, messageProvider)

        val players = setOf("u1", "u2", "u3")
        val activeVote = SurrenderVote(initiator = "u1", eligiblePlayers = players, approvals = setOf("u1", "u2"))
        val completed = activeVote.approve("u3")

        coEvery { sessionStore.sessionExists("chat") } returns true
        coEvery { voteStore.get("chat") } returns activeVote
        coEvery { voteStore.approve("chat", "u3") } returns completed
        coEvery { voteStore.clear("chat") } just runs
        coEvery { gameService.surrender("chat") } returns SurrenderResult(solution = "final")

        val result = runBlocking { handler.handleAgree("chat", "u3") }

        result shouldBe "passed\n\nsolved final"
        coVerify { voteStore.clear("chat") }
        coVerify { gameService.surrender("chat") }
    }
})
