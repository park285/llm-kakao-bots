package party.qwer.twentyq.model

import org.assertj.core.api.Assertions.assertThat
import org.junit.jupiter.api.Test

class CommandRequiresLlmTest {
    @Test
    fun `help should require llm`() {
        assertThat(Command.Help.requiresLlm()).isTrue()
    }

    @Test
    fun `surrender should not require llm`() {
        assertThat(Command.Surrender.requiresLlm()).isFalse()
    }

    @Test
    fun `agree should not require llm`() {
        assertThat(Command.Agree.requiresLlm()).isFalse()
    }

    @Test
    fun `reject should not require llm`() {
        assertThat(Command.Reject.requiresLlm()).isFalse()
    }

    @Test
    fun `status should not require llm`() {
        assertThat(Command.Status.requiresLlm()).isFalse()
    }
}
