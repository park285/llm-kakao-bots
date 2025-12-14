package party.qwer.twentyq.bridge

import io.mockk.coEvery
import io.mockk.mockk
import kotlinx.coroutines.test.runTest
import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Test
import party.qwer.twentyq.model.ChainCondition
import party.qwer.twentyq.model.Command
import party.qwer.twentyq.mcp.NlpAnalysis
import party.qwer.twentyq.rest.NlpRestClient

class CommandParserTest {
    private val nlpRestClient = mockk<NlpRestClient>()
    private val parser = CommandParser("/스자", nlpRestClient)

    init {
        // NLP 분석을 단순화
        coEvery { nlpRestClient.analyzeNlp(any()) } returns NlpAnalysis()
    }

    // IF 조건 테스트
    @Test
    fun `should parse IF_TRUE with if keyword`() =
        runTest {
            // Given
            val input = "/스자 if 동물인가요, 척추동물인가요"

            // When
            val command = parser.parse(input)

            // Then
            assertThat(command).isInstanceOf(Command.ChainedQuestion::class.java)
            val chainCmd = command as Command.ChainedQuestion
            assertThat(chainCmd.condition).isEqualTo(ChainCondition.IF_TRUE)
            assertThat(chainCmd.questions).hasSize(2)
            assertThat(chainCmd.questions[0]).isEqualTo("동물인가요")
            assertThat(chainCmd.questions[1]).isEqualTo("척추동물인가요")
        }

    @Test
    fun `should parse ALWAYS without condition keyword`() =
        runTest {
            // Given
            val input = "/스자 동물인가요, 척추동물인가요, 포유류인가요"

            // When
            val command = parser.parse(input)

            // Then
            assertThat(command).isInstanceOf(Command.ChainedQuestion::class.java)
            val chainCmd = command as Command.ChainedQuestion
            assertThat(chainCmd.condition).isEqualTo(ChainCondition.ALWAYS)
            assertThat(chainCmd.questions).hasSize(3)
        }

    @Test
    fun `should handle uppercase IF keyword`() =
        runTest {
            // Given
            val input = "/스자 IF 동물인가요, 척추동물인가요"

            // When
            val command = parser.parse(input)

            // Then
            assertThat(command).isInstanceOf(Command.ChainedQuestion::class.java)
            val chainCmd = command as Command.ChainedQuestion
            assertThat(chainCmd.condition).isEqualTo(ChainCondition.IF_TRUE)
        }

    @Test
    fun `should NOT parse if without comma as chain question`() =
        runTest {
            // Given
            val input = "/스자 if 동물인가요"

            // When
            val command = parser.parse(input)

            // Then
            // 쉼표 없으면 일반 Ask 명령으로 파싱됨
            assertThat(command).isInstanceOf(Command.Ask::class.java)
        }

    @Test
    fun `should parse single question after if as Ask command`() =
        runTest {
            // Given
            val input = "/스자 if 동물인가요"

            // When
            val command = parser.parse(input)

            // Then
            assertThat(command).isInstanceOf(Command.Ask::class.java)
            val askCmd = command as Command.Ask
            assertThat(askCmd.question).isEqualTo("if 동물인가요")
        }
}
