"""Tests for Gemini client."""

from dataclasses import replace
from unittest.mock import AsyncMock, MagicMock, patch

from google.api_core.exceptions import DeadlineExceeded
import pytest
from pydantic import BaseModel

from mcp_llm_server.config.settings import GeminiSettings, ThinkingConfig
from mcp_llm_server.exceptions import LLMModelError, LLMTimeoutError
from mcp_llm_server.infra.gemini_client import (
    GeminiClient,
    TokenUsage,
    ToolCall,
    get_gemini_client,
)


class TestTokenUsage:
    """Tests for TokenUsage dataclass."""

    def test_create_default(self) -> None:
        usage = TokenUsage()
        assert usage.input_tokens == 0
        assert usage.output_tokens == 0
        assert usage.total_tokens == 0
        assert usage.reasoning_tokens == 0

    def test_create_with_values(self) -> None:
        usage = TokenUsage(
            input_tokens=100,
            output_tokens=50,
            total_tokens=150,
            reasoning_tokens=20,
        )
        assert usage.input_tokens == 100
        assert usage.output_tokens == 50
        assert usage.total_tokens == 150
        assert usage.reasoning_tokens == 20


class TestToolCall:
    """Tests for ToolCall dataclass."""

    def test_create(self) -> None:
        tc = ToolCall(name="my_tool", args={"x": 1, "y": 2}, id="call_123")
        assert tc.name == "my_tool"
        assert tc.args == {"x": 1, "y": 2}
        assert tc.id == "call_123"

    def test_default_id(self) -> None:
        tc = ToolCall(name="tool", args={})
        assert tc.id == ""


@pytest.fixture
def mock_settings() -> GeminiSettings:
    """Create mock settings for testing."""
    return GeminiSettings(
        api_keys=["test-api-key"],
        default_model="gemini-2.5-flash",
        hints_model="gemini-2.5-flash",
        answer_model="gemini-2.5-flash",
        verify_model="gemini-2.5-flash",
        temperature=0.7,
        max_output_tokens=8192,
        thinking=ThinkingConfig(
            level_default="low",
            budget_default=1000,
        ),
    )


@pytest.fixture
def mock_llm() -> MagicMock:
    """Create mock LangChain LLM."""
    llm = MagicMock()

    # Mock response
    response = MagicMock()
    response.content = "Hello! How can I help you?"
    response.usage_metadata = {
        "input_tokens": 10,
        "output_tokens": 5,
        "total_tokens": 15,
    }
    response.tool_calls = None

    # Mock async methods
    llm.ainvoke = AsyncMock(return_value=response)
    llm.astream = AsyncMock(return_value=iter([response]))
    llm.with_structured_output = MagicMock(return_value=llm)
    llm.bind_tools = MagicMock(return_value=llm)

    return llm


class TestGeminiClientBuildMessages:
    """Tests for GeminiClient._build_messages."""

    def test_simple_prompt(self, mock_settings: GeminiSettings) -> None:
        client = GeminiClient(settings=mock_settings)
        messages = client._build_messages("Hello")

        assert len(messages) == 1
        assert messages[0].content == "Hello"

    def test_with_system_prompt(self, mock_settings: GeminiSettings) -> None:
        client = GeminiClient(settings=mock_settings)
        messages = client._build_messages("Hello", system_prompt="You are helpful")

        assert len(messages) == 2
        assert messages[0].content == "You are helpful"
        assert messages[1].content == "Hello"

    def test_with_history(self, mock_settings: GeminiSettings) -> None:
        client = GeminiClient(settings=mock_settings)
        history = [
            {"role": "user", "content": "Hi"},
            {"role": "assistant", "content": "Hello!"},
        ]
        messages = client._build_messages("How are you?", history=history)

        assert len(messages) == 3
        assert messages[0].content == "Hi"
        assert messages[1].content == "Hello!"
        assert messages[2].content == "How are you?"

    def test_full_conversation(self, mock_settings: GeminiSettings) -> None:
        client = GeminiClient(settings=mock_settings)
        history = [{"role": "user", "content": "Previous question"}]
        messages = client._build_messages(
            "New question",
            system_prompt="System instruction",
            history=history,
        )

        assert len(messages) == 3


class TestGeminiClientExtractText:
    """Tests for GeminiClient._extract_text."""

    def test_string_content(self, mock_settings: GeminiSettings) -> None:
        client = GeminiClient(settings=mock_settings)

        response = MagicMock()
        response.content = "Simple text response"

        text = client._extract_text(response)
        assert text == "Simple text response"

    def test_list_content(self, mock_settings: GeminiSettings) -> None:
        # Gemini 3 list format
        client = GeminiClient(settings=mock_settings)

        response = MagicMock()
        response.content = [
            "Part 1 ",
            {"text": "Part 2"},
            "Part 3",
        ]

        text = client._extract_text(response)
        assert text == "Part 1 Part 2Part 3"


class TestGeminiClientChat:
    """Tests for GeminiClient.chat method."""

    @pytest.mark.asyncio
    async def test_chat_basic(
        self, mock_settings: GeminiSettings, mock_llm: MagicMock
    ) -> None:
        with patch(
            "mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI",
            return_value=mock_llm,
        ):
            client = GeminiClient(settings=mock_settings)
            result = await client.chat("Hello")

            assert result == "Hello! How can I help you?"
            mock_llm.ainvoke.assert_called_once()

    @pytest.mark.asyncio
    async def test_chat_without_api_keys_raises(self) -> None:
        settings = GeminiSettings(api_keys=[], default_model="gemini-2.5-flash")

        client = GeminiClient(settings=settings)

        with pytest.raises(LLMModelError):
            await client.chat("hello")


class TestGeminiClientErrorHandling:
    """Ensure Gemini exceptions are translated into MCPLLM errors."""

    @pytest.mark.asyncio
    async def test_deadline_raises_timeout(
        self, mock_settings: GeminiSettings
    ) -> None:
        mock_llm = MagicMock()
        mock_llm.ainvoke = AsyncMock(side_effect=DeadlineExceeded("deadline exceeded"))

        with patch(
            "mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI",
            return_value=mock_llm,
        ):
            client = GeminiClient(settings=mock_settings)
            with pytest.raises(LLMTimeoutError):
                await client.chat("Hello")


class TestGeminiClientChatStructured:
    """Tests for GeminiClient.chat_structured method."""

    class GreetingResponse(BaseModel):
        message: str
        sentiment: str

    @pytest.mark.asyncio
    async def test_chat_structured(
        self, mock_settings: GeminiSettings, mock_llm: MagicMock
    ) -> None:
        # structured_llm returns pydantic model
        structured_response = self.GreetingResponse(
            message="Hello!", sentiment="positive"
        )
        mock_llm.with_structured_output.return_value.ainvoke = AsyncMock(
            return_value=structured_response
        )

        with patch(
            "mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI",
            return_value=mock_llm,
        ):
            client = GeminiClient(settings=mock_settings)
            result = await client.chat_structured(
                "Say hello", output_schema=self.GreetingResponse
            )

            assert isinstance(result, self.GreetingResponse)
            assert result.message == "Hello!"
            mock_llm.with_structured_output.assert_called_once_with(
                self.GreetingResponse
            )


class TestGeminiClientKeyRotation:
    """Tests for GeminiClient API key rotation and cache safety."""

    def test_round_robin_keys(self, mock_settings: GeminiSettings) -> None:
        mock_settings = replace(mock_settings, api_keys=["k1", "k2"])

        with patch(
            "mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI"
        ) as mock_llm:
            client = GeminiClient(settings=mock_settings)
            client._create_llm("gemini-2.5-flash")
            client._create_llm("gemini-2.5-flash", task="hints")

        keys = [call.kwargs["google_api_key"] for call in mock_llm.call_args_list]
        assert keys == ["k1", "k2"]


class TestGeminiClientChatWithTools:
    """Tests for GeminiClient.chat_with_tools method."""

    @pytest.mark.asyncio
    async def test_chat_with_tool_calls(
        self, mock_settings: GeminiSettings, mock_llm: MagicMock
    ) -> None:
        # Response with tool calls
        response = MagicMock()
        response.content = ""
        response.usage_metadata = {
            "input_tokens": 10,
            "output_tokens": 5,
            "total_tokens": 15,
        }
        response.tool_calls = [
            {"name": "get_weather", "args": {"city": "Seoul"}, "id": "call_1"},
        ]

        mock_llm.bind_tools.return_value.ainvoke = AsyncMock(return_value=response)

        with patch(
            "mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI",
            return_value=mock_llm,
        ):
            client = GeminiClient(settings=mock_settings)

            # dummy tool
            def get_weather(city: str) -> str:
                return f"Weather in {city}"

            _text, tool_calls = await client.chat_with_tools(
                "What's the weather?", tools=[get_weather]
            )

            assert len(tool_calls) == 1
            assert tool_calls[0].name == "get_weather"
            assert tool_calls[0].args == {"city": "Seoul"}

    @pytest.mark.asyncio
    async def test_chat_without_tool_calls(
        self, mock_settings: GeminiSettings, mock_llm: MagicMock
    ) -> None:
        response = MagicMock()
        response.content = "I don't need tools for this"
        response.usage_metadata = {
            "input_tokens": 10,
            "output_tokens": 5,
            "total_tokens": 15,
        }
        response.tool_calls = []

        mock_llm.bind_tools.return_value.ainvoke = AsyncMock(return_value=response)

        with patch(
            "mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI",
            return_value=mock_llm,
        ):
            client = GeminiClient(settings=mock_settings)
            text, tool_calls = await client.chat_with_tools("Hello", tools=[])

            assert text == "I don't need tools for this"
            assert tool_calls == []


class TestGeminiClientGetLlmForTask:
    """Tests for GeminiClient.get_llm_for_task method."""

    def test_get_llm_for_task(self, mock_settings: GeminiSettings) -> None:
        with patch(
            "mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI",
        ) as mock_llm_class:
            mock_llm_class.return_value = MagicMock()
            client = GeminiClient(settings=mock_settings)

            # 첫 번째 호출
            llm1 = client.get_llm_for_task("hints")
            # 두 번째 호출 (캐시에서)
            llm2 = client.get_llm_for_task("hints")

            assert llm1 is llm2  # same instance


class TestGeminiClientCaching:
    """Tests for LLM instance caching."""

    def test_model_caching(self, mock_settings: GeminiSettings) -> None:
        with patch(
            "mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI"
        ) as mock_llm_class:
            mock_llm_class.return_value = MagicMock()
            client = GeminiClient(settings=mock_settings)

            # 같은 model+task 조합은 캐시됨
            llm1 = client._get_llm("gemini-2.5-flash", None)
            llm2 = client._get_llm("gemini-2.5-flash", None)

            assert llm1 is llm2
            assert mock_llm_class.call_count == 1

    def test_different_tasks_different_llms(
        self, mock_settings: GeminiSettings
    ) -> None:
        with patch(
            "mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI"
        ) as mock_llm_class:
            mock_llm_class.side_effect = lambda **kwargs: MagicMock(**kwargs)
            client = GeminiClient(settings=mock_settings)

            llm1 = client._get_llm("gemini-2.5-flash", "hints")
            llm2 = client._get_llm("gemini-2.5-flash", "answer")

            # 다른 task는 다른 인스턴스
            assert llm1 is not llm2


class TestGeminiClientThinkingConfig:
    """Tests for thinking config selection."""

    def test_gemini25_uses_thinking_budget(self, mock_settings: GeminiSettings) -> None:
        budget_settings = replace(
            mock_settings,
            default_model="gemini-2.5-flash",
            thinking=replace(
                mock_settings.thinking,
                budget_default=2048,
                budget_answer=1024,
            ),
        )

        with patch(
            "mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI",
        ) as mock_llm_class:
            mock_llm_class.return_value = MagicMock()
            client = GeminiClient(settings=budget_settings)

            client._get_llm(budget_settings.default_model, "answer")

            kwargs = mock_llm_class.call_args.kwargs
            assert kwargs["thinking_budget"] == 1024
            assert "thinking_level" not in kwargs


class TestGetGeminiClientSingleton:
    """Tests for get_gemini_client singleton."""

    def test_returns_client(self) -> None:
        with patch("mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI"):
            client = get_gemini_client()
            assert isinstance(client, GeminiClient)

    def test_singleton(self) -> None:
        with patch("mcp_llm_server.infra.gemini_client.ChatGoogleGenerativeAI"):
            c1 = get_gemini_client()
            c2 = get_gemini_client()
            assert c1 is c2
