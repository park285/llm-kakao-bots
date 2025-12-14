package party.qwer.twentyq.bridge.handlers

import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.mockk
import kotlinx.coroutines.test.runTest
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.assertThrows
import party.qwer.twentyq.config.AppProperties
import party.qwer.twentyq.config.properties.Commands
import party.qwer.twentyq.model.Command
import party.qwer.twentyq.model.FiveScaleKo
import party.qwer.twentyq.mq.MessageQueueCoordinator
import party.qwer.twentyq.service.RiddleService
import party.qwer.twentyq.service.dto.AnswerResult
import party.qwer.twentyq.service.dto.AnswerSource
import party.qwer.twentyq.service.exception.GameMessageKeys
import party.qwer.twentyq.util.game.GameMessageProvider

class ChainedQuestionHandlerTest {
    private val riddleService = mockk<RiddleService>()
    private val queueCoordinator = mockk<MessageQueueCoordinator>()
    private val messageProvider = mockk<GameMessageProvider>(relaxed = true)
    private val appProperties = mockk<AppProperties>()

    private val handler =
        ChainedQuestionHandler(
            riddleService,
            queueCoordinator,
            messageProvider,
        )

    init {
        every { appProperties.commands } returns
            mockk<Commands>().apply {
                every { prefix } returns "/스자"
            }
    }

    @Test
    fun `should handle first question immediately`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("전기쓰?", "배터리도 있음?", "충전 가능?")
            val command = Command.ChainedQuestion(questions)

            coEvery { riddleService.answer(chatId, "전기쓰?", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_YES,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.enqueue(any(), any()) } returns
                party.qwer.twentyq.mq.queue.EnqueueResult.SUCCESS
            every { messageProvider.get(any(), *anyVararg()) } returns "큐에 추가됨"

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).isEqualTo("예")
            coVerify(exactly = 1) { riddleService.answer(chatId, "전기쓰?", userId) }
        }

    @Test
    fun `should throw exception for empty questions`() =
        runTest {
            // Given
            val command = Command.ChainedQuestion(emptyList())
            every { messageProvider.get("error.invalid_question") } returns "잘못된 질문입니다"

            // When & Then
            val exception =
                assertThrows<party.qwer.twentyq.service.exception.InvalidQuestionException> {
                    handler.handle("chat", command, "user", null)
                }
            assertThat(exception.message).isEqualTo("잘못된 질문입니다")
            coVerify(exactly = 0) { riddleService.answer(any(), any(), any()) }
        }

    @Test
    fun `should handle single question without enqueuing`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("단일 질문?")
            val command = Command.ChainedQuestion(questions)

            coEvery { riddleService.answer(chatId, "단일 질문?", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.MOSTLY_YES,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).isEqualTo("아마도 예")
            coVerify(exactly = 1) { riddleService.answer(chatId, "단일 질문?", userId) }
            coVerify(exactly = 0) { queueCoordinator.enqueue(any(), any()) }
        }

    @Test
    fun `should process first question only`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("Q1", "Q2", "Q3", "Q4", "Q5")
            val command = Command.ChainedQuestion(questions)

            coEvery { riddleService.answer(chatId, "Q1", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_NO,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.enqueue(any(), any()) } returns
                party.qwer.twentyq.mq.queue.EnqueueResult.SUCCESS
            every { messageProvider.get(any(), *anyVararg()) } returns "큐에 추가됨"

            // When
            handler.handle(chatId, command, userId, null)

            // Then
            coVerify(exactly = 1) { riddleService.answer(chatId, "Q1", userId) }
        }

    @Test
    fun `should return success message when answer is correct`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("정답이야?")
            val command = Command.ChainedQuestion(questions)

            coEvery { riddleService.answer(chatId, "정답이야?", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_YES,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                    isCorrect = true,
                    successMessage = "축하합니다! 정답입니다!",
                )

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).isEqualTo("축하합니다! 정답입니다!")
            coVerify(exactly = 1) { riddleService.answer(chatId, "정답이야?", userId) }
            coVerify(exactly = 0) { queueCoordinator.enqueue(any(), any()) }
        }

    @Test
    fun `should return default success message when isCorrect but successMessage is null`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("정답?")
            val command = Command.ChainedQuestion(questions)

            coEvery { riddleService.answer(chatId, "정답?", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_YES,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                    isCorrect = true,
                    successMessage = null,
                )
            every { messageProvider.get("answer.correct_default") } returns "정답입니다!"

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).isEqualTo("정답입니다!")
        }

    @Test
    fun `should return error message when guardDegraded is true`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("이상한 질문?", "두 번째 질문?")
            val command = Command.ChainedQuestion(questions)

            coEvery { riddleService.answer(chatId, "이상한 질문?", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.INVALID,
                    source = AnswerSource.FALLBACK_DEFAULT,
                    guardDegraded = true,
                )
            coEvery { queueCoordinator.enqueue(any(), any()) } returns
                party.qwer.twentyq.mq.queue.EnqueueResult.SUCCESS
            every { messageProvider.get(any(), *anyVararg()) } returns "큐에 추가됨"
            every { messageProvider.get("error.invalid_question.default") } returns "질문을 이해할 수 없습니다"

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).isEqualTo("질문을 이해할 수 없습니다")
            coVerify(exactly = 1) { riddleService.answer(chatId, "이상한 질문?", userId) }
        }

    @Test
    fun `should handle MOSTLY_NO scale correctly`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("그럴까?")
            val command = Command.ChainedQuestion(questions)

            coEvery { riddleService.answer(chatId, "그럴까?", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.MOSTLY_NO,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).isEqualTo("아마도 아니오")
        }

    @Test
    fun `should handle INVALID scale correctly`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("???")
            val command = Command.ChainedQuestion(questions)

            coEvery { riddleService.answer(chatId, "???", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.INVALID,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).isEqualTo("이해할 수 없는 질문입니다")
        }

    @Test
    fun `should handle wrong guess`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val sender = "테스터"
            val questions = listOf("정답은 사과?", "두 번째 질문?", "세 번째 질문?")
            val command = Command.ChainedQuestion(questions)

            coEvery { riddleService.answer(chatId, "정답은 사과?", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_NO,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                    isWrongGuess = true,
                    guessedAnswer = "사과",
                )
            coEvery { queueCoordinator.enqueue(any(), any()) } returns
                party.qwer.twentyq.mq.queue.EnqueueResult.SUCCESS
            every { messageProvider.get(any(), *anyVararg()) } returns "큐에 추가됨"
            every {
                messageProvider.get(
                    GameMessageKeys.ANSWER_WRONG_GUESS,
                    "nickname" to sender,
                    "guess" to "사과",
                )
            } returns "테스터님 「사과」는 정답이 아닙니다"

            // When
            val result = handler.handle(chatId, command, userId, sender)

            // Then
            assertThat(result).isEqualTo("테스터님 「사과」는 정답이 아닙니다")
            coVerify(exactly = 1) { riddleService.answer(chatId, "정답은 사과?", userId) }
        }

    // Conditional chain tests

    @Test
    fun `should queue remaining questions when IF_TRUE and first answer is YES`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val sender = "테스터"
            val questions = listOf("동물인가요", "척추동물인가요", "포유류인가요")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "동물인가요", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_YES,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.enqueue(any(), any()) } returns
                party.qwer.twentyq.mq.queue.EnqueueResult.SUCCESS
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit
            every { messageProvider.get(any(), *anyVararg()) } returns "큐에 추가됨"
            every { messageProvider.get("chain.queued", "questions" to "척추동물인가요, 포유류인가요") } returns
                "\n\n📋 다음 질문 등록됨: 척추동물인가요, 포유류인가요"

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).isEqualTo("예")
            coVerify(exactly = 0) { queueCoordinator.enqueue(any(), any()) }
        }

    @Test
    fun `should queue remaining questions when IF_TRUE and first answer is MOSTLY_YES`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val sender = "테스터"
            val questions = listOf("동물인가요", "척추동물인가요")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "동물인가요", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.MOSTLY_YES,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.enqueue(any(), any()) } returns
                party.qwer.twentyq.mq.queue.EnqueueResult.SUCCESS
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit
            every { messageProvider.get(any(), *anyVararg()) } returns "큐에 추가됨"
            every { messageProvider.get("chain.queued", "questions" to "척추동물인가요") } returns
                "\n\n📋 다음 질문 등록됨: 척추동물인가요"

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).isEqualTo("아마도 예")
            coVerify(exactly = 0) { queueCoordinator.enqueue(any(), any()) }
        }

    @Test
    fun `should NOT queue remaining questions when IF_TRUE and first answer is NO`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val sender = "테스터"
            val questions = listOf("동물인가요", "척추동물인가요", "포유류인가요")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "동물인가요", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_NO,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit
            every { messageProvider.get(GameMessageKeys.CHAIN_CONDITION_NOT_MET, "questions" to "척추동물인가요, 포유류인가요") } returns
                "(조건 불일치로 스킵: 척추동물인가요, 포유류인가요)"

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).contains("아니오")
            assertThat(result).contains("(조건 불일치로 스킵: 척추동물인가요, 포유류인가요)")
            coVerify(exactly = 0) { queueCoordinator.enqueue(any(), any()) }
        }

    @Test
    fun `should NOT queue remaining questions when IF_TRUE and first answer is MOSTLY_NO`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val sender = "테스터"
            val questions = listOf("동물인가요", "척추동물인가요")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "동물인가요", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.MOSTLY_NO,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit
            every { messageProvider.get(GameMessageKeys.CHAIN_CONDITION_NOT_MET, "questions" to "척추동물인가요") } returns
                "(조건 불일치로 스킵: 척추동물인가요)"

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).contains("아마도 아니오")
            assertThat(result).contains("(조건 불일치로 스킵: 척추동물인가요)")
            coVerify(exactly = 0) { queueCoordinator.enqueue(any(), any()) }
        }

    @Test
    fun `should NOT queue when IF_TRUE and first answer is INVALID`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val sender = "테스터"
            val questions = listOf("이상한 질문?", "두 번째 질문?")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "이상한 질문?", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.INVALID,
                    source = AnswerSource.FALLBACK_DEFAULT,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit
            every { messageProvider.get(GameMessageKeys.CHAIN_CONDITION_NOT_MET, "questions" to "두 번째 질문?") } returns
                "(조건 불일치로 스킵: 두 번째 질문?)"

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).contains("이해할 수 없는 질문입니다")
            assertThat(result).contains("(조건 불일치로 스킵: 두 번째 질문?)")
            coVerify(exactly = 0) { queueCoordinator.enqueue(any(), any()) }
        }

    @Test
    fun `should NOT show skip notification when only one question`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val sender = "테스터"
            val questions = listOf("동물인가요")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "동물인가요", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_NO,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )

            // When
            val result = handler.handle(chatId, command, userId, null)

            // Then
            assertThat(result).isEqualTo("아니오")
            assertThat(result).doesNotContain("스킵")
        }

    // Skip flag tests

    @Test
    fun `should set skip flag when IF_TRUE condition fails with NO answer`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("동물인가요", "척추동물인가요")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "동물인가요", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_NO,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            every { messageProvider.get(any(), *anyVararg()) } returns "스킵 메시지"
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit

            // When
            handler.handle(chatId, command, userId, null)

            // Then
            coVerify(exactly = 1) { queueCoordinator.setChainSkipFlag(chatId, userId) }
        }

    @Test
    fun `should set skip flag when IF_TRUE condition fails with MOSTLY_NO answer`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("동물인가요", "척추동물인가요")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "동물인가요", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.MOSTLY_NO,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            every { messageProvider.get(any(), *anyVararg()) } returns "스킵 메시지"
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit

            // When
            handler.handle(chatId, command, userId, null)

            // Then
            coVerify(exactly = 1) { queueCoordinator.setChainSkipFlag(chatId, userId) }
        }

    @Test
    fun `should set skip flag when IF_TRUE condition fails with INVALID answer`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("이상한 질문", "두 번째 질문")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "이상한 질문", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.INVALID,
                    source = AnswerSource.FALLBACK_DEFAULT,
                    guardDegraded = false,
                )
            every { messageProvider.get(any(), *anyVararg()) } returns "스킵 메시지"
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit

            // When
            handler.handle(chatId, command, userId, null)

            // Then
            coVerify(exactly = 1) { queueCoordinator.setChainSkipFlag(chatId, userId) }
        }

    @Test
    fun `should NOT set skip flag when IF_TRUE condition succeeds with YES`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("동물인가요", "척추동물인가요")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "동물인가요", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_YES,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.enqueue(any(), any()) } returns
                party.qwer.twentyq.mq.queue.EnqueueResult.SUCCESS
            every { messageProvider.get(any(), *anyVararg()) } returns "큐 메시지"
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit

            // When
            handler.handle(chatId, command, userId, null)

            // Then
            coVerify(exactly = 0) { queueCoordinator.setChainSkipFlag(any(), any()) }
        }

    @Test
    fun `should NOT set skip flag when IF_TRUE condition succeeds with MOSTLY_YES`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("동물인가요", "척추동물인가요")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "동물인가요", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.MOSTLY_YES,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.enqueue(any(), any()) } returns
                party.qwer.twentyq.mq.queue.EnqueueResult.SUCCESS
            every { messageProvider.get(any(), *anyVararg()) } returns "큐 메시지"
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit

            // When
            handler.handle(chatId, command, userId, null)

            // Then
            coVerify(exactly = 0) { queueCoordinator.setChainSkipFlag(any(), any()) }
        }

    @Test
    fun `should NOT set skip flag when ALWAYS condition regardless of answer`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("질문1", "질문2", "질문3")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.ALWAYS)

            coEvery { riddleService.answer(chatId, "질문1", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_NO,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.enqueue(any(), any()) } returns
                party.qwer.twentyq.mq.queue.EnqueueResult.SUCCESS
            every { messageProvider.get(any(), *anyVararg()) } returns "큐 메시지"
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit

            // When
            handler.handle(chatId, command, userId, null)

            // Then
            coVerify(exactly = 0) { queueCoordinator.setChainSkipFlag(any(), any()) }
        }

    @Test
    fun `should NOT set skip flag when only one question even if condition fails`() =
        runTest {
            // Given
            val chatId = "chat123"
            val userId = "user456"
            val questions = listOf("동물인가요")
            val command = Command.ChainedQuestion(questions, party.qwer.twentyq.model.ChainCondition.IF_TRUE)

            coEvery { riddleService.answer(chatId, "동물인가요", userId) } returns
                AnswerResult(
                    scale = FiveScaleKo.ALWAYS_NO,
                    source = AnswerSource.ENUM_SCHEMA_PRIMARY,
                    guardDegraded = false,
                )
            coEvery { queueCoordinator.setChainSkipFlag(any(), any()) } returns Unit

            // When
            handler.handle(chatId, command, userId, null)

            // Then
            coVerify(exactly = 0) { queueCoordinator.setChainSkipFlag(any(), any()) }
        }
}
