"""LangGraph-based session management with Redis checkpointer"""

from __future__ import annotations

from contextlib import AbstractAsyncContextManager, asynccontextmanager
from dataclasses import dataclass, field
from datetime import datetime, timedelta
from typing import TYPE_CHECKING, cast

from langchain_core.messages import AIMessage, BaseMessage, HumanMessage, SystemMessage
from langgraph.checkpoint.memory import MemorySaver
from langgraph.graph import END, START, MessagesState, StateGraph

from mcp_llm_server.config.logging import get_logger
from mcp_llm_server.config.settings import get_settings
from mcp_llm_server.exceptions import (
    SessionExpiredError,
    SessionLimitExceededError,
)


if TYPE_CHECKING:
    from collections.abc import AsyncIterator, Awaitable, Callable, Sequence

    from langchain_core.runnables import RunnableConfig
    from langgraph.checkpoint.redis.aio import AsyncRedisSaver
    from langgraph.graph.state import CompiledStateGraph

    from mcp_llm_server.types import JSONMapping, JSONValue
else:
    AsyncRedisSaver = MemorySaver  # type: ignore[assignment]


log = get_logger(__name__)


GraphConfig = dict[str, dict[str, str]]
type Checkpointer = MemorySaver | AsyncRedisSaver


@dataclass
class GraphSession:
    """Session metadata for LangGraph

    Attributes:
        session_id: Unique session identifier (UUID)
        model: LLM model name
        system_prompt: Optional system prompt
        domain_data: Domain-specific metadata (e.g., target, scenario)
        created_at: Session creation timestamp
        last_accessed: Last access timestamp
    """

    session_id: str
    model: str
    system_prompt: str | None = None
    domain_data: JSONMapping = field(default_factory=dict)
    created_at: datetime = field(default_factory=datetime.now)
    last_accessed: datetime = field(default_factory=datetime.now)


class LangGraphSessionManager:
    """LangGraph-based session manager with Redis checkpointer

    Features:
    - Automatic conversation history persistence via checkpointer
    - Thread-based session isolation
    - Support for InMemory (dev) and Redis (prod) backends
    """

    def __init__(
        self,
        checkpointer: Checkpointer | None = None,
        max_sessions: int = 50,
        session_ttl_minutes: int = 1440,
    ) -> None:
        self._checkpointer: Checkpointer | None = checkpointer
        self._sessions: dict[str, GraphSession] = {}
        self._graph: CompiledStateGraph[MessagesState] | None = None
        self._max_sessions = max_sessions
        self._session_ttl = timedelta(minutes=session_ttl_minutes)
        log.info(
            "LangGraphSessionManager initialized, max_sessions={}, ttl_min={}",
            max_sessions,
            session_ttl_minutes,
        )

    def _is_expired(self, session: GraphSession, now: datetime) -> bool:
        if self._session_ttl.total_seconds() <= 0:
            return False
        return now - session.last_accessed > self._session_ttl

    async def _prune_expired(self) -> set[str]:
        """Remove expired sessions and clear history."""
        if self._session_ttl.total_seconds() <= 0:
            return set()

        now = datetime.now()
        expired_ids = [
            session_id
            for session_id, session in self._sessions.items()
            if self._is_expired(session, now)
        ]
        for session_id in expired_ids:
            del self._sessions[session_id]
            await self.clear_history(session_id)
            log.info("LANGGRAPH_SESSION_EXPIRED session_id={}", session_id)
        return set(expired_ids)

    def _build_graph(self) -> CompiledStateGraph[MessagesState]:
        """Build simple message-passing graph"""
        builder: StateGraph[MessagesState] = StateGraph(MessagesState)

        # passthrough: 메시지를 그대로 반환 (실제 LLM 호출은 외부에서)
        def passthrough(state: MessagesState) -> dict[str, list[BaseMessage]]:
            messages = [
                msg for msg in state["messages"] if isinstance(msg, BaseMessage)
            ]
            return {"messages": cast("list[BaseMessage]", messages)}

        builder.add_node("passthrough", passthrough)
        builder.add_edge(START, "passthrough")
        builder.add_edge("passthrough", END)

        graph = builder.compile(checkpointer=self._checkpointer)
        return cast("CompiledStateGraph[MessagesState]", graph)

    @property
    def graph(self) -> CompiledStateGraph[MessagesState]:
        """Lazy graph initialization"""
        if self._graph is None:
            self._graph = self._build_graph()
        return self._graph

    async def create_session(
        self,
        session_id: str,
        model: str,
        system_prompt: str | None = None,
    ) -> GraphSession:
        """Create or get session

        Raises:
            SessionLimitExceededError: If max_sessions limit is reached
        """
        await self._prune_expired()

        if session_id in self._sessions:
            session = self._sessions[session_id]
            session.last_accessed = datetime.now()
            return session

        # 세션 수 제한 체크
        if len(self._sessions) >= self._max_sessions:
            log.warning(
                "SESSION_LIMIT_EXCEEDED current={}, max={}",
                len(self._sessions),
                self._max_sessions,
            )
            raise SessionLimitExceededError(self._max_sessions)

        session = GraphSession(
            session_id=session_id,
            model=model,
            system_prompt=system_prompt,
        )
        self._sessions[session_id] = session
        log.info(
            "LANGGRAPH_SESSION_CREATE session_id={}, model={}, total={}",
            session_id,
            model,
            len(self._sessions),
        )
        return session

    async def get_session(self, session_id: str) -> GraphSession | None:
        """Get existing session"""
        expired = await self._prune_expired()
        if session_id in expired:
            raise SessionExpiredError(session_id)

        session = self._sessions.get(session_id)
        if session:
            session.last_accessed = datetime.now()
        return session

    async def end_session(self, session_id: str) -> bool:
        """End session and clear history from checkpointer.

        Args:
            session_id: Session to end

        Returns:
            True if session existed and was removed
        """
        existed = session_id in self._sessions
        if existed:
            del self._sessions[session_id]

        # Clear history from checkpointer (Redis or Memory)
        await self.clear_history(session_id)
        log.info("LANGGRAPH_SESSION_END session_id={}, existed={}", session_id, existed)
        return existed

    async def create_fresh_session(
        self,
        session_id: str,
        model: str,
        system_prompt: str | None = None,
        domain_data: JSONMapping | None = None,
    ) -> GraphSession:
        """Create a fresh session, clearing any existing history.

        Always creates a new session. If session_id already exists,
        clears the history and resets metadata.

        Args:
            session_id: Unique session identifier
            model: LLM model name
            system_prompt: Optional system prompt
            domain_data: Domain-specific metadata

        Returns:
            New GraphSession instance

        Raises:
            SessionLimitExceededError: If max_sessions limit is reached
        """
        expired = await self._prune_expired()
        if session_id in expired:
            log.info("LANGGRAPH_SESSION_EXPIRED_CREATE session_id={}", session_id)
        # Clear existing history if any
        await self.clear_history(session_id)

        # Remove from sessions dict if exists
        if session_id in self._sessions:
            del self._sessions[session_id]

        # Check session limit
        if len(self._sessions) >= self._max_sessions:
            log.warning(
                "SESSION_LIMIT_EXCEEDED current={}, max={}",
                len(self._sessions),
                self._max_sessions,
            )
            raise SessionLimitExceededError(self._max_sessions)

        session = GraphSession(
            session_id=session_id,
            model=model,
            system_prompt=system_prompt,
            domain_data=domain_data or {},
        )
        self._sessions[session_id] = session
        log.info(
            "LANGGRAPH_SESSION_CREATE_FRESH session_id={}, model={}, total={}",
            session_id,
            model,
            len(self._sessions),
        )
        return session

    async def clear_history(self, session_id: str) -> bool:
        """Clear conversation history for a session (keeps session metadata).

        This resets the checkpointer state for the given thread_id,
        effectively starting fresh while keeping the same session_id.
        """
        if self._checkpointer is None:
            return True

        delete_thread = getattr(self._checkpointer, "adelete_thread", None)
        if callable(delete_thread):
            try:
                await cast(
                    "Callable[[str], Awaitable[None]]",
                    delete_thread,
                )(session_id)
                log.info("LANGGRAPH_HISTORY_DELETED session_id={}", session_id)
                return True
            except Exception as e:  # noqa: BLE001 - log checkpointer failures only
                log.warning(
                    "LANGGRAPH_DELETE_HISTORY_FAILED session_id={}, error={}",
                    session_id,
                    e,
                )

        config: GraphConfig = {"configurable": {"thread_id": session_id}}

        try:
            # 빈 상태로 덮어쓰기 - checkpointer에 새 체크포인트 생성
            await self.graph.ainvoke({"messages": []}, config)  # type: ignore[arg-type]
            log.info("LANGGRAPH_HISTORY_CLEARED session_id={}", session_id)
            return True
        except Exception as e:  # noqa: BLE001 log checkpointer failures only
            log.warning(
                "LANGGRAPH_CLEAR_HISTORY_FAILED session_id={}, error={}", session_id, e
            )
            return False

    def update_domain_data(self, session_id: str, key: str, value: JSONValue) -> bool:
        """Update domain-specific data for a session.

        Args:
            session_id: Session identifier
            key: Data key (e.g., 'target', 'scenario')
            value: Data value

        Returns:
            True if session exists and was updated
        """
        session = self._sessions.get(session_id)
        if session:
            session.domain_data[key] = value
            session.last_accessed = datetime.now()
            return True
        return False

    def get_domain_data(self, session_id: str, key: str) -> JSONValue | None:
        """Get domain-specific data for a session.

        Args:
            session_id: Session identifier
            key: Data key

        Returns:
            Data value or None if not found
        """
        session = self._sessions.get(session_id)
        if session:
            session.last_accessed = datetime.now()
            return session.domain_data.get(key)
        return None

    async def add_messages(
        self,
        session_id: str,
        messages: Sequence[BaseMessage],
    ) -> list[BaseMessage]:
        """Add messages to session history via graph invoke

        Args:
            session_id: Thread identifier
            messages: Messages to add

        Returns:
            Updated message history
        """
        expired = await self._prune_expired()
        if session_id in expired:
            raise SessionExpiredError(session_id)

        config: GraphConfig = {"configurable": {"thread_id": session_id}}
        state = cast("MessagesState", {"messages": list(messages)})

        # invoke를 통해 checkpointer에 자동 저장
        result = await self.graph.ainvoke(state, cast("RunnableConfig", config))
        messages_result = result.get("messages", [])
        if isinstance(messages_result, list):
            return [msg for msg in messages_result if isinstance(msg, BaseMessage)]
        return []

    async def get_history(self, session_id: str) -> list[BaseMessage]:
        """Get conversation history from checkpointer

        Args:
            session_id: Thread identifier

        Returns:
            Message history
        """
        expired = await self._prune_expired()
        if session_id in expired:
            raise SessionExpiredError(session_id)

        config: GraphConfig = {"configurable": {"thread_id": session_id}}

        # 빈 메시지로 현재 상태 조회
        try:
            state = await self.graph.aget_state(config)  # type: ignore[arg-type]
            if state and state.values:
                messages = state.values.get("messages")
                if isinstance(messages, list):
                    return [msg for msg in messages if isinstance(msg, BaseMessage)]
        except Exception as e:  # noqa: BLE001 history fetch should not raise
            log.warning(
                "LANGGRAPH_GET_HISTORY_FAILED session_id={}, error={}", session_id, e
            )

        return []

    def get_config(self, session_id: str) -> GraphConfig:
        """Get LangGraph config for session"""
        return {"configurable": {"thread_id": session_id}}

    # =========================================================================
    # Compatibility methods (migrated from session_manager.py)
    # =========================================================================

    async def get_or_create_session(
        self,
        session_id: str,
        model: str,
        system_prompt: str | None = None,
    ) -> GraphSession:
        """Alias for create_session (backward compatibility)"""
        return await self.create_session(session_id, model, system_prompt)

    async def get_history_as_dicts(self, session_id: str) -> list[dict[str, str]]:
        """Get conversation history as list of dicts

        Args:
            session_id: Session identifier

        Returns:
            List of {"role": "user"|"assistant", "content": "..."}
        """
        messages = await self.get_history(session_id)
        return from_langchain_messages(messages)

    async def add_message(
        self,
        session_id: str,
        role: str,
        content: str,
    ) -> bool:
        """Add a single message to session history

        Args:
            session_id: Session identifier
            role: Message role ("user" | "assistant")
            content: Message content

        Returns:
            True if added successfully
        """
        msg: BaseMessage
        if role == "user":
            msg = HumanMessage(content=content)
        elif role == "assistant":
            msg = AIMessage(content=content)
        else:
            msg = SystemMessage(content=content)

        await self.add_messages(session_id, [msg])
        return True

    def get_session_info(self, session_id: str) -> dict[str, str | int] | None:
        """Get session metadata

        Args:
            session_id: Session identifier

        Returns:
            dict with session info or None
        """
        session = self._sessions.get(session_id)
        if session:
            return {
                "session_id": session.session_id,
                "model": session.model,
                "created_at": session.created_at.isoformat(),
                "last_accessed": session.last_accessed.isoformat(),
            }
        return None

    def get_active_session_count(self) -> int:
        """Get number of active sessions"""
        return len(self._sessions)


# Checkpointer factory


@asynccontextmanager
async def get_redis_checkpointer() -> AsyncIterator[Checkpointer]:
    """Get Redis Stack checkpointer as async context manager"""
    settings = get_settings()

    if not settings.redis.enabled:
        log.debug("Using InMemorySaver (Redis disabled)")
        yield MemorySaver()
        return

    try:
        from langgraph.checkpoint.redis.aio import AsyncRedisSaver

        # TTL 설정: 세션 만료 시간 + 읽기 시 갱신
        ttl_config = {
            "default_ttl": settings.session.session_ttl_minutes,
            "refresh_on_read": True,
        }

        async with AsyncRedisSaver.from_conn_string(
            settings.redis.url, ttl=ttl_config
        ) as checkpointer:
            log.info(
                "AsyncRedisSaver connected to {}, TTL={}min",
                settings.redis.url,
                settings.session.session_ttl_minutes,
            )
            yield checkpointer
    except ImportError:
        log.warning("langgraph-checkpoint-redis not available, using InMemorySaver")
        yield MemorySaver()
    except Exception as e:  # noqa: BLE001 - Redis connection issues fallback to memory
        log.error("Redis connection failed: {}, using InMemorySaver", e)
        yield MemorySaver()


# Message conversion utilities


def to_langchain_messages(
    history: list[dict[str, str]],
    system_prompt: str | None = None,
) -> list[BaseMessage]:
    """Convert dict history to LangChain messages"""
    messages: list[BaseMessage] = []

    if system_prompt:
        messages.append(SystemMessage(content=system_prompt))

    for msg in history:
        role = msg.get("role", "user")
        content = msg.get("content", "")

        if role == "user":
            messages.append(HumanMessage(content=content))
        elif role == "assistant":
            messages.append(AIMessage(content=content))
        elif role == "system":
            messages.append(SystemMessage(content=content))

    return messages


def from_langchain_messages(messages: Sequence[BaseMessage]) -> list[dict[str, str]]:
    """Convert LangChain messages to dict format"""
    result: list[dict[str, str]] = []

    for msg in messages:
        if isinstance(msg, HumanMessage):
            result.append({"role": "user", "content": str(msg.content)})
        elif isinstance(msg, AIMessage):
            result.append({"role": "assistant", "content": str(msg.content)})
        elif isinstance(msg, SystemMessage):
            result.append({"role": "system", "content": str(msg.content)})

    return result


# Singleton


_manager: LangGraphSessionManager | None = None
_redis_saver: Checkpointer | None = None
_redis_cm: AbstractAsyncContextManager[AsyncRedisSaver] | None = None


async def init_langgraph_with_redis() -> LangGraphSessionManager:
    """Initialize LangGraph session manager with Redis checkpointer.

    Must be called during server startup (lifespan).
    """
    global _manager, _redis_saver, _redis_cm
    settings = get_settings()

    if not settings.redis.enabled:
        log.info("Redis disabled, using InMemorySaver")
        _manager = LangGraphSessionManager(
            checkpointer=MemorySaver(),
            max_sessions=settings.session.max_sessions,
            session_ttl_minutes=settings.session.session_ttl_minutes,
        )
        return _manager

    try:
        from langgraph.checkpoint.redis.aio import AsyncRedisSaver

        # TTL 설정: 세션 만료 시간 + 읽기 시 갱신
        ttl_config = {
            "default_ttl": settings.session.session_ttl_minutes,  # 1440분 = 24시간
            "refresh_on_read": True,  # 읽기 시 TTL 갱신
        }

        # from_conn_string returns a context manager, __aenter__ returns the actual saver
        _redis_cm = AsyncRedisSaver.from_conn_string(settings.redis.url, ttl=ttl_config)
        _redis_saver = await _redis_cm.__aenter__()  # pylint: disable=no-member
        _manager = LangGraphSessionManager(
            checkpointer=_redis_saver,
            max_sessions=settings.session.max_sessions,
            session_ttl_minutes=settings.session.session_ttl_minutes,
        )
        log.info(
            "LangGraph initialized with AsyncRedisSaver: {}, TTL={}min",
            settings.redis.url,
            settings.session.session_ttl_minutes,
        )
    except ImportError:
        log.warning("langgraph-checkpoint-redis not available, using InMemorySaver")
        _manager = LangGraphSessionManager(
            checkpointer=MemorySaver(),
            max_sessions=settings.session.max_sessions,
            session_ttl_minutes=settings.session.session_ttl_minutes,
        )
    except Exception as e:  # noqa: BLE001 - Redis connection issues fallback to memory
        log.error("Redis connection failed: {}, using InMemorySaver", e)
        _manager = LangGraphSessionManager(
            checkpointer=MemorySaver(),
            max_sessions=settings.session.max_sessions,
            session_ttl_minutes=settings.session.session_ttl_minutes,
        )

    return _manager


async def shutdown_langgraph() -> None:
    """Cleanup Redis connection on shutdown."""
    global _redis_cm, _redis_saver
    if _redis_cm is not None:
        try:
            await _redis_cm.__aexit__(None, None, None)  # pylint: disable=no-member
            log.info("AsyncRedisSaver connection closed")
        except Exception as e:  # noqa: BLE001 - log and continue shutdown cleanup
            log.warning("Error closing Redis connection: {}", e)
        _redis_cm = None
        _redis_saver = None


async def get_langgraph_manager() -> LangGraphSessionManager:
    """Get LangGraph session manager singleton with lazy initialization.

    Initializes Redis connection on first call within the current event loop.
    Falls back to MemorySaver if Redis is disabled or unavailable.
    """
    global _manager, _redis_saver, _redis_cm

    if _manager is not None:
        return _manager

    settings = get_settings()

    if not settings.redis.enabled:
        log.info("Redis disabled, using InMemorySaver")
        _manager = LangGraphSessionManager(
            checkpointer=MemorySaver(),
            max_sessions=settings.session.max_sessions,
            session_ttl_minutes=settings.session.session_ttl_minutes,
        )
        return _manager

    try:
        from langgraph.checkpoint.redis.aio import AsyncRedisSaver

        # TTL config
        ttl_config = {
            "default_ttl": settings.session.session_ttl_minutes,
            "refresh_on_read": True,
        }

        # from_conn_string returns context manager
        _redis_cm = AsyncRedisSaver.from_conn_string(settings.redis.url, ttl=ttl_config)
        _redis_saver = await _redis_cm.__aenter__()  # pylint: disable=no-member
        _manager = LangGraphSessionManager(
            checkpointer=_redis_saver,
            max_sessions=settings.session.max_sessions,
            session_ttl_minutes=settings.session.session_ttl_minutes,
        )
        log.info(
            "LangGraph initialized with AsyncRedisSaver: {}, TTL={}min",
            settings.redis.url,
            settings.session.session_ttl_minutes,
        )
    except ImportError:
        log.warning("langgraph-checkpoint-redis not available, using InMemorySaver")
        _manager = LangGraphSessionManager(
            checkpointer=MemorySaver(),
            max_sessions=settings.session.max_sessions,
            session_ttl_minutes=settings.session.session_ttl_minutes,
        )
    except Exception as e:  # noqa: BLE001 - Redis connection issues fallback to memory
        log.error("Redis connection failed: {}, using InMemorySaver", e)
        _manager = LangGraphSessionManager(
            checkpointer=MemorySaver(),
            max_sessions=settings.session.max_sessions,
            session_ttl_minutes=settings.session.session_ttl_minutes,
        )

    return _manager


async def get_langgraph_health() -> dict[str, bool | int | str]:
    """Return LangGraph backend health info."""
    settings = get_settings()
    redis_enabled = settings.redis.enabled
    backend = "memory"
    redis_connected = False
    session_count = 0

    try:
        if redis_enabled and _redis_saver is None:
            # self-heal: 재시도하여 Redis Saver 재연결 시도
            await init_langgraph_with_redis()

        manager = await get_langgraph_manager()
        session_count = manager.get_active_session_count()
        if _redis_saver is not None:
            backend = "redis"
            redis_connected = True
    except Exception as e:  # noqa: BLE001  # pragma: no cover - defensive
        log.warning("LANGGRAPH_HEALTH_FAILED error={}", e)

    return {
        "redis_enabled": redis_enabled,
        "redis_connected": redis_connected,
        "backend": backend,
        "session_count": session_count,
        "redis_url": settings.redis.url if redis_enabled else "",
        "session_ttl_minutes": settings.session.session_ttl_minutes,
    }


async def ping_langgraph_backend() -> bool:
    """Ping LangGraph backend (Redis) if available."""
    if _redis_saver is None:
        return True

    redis_client = (
        getattr(_redis_saver, "redis", None)
        or getattr(_redis_saver, "client", None)
        or getattr(_redis_saver, "_redis", None)
    )
    if redis_client is None:
        return False

    try:
        response = await redis_client.ping()
    except Exception as e:  # noqa: BLE001 - defensive ping
        log.warning("LANGGRAPH_REDIS_PING_FAILED error={}", e)
        return False

    return bool(response)
