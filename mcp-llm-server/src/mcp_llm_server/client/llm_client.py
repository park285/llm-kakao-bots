"""LLM Client - High-level MCP client for external LLM hosts

Supports two transport modes:
- stdio: Local process communication (default)
- streamable-http: Remote HTTP server connection

Usage:
    # Stdio transport (local MCP server)
    async with LLMClient(transport="stdio", command="mcp-llm-server") as client:
        response = await client.chat("Hello!")

    # Streamable HTTP transport (remote MCP server)
    async with LLMClient(transport="http", url="http://localhost:8000/mcp") as client:
        response = await client.chat("Hello!")

    # Session-based chat
    await client.create_session("user123", model="gemini-2.5-flash")
    response = await client.chat_session("user123", "What's your name?")
    await client.end_session("user123")

    # Guard check
    is_safe = not await client.is_malicious("user input")

    # NLP analysis
    tokens = await client.analyze("한국어 텍스트")
"""

import json
import logging
from collections.abc import AsyncIterator
from contextlib import AbstractAsyncContextManager, asynccontextmanager
from dataclasses import dataclass
from enum import Enum
from types import TracebackType
from typing import Literal, cast

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

from mcp_llm_server.types import JSONMapping, JSONValue


log = logging.getLogger(__name__)


class TransportType(str, Enum):
    """MCP transport type"""

    STDIO = "stdio"
    HTTP = "http"
    STREAMABLE_HTTP = "streamable-http"


@dataclass
class GuardResult:
    """Injection guard evaluation result"""

    score: float
    hits: list[JSONMapping]
    threshold: float
    malicious: bool


@dataclass
class NlpToken:
    """Morphological analysis token"""

    form: str
    tag: str
    position: int
    length: int


@dataclass
class NlpHeuristics:
    """Heuristic analysis result"""

    numeric_quantifier: bool
    unit_noun: bool
    boundary_ref: bool
    comparison_word: bool


type TransportContext = AbstractAsyncContextManager[tuple[object, ...]]
type SessionContext = AbstractAsyncContextManager[ClientSession]
type ToolArguments = JSONMapping
type ToolResult = JSONValue | str | None


class LLMClient:
    """High-level MCP client for LLM operations

    Provides typed methods for:
    - LLM chat (stateless and session-based)
    - Injection guard evaluation
    - Korean NLP analysis

    Supports both stdio (local) and streamable-http (remote) transports.
    """

    def __init__(
        self,
        transport: Literal["stdio", "http", "streamable-http"] = "stdio",
        *,
        # stdio transport options
        command: str | None = None,
        args: list[str] | None = None,
        # http transport options
        url: str | None = None,
        headers: dict[str, str] | None = None,
    ) -> None:
        """Initialize client

        Args:
            transport: Transport type ("stdio", "http", or "streamable-http")

            For stdio transport:
                command: MCP server command (default: mcp-llm-server)
                args: Optional server arguments

            For http/streamable-http transport:
                url: MCP server URL (e.g., "http://localhost:8000/mcp")
                headers: Optional HTTP headers for authentication
        """
        self._transport = TransportType(transport)
        self._session: ClientSession | None = None
        self._transport_cm: TransportContext | None = None
        self._session_cm: SessionContext | None = None

        # stdio transport options
        self._command: str = command or "mcp-llm-server"
        self._args: list[str] = args or []
        # http transport options
        self._url: str | None = url
        self._headers: dict[str, str] | None = headers

        if self._transport != TransportType.STDIO and not url:
            raise ValueError("URL is required for HTTP transport")

    async def __aenter__(self) -> "LLMClient":
        """Async context manager entry"""
        await self.connect()
        return self

    async def __aexit__(
        self,
        exc_type: type[BaseException] | None,
        exc_val: BaseException | None,
        exc_tb: TracebackType | None,
    ) -> None:
        """Async context manager exit"""
        await self.disconnect()

    async def connect(self) -> None:
        """Connect to MCP server"""
        if self._session:
            return

        if self._transport == TransportType.STDIO:
            await self._connect_stdio()
        else:
            await self._connect_http()

        log.info("LLMClient connected via %s transport", self._transport.value)

    async def _connect_stdio(self) -> None:
        """Connect using stdio transport"""
        server_params = StdioServerParameters(
            command=self._command,
            args=self._args or [],
        )
        self._transport_cm = stdio_client(server_params)
        read_stream, write_stream = await self._transport_cm.__aenter__()

        self._session_cm = ClientSession(read_stream, write_stream)
        self._session = await self._session_cm.__aenter__()
        await self._session.initialize()

    async def _connect_http(self) -> None:
        """Connect using streamable HTTP transport"""
        from mcp.client.streamable_http import streamablehttp_client

        if not self._url:
            raise RuntimeError("URL not configured for HTTP transport")

        # streamablehttp_client returns (read, write, get_session_id)
        self._transport_cm = streamablehttp_client(
            self._url,
            headers=self._headers,
        )
        read_stream, write_stream, _ = await self._transport_cm.__aenter__()

        self._session_cm = ClientSession(read_stream, write_stream)
        self._session = await self._session_cm.__aenter__()
        await self._session.initialize()

    async def disconnect(self) -> None:
        """Disconnect from MCP server"""
        if self._session_cm:
            await self._session_cm.__aexit__(None, None, None)
            self._session_cm = None
            self._session = None

        if self._transport_cm:
            await self._transport_cm.__aexit__(None, None, None)
            self._transport_cm = None

        log.info("LLMClient disconnected")

    @property
    def transport(self) -> str:
        """Get current transport type"""
        return self._transport.value

    @property
    def is_connected(self) -> bool:
        """Check if client is connected"""
        return self._session is not None

    async def _call_tool(self, name: str, arguments: ToolArguments) -> ToolResult:
        """Call MCP tool and return result"""
        if not self._session:
            raise RuntimeError(
                "Client not connected. Use 'async with' or call connect()"
            )

        result = await self._session.call_tool(name, arguments)

        # Extract content from result
        if result.content and len(result.content) > 0:
            content = result.content[0]
            text = getattr(content, "text", None)
            if isinstance(text, str):
                # Try to parse as JSON
                try:
                    return cast("JSONValue", json.loads(text))
                except json.JSONDecodeError:
                    return text
            return str(content)

        return None

    @staticmethod
    def _expect_mapping(value: ToolResult, context: str) -> JSONMapping:
        if not isinstance(value, dict):
            raise TypeError(
                f"{context} returned {type(value).__name__}, expected mapping"
            )
        return value

    @staticmethod
    def _expect_str(value: ToolResult, context: str) -> str:
        if isinstance(value, str):
            return value
        raise TypeError(f"{context} returned {type(value).__name__}, expected string")

    @staticmethod
    def _expect_mapping_list(value: ToolResult, context: str) -> list[JSONMapping]:
        if not isinstance(value, list):
            raise TypeError(
                f"{context} returned {type(value).__name__}, expected list of mappings"
            )
        mappings: list[JSONMapping] = []
        for item in value:
            if not isinstance(item, dict):
                raise TypeError(
                    f"{context} returned non-dict item: {type(item).__name__}"
                )
            mappings.append(item)
        return mappings

    @staticmethod
    def _expect_float(value: JSONValue, context: str) -> float:
        if isinstance(value, (int, float, str, bool)):
            return float(value)
        raise TypeError(f"{context} returned non-numeric value: {type(value).__name__}")

    async def list_tools(self) -> list[str]:
        """List available tools on the server

        Returns:
            List of tool names
        """
        if not self._session:
            raise RuntimeError("Client not connected")

        tools = await self._session.list_tools()
        return [tool.name for tool in tools.tools]

    # =========================================================================
    # LLM Methods
    # =========================================================================

    async def chat(
        self,
        prompt: str,
        system_prompt: str | None = None,
        history: list[dict[str, str]] | None = None,
        model: str | None = None,
    ) -> str:
        """Stateless LLM chat

        Args:
            prompt: User message
            system_prompt: Optional system instruction
            history: Optional conversation history
            model: Optional model override

        Returns:
            LLM response text
        """
        args: ToolArguments = {"prompt": prompt}
        if system_prompt:
            args["system_prompt"] = system_prompt
        if history:
            args["history"] = history  # type: ignore[assignment]
        if model:
            args["model"] = model

        result = await self._call_tool("llm_chat_stateless", args)
        return self._expect_str(result, "llm_chat_stateless")

    async def chat_stream(
        self,
        prompt: str,
        system_prompt: str | None = None,
        history: list[dict[str, str]] | None = None,
        model: str | None = None,
    ) -> str:
        """Streaming LLM chat (returns complete response)

        Args:
            prompt: User message
            system_prompt: Optional system instruction
            history: Optional conversation history
            model: Optional model override

        Returns:
            Complete LLM response text
        """
        args: ToolArguments = {"prompt": prompt}
        if system_prompt:
            args["system_prompt"] = system_prompt
        if history:
            args["history"] = history  # type: ignore[assignment]
        if model:
            args["model"] = model

        result = await self._call_tool("llm_stream", args)
        return self._expect_str(result, "llm_stream")

    # =========================================================================
    # Session Methods
    # =========================================================================

    async def create_session(
        self,
        session_id: str,
        model: str | None = None,
        system_prompt: str | None = None,
    ) -> JSONMapping:
        """Create a chat session

        Args:
            session_id: Unique session identifier
            model: Model to use
            system_prompt: Optional system instruction

        Returns:
            dict with session_id, model, created status
        """
        args: ToolArguments = {"session_id": session_id}
        if model:
            args["model"] = model
        if system_prompt:
            args["system_prompt"] = system_prompt

        result = await self._call_tool("session_create", args)
        if isinstance(result, dict):
            return result
        raise TypeError("Unexpected response from session_create")

    async def chat_session(self, session_id: str, prompt: str) -> str:
        """Session-based chat

        Args:
            session_id: Session identifier
            prompt: User message

        Returns:
            LLM response text
        """
        result = await self._call_tool(
            "llm_chat_session",
            {"session_id": session_id, "prompt": prompt},
        )
        return self._expect_str(result, "llm_chat_session")

    async def end_session(self, session_id: str) -> JSONMapping:
        """End a chat session

        Args:
            session_id: Session to end

        Returns:
            dict with session_id and removed status
        """
        result = await self._call_tool("session_end", {"session_id": session_id})
        if isinstance(result, dict):
            return result
        raise TypeError("Unexpected response from session_end")

    async def get_session_info(self, session_id: str) -> JSONMapping | None:
        """Get session information

        Args:
            session_id: Session identifier

        Returns:
            Session info dict or None if not found
        """
        result = await self._call_tool("session_info", {"session_id": session_id})
        if isinstance(result, dict) and "error" not in result:
            return result
        return None

    # =========================================================================
    # Guard Methods
    # =========================================================================

    async def evaluate_guard(self, input_text: str) -> GuardResult:
        """Evaluate input for injection attacks

        Args:
            input_text: User input to evaluate

        Returns:
            GuardResult with score, hits, threshold, malicious flag
        """
        result = self._expect_mapping(
            await self._call_tool("guard_evaluate", {"input_text": input_text}),
            "guard_evaluate",
        )
        hits_value = result.get("hits", [])
        hits = self._expect_mapping_list(hits_value, "guard_evaluate.hits")
        return GuardResult(
            score=self._expect_float(result["score"], "guard_evaluate.score"),
            hits=hits,
            threshold=self._expect_float(
                result["threshold"], "guard_evaluate.threshold"
            ),
            malicious=bool(result["malicious"]),
        )

    async def is_malicious(self, input_text: str) -> bool:
        """Quick check if input is malicious

        Args:
            input_text: User input to check

        Returns:
            True if input is considered malicious
        """
        result = await self._call_tool("guard_is_malicious", {"input_text": input_text})
        if isinstance(result, bool):
            return result
        raise TypeError("guard_is_malicious returned non-bool response")

    # =========================================================================
    # NLP Methods
    # =========================================================================

    async def analyze(self, text: str) -> list[NlpToken]:
        """Perform Korean morphological analysis

        Args:
            text: Korean text to analyze

        Returns:
            List of NlpToken
        """
        result = self._expect_mapping_list(
            await self._call_tool("nlp_analyze", {"text": text}), "nlp_analyze"
        )
        tokens: list[NlpToken] = []
        for item in result:
            position_raw = cast("int | float | str | bool", item.get("position", 0))
            length_raw = cast("int | float | str | bool", item.get("length", 0))
            tokens.append(
                NlpToken(
                    form=str(item.get("form", "")),
                    tag=str(item.get("tag", "")),
                    position=int(position_raw),
                    length=int(length_raw),
                )
            )
        return tokens

    async def anomaly_score(self, text: str) -> float:
        """Calculate anomaly score for text

        Args:
            text: Text to analyze

        Returns:
            Anomaly score (0.0 ~ 1.0)
        """
        result = await self._call_tool("nlp_anomaly_score", {"text": text})
        if isinstance(result, (int, float, str, bool)):
            return float(result)
        raise TypeError("nlp_anomaly_score returned non-numeric response")

    async def analyze_heuristics(self, text: str) -> NlpHeuristics:
        """Analyze text for answer validation heuristics

        Args:
            text: Text to analyze

        Returns:
            NlpHeuristics with detection flags
        """
        result = self._expect_mapping(
            await self._call_tool("nlp_heuristics", {"text": text}),
            "nlp_heuristics",
        )
        return NlpHeuristics(
            numeric_quantifier=bool(result["numeric_quantifier"]),
            unit_noun=bool(result["unit_noun"]),
            boundary_ref=bool(result["boundary_ref"]),
            comparison_word=bool(result["comparison_word"]),
        )


# =========================================================================
# Convenience functions
# =========================================================================


@asynccontextmanager
async def get_llm_client(
    transport: Literal["stdio", "http", "streamable-http"] = "stdio",
    *,
    command: str | None = None,
    args: list[str] | None = None,
    url: str | None = None,
    headers: dict[str, str] | None = None,
) -> AsyncIterator[LLMClient]:
    """Get LLM client as async context manager

    Usage:
        # Stdio transport
        async with get_llm_client(transport="stdio", command="mcp-server") as client:
            response = await client.chat("Hello!")

        # HTTP transport
        async with get_llm_client(transport="http", url="http://localhost:8000/mcp") as client:
            response = await client.chat("Hello!")
    """
    client = LLMClient(
        transport=transport,
        command=command,
        args=args,
        url=url,
        headers=headers,
    )
    try:
        await client.connect()
        yield client
    finally:
        await client.disconnect()


@asynccontextmanager
async def get_http_client(
    url: str,
    headers: dict[str, str] | None = None,
) -> AsyncIterator[LLMClient]:
    """Convenience function for HTTP transport

    Usage:
        async with get_http_client("http://localhost:8000/mcp") as client:
            response = await client.chat("Hello!")
    """
    async with get_llm_client(transport="http", url=url, headers=headers) as client:
        yield client


@asynccontextmanager
async def get_stdio_client(
    command: str = "mcp-llm-server",
    args: list[str] | None = None,
) -> AsyncIterator[LLMClient]:
    """Convenience function for stdio transport

    Usage:
        async with get_stdio_client("mcp-llm-server") as client:
            response = await client.chat("Hello!")
    """
    async with get_llm_client(transport="stdio", command=command, args=args) as client:
        yield client
