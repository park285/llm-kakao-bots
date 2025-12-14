"""MCP LLM Server - HTTP REST API."""

import asyncio
import os
from collections.abc import AsyncIterator, Awaitable
from contextlib import asynccontextmanager
from typing import TYPE_CHECKING, Any, cast

from fastapi import FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import ORJSONResponse
from hypercorn.asyncio import serve
from hypercorn.config import Config as HyperConfig
from langchain_core.messages import AIMessage, HumanMessage
from langchain_core.prompts import ChatPromptTemplate
from pydantic import BaseModel, Field


if TYPE_CHECKING:
    from hypercorn.typing import ASGIFramework
    from langchain_core.messages import BaseMessage

from mcp_llm_server.config.logging import get_logger, setup_logging
from mcp_llm_server.config.settings import get_settings, log_env_status
from mcp_llm_server.domains.turtle_soup.models import (
    AnswerQuestionResponse,
    AnswerType,
    QuestionHistoryItem,
    format_answer_text,
)
from mcp_llm_server.domains.turtle_soup.prompts import get_turtle_soup_prompts
from mcp_llm_server.exceptions import (
    ErrorCode,
    ErrorContext,
    LLMModelError,
    LLMTimeoutError,
    MCPLLMError,
)
from mcp_llm_server.infra.bot_health_monitor import (
    BotHealthConfig,
    BotHealthMonitor,
)
from mcp_llm_server.infra.callbacks import get_callback_config
from mcp_llm_server.infra.gemini_client import GeminiClient, get_gemini_client
from mcp_llm_server.infra.injection_guard import get_injection_guard
from mcp_llm_server.infra.korean_nlp import get_korean_nlp_service
from mcp_llm_server.infra.langgraph_session import get_langgraph_manager
from mcp_llm_server.middleware import RequestIdMiddleware, get_request_id
from mcp_llm_server.models.error import (
    ErrorDetail,
    ErrorResponse,
    ValidationErrorResponse,
)
from mcp_llm_server.routes import guard as guard_routes
from mcp_llm_server.routes import health as health_routes
from mcp_llm_server.routes import llm as llm_routes
from mcp_llm_server.routes import nlp as nlp_routes
from mcp_llm_server.routes import session as session_routes
from mcp_llm_server.routes import usage as usage_routes
from mcp_llm_server.routes.guard import GuardRequest, GuardResponse
from mcp_llm_server.routes.llm import (
    ChatRequest,
    ChatResponse,
    ChatWithUsageRequest,
    ChatWithUsageResponse,
    StreamEventsRequest,
    StructuredRequest,
    UsageResponse,
)
from mcp_llm_server.routes.nlp import NlpRequest
from mcp_llm_server.routes.session import SessionChatRequest, SessionCreateRequest
from mcp_llm_server.routes.usage import (
    _build_daily_usage_response,
    _build_usage_list_response,
)
from mcp_llm_server.utils.session import resolve_session_id
from mcp_llm_server.utils.toon import encode_puzzle


__all__ = [
    "ChatRequest",
    "ChatResponse",
    "ChatWithUsageRequest",
    "ChatWithUsageResponse",
    "GuardRequest",
    "GuardResponse",
    "NlpRequest",
    "SessionChatRequest",
    "SessionCreateRequest",
    "StreamEventsRequest",
    "StructuredRequest",
    "UsageResponse",
    "_build_daily_usage_response",
    "_build_usage_list_response",
    "app",
    "lifespan",
    "main",
]


log = get_logger(__name__)


def _build_recent_qa_history_context(
    history: list["BaseMessage"],
    header: str,
    max_pairs: int,
) -> str:
    """최근 Q/A 히스토리를 프롬프트용 문자열로 구성."""
    history_lines: list[str] = []
    for msg in history:
        content = str(msg.content)
        if content.startswith("Q:") or content.startswith("A:"):
            history_lines.append(content)

    max_lines = max_pairs * 2
    if max_lines <= 0:
        return ""
    if len(history_lines) > max_lines:
        history_lines = history_lines[-max_lines:]
    if not history_lines:
        return ""

    return "\n\n" + header + "\n" + "\n".join(history_lines)


def _build_turtle_history_items(
    history: list["BaseMessage"],
    current_question: str,
    current_answer: str,
) -> list[QuestionHistoryItem]:
    """Turtle Soup용 Q/A 히스토리 아이템 구성."""
    history_items: list[QuestionHistoryItem] = []
    for i in range(0, len(history), 2):
        if i + 1 < len(history):
            q_msg = str(history[i].content)
            a_msg = str(history[i + 1].content)
            q_text = q_msg[3:] if q_msg.startswith("Q: ") else q_msg
            a_text = a_msg[3:] if a_msg.startswith("A: ") else a_msg
            history_items.append(QuestionHistoryItem(question=q_text, answer=a_text))
    history_items.append(
        QuestionHistoryItem(question=current_question, answer=current_answer)
    )
    return history_items


async def _invoke_llm_with_timeout[ResultT](
    client: GeminiClient,
    operation: Awaitable[ResultT],
    operation_name: str,
    session_id: str | None = None,
) -> ResultT:
    """LLM 호출에 공통 타임아웃과 에러 래핑을 적용."""
    timeout_seconds = get_settings().gemini.timeout

    try:
        async with asyncio.timeout(timeout_seconds):
            return await client._invoke_with_error_handling(operation)
    except TimeoutError as exc:
        raise LLMTimeoutError(
            message=f"{operation_name} timed out after {timeout_seconds}s",
            context=ErrorContext(
                request_id=get_request_id(),
                session_id=session_id,
                operation=operation_name,
                details={"timeout_seconds": timeout_seconds},
            ),
        ) from exc
    except MCPLLMError:
        raise
    except Exception as exc:
        raise LLMModelError(
            message=f"{operation_name} failed: {exc}",
            context=ErrorContext(
                request_id=get_request_id(),
                session_id=session_id,
                operation=operation_name,
            ),
        ) from exc


# =============================================================================
# TwentyQ Request/Response Models
# =============================================================================


class TwentyQHintsRequest(BaseModel):
    """Hint generation request"""

    target: str = Field(..., description="Secret answer (e.g., '스마트폰')")
    category: str = Field(..., description="Category (e.g., '사물', '음식')")
    details: dict[str, Any] | None = Field(None, description="Additional details")


class TwentyQAnswerRequest(BaseModel):
    """Answer question request"""

    session_id: str | None = Field(
        None, description="Session ID for history tracking (optional)"
    )
    chat_id: str | None = Field(
        None,
        description="Chat/room identifier for automatic session mapping (optional)",
    )
    namespace: str | None = Field(
        None, description="Bot namespace for automatic session mapping (optional)"
    )
    target: str = Field(..., description="Secret answer")
    category: str = Field(..., description="Category")
    question: str = Field(..., description="Player's yes/no question")
    details: dict[str, Any] | None = Field(None, description="Additional details")


class TwentyQVerifyRequest(BaseModel):
    """Verify guess request"""

    target: str = Field(..., description="Correct answer")
    guess: str = Field(..., description="Player's guess")


class TwentyQNormalizeRequest(BaseModel):
    """Normalize question request"""

    question: str = Field(..., description="Raw question from player")


class TwentyQSynonymRequest(BaseModel):
    """Synonym check request"""

    target: str = Field(..., description="Correct answer")
    guess: str = Field(..., description="Player's guess")


# =============================================================================
# Turtle Soup Request/Response Models
# =============================================================================


class TurtleAnswerRequest(BaseModel):
    """Turtle Soup question answer request"""

    session_id: str | None = Field(
        None, description="Session ID for history tracking (optional)"
    )
    chat_id: str | None = Field(
        None,
        description="Chat/room identifier for automatic session mapping (optional)",
    )
    namespace: str | None = Field(
        None, description="Bot namespace for automatic session mapping (optional)"
    )
    scenario: str = Field(..., description="The puzzle scenario")
    solution: str = Field(..., description="The hidden solution")
    question: str = Field(..., description="Player's yes/no question")


class TurtleHintRequest(BaseModel):
    """Turtle Soup hint generation request"""

    session_id: str | None = Field(None, description="Session ID (optional)")
    chat_id: str | None = Field(
        None,
        description="Chat/room identifier for automatic session mapping (optional)",
    )
    namespace: str | None = Field(
        None, description="Bot namespace for automatic session mapping (optional)"
    )
    scenario: str = Field(..., description="The puzzle scenario")
    solution: str = Field(..., description="The hidden solution")
    level: int = Field(..., ge=1, le=3, description="Hint level (1-3)")


class TurtleValidateRequest(BaseModel):
    """Turtle Soup solution validation request"""

    session_id: str | None = Field(None, description="Session ID (optional)")
    chat_id: str | None = Field(
        None,
        description="Chat/room identifier for automatic session mapping (optional)",
    )
    namespace: str | None = Field(
        None, description="Bot namespace for automatic session mapping (optional)"
    )
    solution: str = Field(..., description="The correct solution")
    player_answer: str = Field(..., description="Player's submitted answer")


class TurtleRevealRequest(BaseModel):
    """Turtle Soup solution reveal request"""

    session_id: str | None = Field(None, description="Session ID (optional)")
    chat_id: str | None = Field(
        None,
        description="Chat/room identifier for automatic session mapping (optional)",
    )
    namespace: str | None = Field(
        None, description="Bot namespace for automatic session mapping (optional)"
    )
    scenario: str = Field(..., description="The puzzle scenario")
    solution: str = Field(..., description="The solution to reveal")


class TurtleGenerateRequest(BaseModel):
    """Turtle Soup puzzle generation request"""

    category: str = Field(
        ..., description="Puzzle category (MYSTERY, HORROR, ABSURD, LOGIC)"
    )
    difficulty: int = Field(..., ge=1, le=5, description="Difficulty level (1-5)")
    theme: str = Field("", description="Optional theme/topic hint")


class TurtleRewriteRequest(BaseModel):
    """Turtle Soup scenario rewrite request"""

    title: str = Field(..., description="Puzzle title")
    scenario: str = Field(..., description="Original scenario")
    solution: str = Field(..., description="The solution to rewrite")
    difficulty: int = Field(..., ge=1, le=5, description="Difficulty level")


# =============================================================================
# Lifespan Management
# =============================================================================


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncIterator[None]:
    """Initialize services on startup, cleanup on shutdown"""
    setup_logging()
    log_env_status(get_settings())

    # Initialize guard with NLP
    guard = get_injection_guard()
    nlp = get_korean_nlp_service()
    guard.set_anomaly_scorer(nlp.calculate_anomaly_score_async)

    # Initialize LangGraph with Redis checkpointer
    from mcp_llm_server.infra.langgraph_session import (
        init_langgraph_with_redis,
        shutdown_langgraph,
    )

    await init_langgraph_with_redis()

    # Initialize UsageRepository and inject into MetricsCallbackHandler
    from mcp_llm_server.infra.callbacks import get_metrics_handler
    from mcp_llm_server.infra.usage_repository import get_usage_repository

    repository = get_usage_repository()
    metrics_handler = get_metrics_handler()
    metrics_handler.set_repository(repository)
    log.info("UsageRepository connected to MetricsCallbackHandler")

    bot_monitor = BotHealthMonitor(BotHealthConfig.from_env())
    await bot_monitor.start()

    try:
        yield
    finally:
        await bot_monitor.stop()
        # Cleanup on shutdown
        await shutdown_langgraph()
        await repository.close()


# =============================================================================
# App Setup
# =============================================================================

app = FastAPI(
    title="MCP LLM Server",
    description="REST API for LLM operations (external testing)",
    version="0.1.0",
    lifespan=lifespan,
    default_response_class=ORJSONResponse,
)

# Request ID 미들웨어 추가
app.add_middleware(RequestIdMiddleware)

# Routers
app.include_router(llm_routes.router)
app.include_router(session_routes.router)
app.include_router(guard_routes.router)
app.include_router(nlp_routes.router)
app.include_router(usage_routes.router)
app.include_router(health_routes.router)


# =============================================================================
# Exception Handlers
# =============================================================================


@app.exception_handler(MCPLLMError)
async def mcp_llm_error_handler(request: Request, exc: MCPLLMError) -> ORJSONResponse:
    """MCPLLMError 계층의 모든 예외 처리.

    커스텀 예외를 표준 ErrorResponse 포맷으로 변환.
    """
    # request_id 추가
    exc.context.request_id = get_request_id()

    log.warning(
        "EXCEPTION error_code={} type={} message={} request_id={}",
        exc.error_code.value,
        exc.__class__.__name__,
        exc.message,
        exc.context.request_id,
    )

    return ORJSONResponse(
        status_code=exc.status_code,
        content=ErrorResponse(
            error_code=exc.error_code.value,
            error_type=exc.__class__.__name__,
            message=exc.message,
            request_id=exc.context.request_id,
            details=exc.context.details,
        ).model_dump(),
        headers={"X-Request-ID": exc.context.request_id or ""},
    )


@app.exception_handler(RequestValidationError)
async def validation_error_handler(
    request: Request, exc: RequestValidationError
) -> ORJSONResponse:
    """Pydantic validation 에러 처리.

    필드별 에러 목록을 포함한 ValidationErrorResponse 반환.
    """
    request_id = get_request_id()

    errors = [
        ErrorDetail(
            field=".".join(str(loc) for loc in err["loc"]),
            message=err["msg"],
            value=err.get("input"),
        )
        for err in exc.errors()
    ]

    log.warning(
        "VALIDATION_ERROR fields={} request_id={}",
        [e.field for e in errors],
        request_id,
    )

    return ORJSONResponse(
        status_code=422,
        content=ValidationErrorResponse(
            error_code=ErrorCode.VALIDATION_ERROR.value,
            error_type="ValidationError",
            message="Input validation failed",
            request_id=request_id,
            details=None,
            errors=errors,
        ).model_dump(),
        headers={"X-Request-ID": request_id or ""},
    )


@app.exception_handler(Exception)
async def generic_error_handler(request: Request, exc: Exception) -> ORJSONResponse:
    """예상치 못한 예외 처리.

    모든 unhandled exception을 INTERNAL_ERROR로 변환.
    스택 트레이스는 로그에만 기록.
    """
    request_id = get_request_id()

    log.exception(
        "UNHANDLED_EXCEPTION type={} message={} request_id={}",
        exc.__class__.__name__,
        str(exc),
        request_id,
    )

    return ORJSONResponse(
        status_code=500,
        content=ErrorResponse(
            error_code=ErrorCode.INTERNAL_ERROR.value,
            error_type=exc.__class__.__name__,
            message="Internal server error",
            request_id=request_id,
            details=None,
        ).model_dump(),
        headers={"X-Request-ID": request_id or ""},
    )


# =============================================================================
# REST API - TwentyQ (Domain-Specific)
# =============================================================================


@app.post("/api/twentyq/hints", tags=["TwentyQ"])
async def api_twentyq_hints(request: TwentyQHintsRequest) -> dict[str, Any]:
    """Generate hints for Twenty Questions game

    Returns:
        dict with 'hints' list and optional 'thought_signature'
    """

    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.domains.twentyq.models import HintsOutput, HintsResponse
    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts
    from mcp_llm_server.utils.toon import encode_secret

    client = get_gemini_client()
    prompts = get_twentyq_prompts()

    secret_toon = encode_secret(request.target, request.category)
    system = prompts.hints_system(request.category)
    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{secret_info}"),
        ]
    )

    llm = client.get_llm_for_task("hints")
    structured_llm = llm.with_structured_output(HintsOutput)
    chain = prompt | structured_llm

    user_content = prompts.hints_user(secret_toon)
    result = cast(
        "HintsOutput",
        await _invoke_llm_with_timeout(
            client,
            chain.ainvoke({"secret_info": user_content}, config=get_callback_config()),
            "twentyq.hints",
        ),
    )

    response = HintsResponse(hints=result.hints, thought_signature=None)
    return response.model_dump()


@app.post("/api/twentyq/answers", tags=["TwentyQ"])
async def api_twentyq_answer(request: TwentyQAnswerRequest) -> dict[str, Any]:
    """Answer a yes/no question for Twenty Questions game

    Returns:
        dict with 'scale', 'raw_text', 'thought_signature'
    """
    from langchain_core.messages import AIMessage, HumanMessage
    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.domains.twentyq.models import AnswerResponse, AnswerScale
    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts
    from mcp_llm_server.utils.toon import encode_secret

    client = get_gemini_client()
    prompts = get_twentyq_prompts()
    manager = await get_langgraph_manager()

    session_id = resolve_session_id(
        session_id=request.session_id,
        chat_id=request.chat_id,
        namespace=request.namespace,
        default_namespace="twentyq",
    )
    if session_id and request.session_id is None:
        settings = get_settings()
        await manager.create_session(
            session_id=session_id,
            model=settings.gemini.default_model,
        )

    secret_toon = encode_secret(request.target, request.category)

    # session_id가 있으면 히스토리 사용, 없으면 stateless
    history_context = ""
    history_count = 0
    if session_id:
        history = await manager.get_history(session_id)
        history_context = _build_recent_qa_history_context(
            history,
            header=f"[이전 질문/답변 기록 - 정답: {request.target}]",
            max_pairs=get_settings().session.history_max_pairs,
        )
        history_count = len(history)

    log.info(
        "TWENTYQ_ANSWER session={}, count={}, q={}",
        session_id or "stateless",
        history_count,
        request.question[:30],
    )

    system = prompts.answer_system()
    user_content = prompts.answer_user(secret_toon, request.question, history_context)

    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    llm = client.get_llm_for_task("answer")
    chain = prompt | llm
    result = await _invoke_llm_with_timeout(
        client,
        chain.ainvoke({"user_prompt": user_content}, config=get_callback_config()),
        "twentyq.answer",
        session_id,
    )

    raw_text = client.extract_text(result)
    scale = AnswerScale.from_text(raw_text)

    # Retry if parsing failed
    if scale is None:
        retry_prompt = (
            user_content
            + "\n\n반드시 다음 중 하나만 출력: 예 | 아마도 예 | 아마도 아니오 | 아니오"
        )
        result = await _invoke_llm_with_timeout(
            client,
            chain.ainvoke({"user_prompt": retry_prompt}, config=get_callback_config()),
            "twentyq.answer.retry",
            session_id,
        )
        raw_text = client.extract_text(result)
        scale = AnswerScale.from_text(raw_text)

    # Save to history (only if session_id provided)
    scale_text = scale.value if scale else "UNKNOWN"
    if session_id:
        await manager.add_messages(
            session_id,
            [
                HumanMessage(content=f"Q: {request.question}"),
                AIMessage(content=f"A: {scale_text}"),
            ],
        )

    response = AnswerResponse(
        scale=scale.value if scale else None,
        raw_text=raw_text,
        thought_signature=None,
    )
    return response.model_dump()


@app.post("/api/twentyq/verifications", tags=["TwentyQ"])
async def api_twentyq_verify(request: TwentyQVerifyRequest) -> dict[str, Any]:
    """Verify if a guess matches the target answer

    Returns:
        dict with 'result' (ACCEPT/CLOSE/REJECT), 'raw_text'
    """
    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.domains.twentyq.models import (
        VerifyOutput,
        VerifyResponse,
        VerifyResult,
    )
    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts

    client = get_gemini_client()
    prompts = get_twentyq_prompts()

    system = prompts.verify_system()
    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    llm = client.get_llm_for_task("verify")
    structured_llm = llm.with_structured_output(VerifyOutput)
    chain = prompt | structured_llm

    user_content = prompts.verify_user(request.target, request.guess)
    try:
        result = cast(
            "VerifyOutput",
            await _invoke_llm_with_timeout(
                client,
                chain.ainvoke(
                    {"user_prompt": user_content}, config=get_callback_config()
                ),
                "twentyq.verify",
            ),
        )
        response = VerifyResponse(
            result=result.result.value, raw_text=result.result.value
        )
    except Exception:  # noqa: BLE001 fallback to plain parse
        # Fallback: direct parsing
        plain_chain = prompt | llm
        plain_result = await _invoke_llm_with_timeout(
            client,
            plain_chain.ainvoke(
                {"user_prompt": user_content}, config=get_callback_config()
            ),
            "twentyq.verify.fallback",
        )
        raw = client.extract_text(plain_result).upper()
        parsed = None
        for r in VerifyResult:
            if r.value in raw:
                parsed = r.value
                break
        response = VerifyResponse(result=parsed, raw_text=raw)

    return response.model_dump()


@app.post("/api/twentyq/normalizations", tags=["TwentyQ"])
async def api_twentyq_normalize(request: TwentyQNormalizeRequest) -> dict[str, Any]:
    """Normalize a player's question (fix typos, standardize format)

    Returns:
        dict with 'normalized' text and 'original' text
    """
    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.domains.twentyq.models import NormalizeOutput, NormalizeResponse
    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts

    client = get_gemini_client()
    prompts = get_twentyq_prompts()

    system = prompts.normalize_system()
    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{question}"),
        ]
    )

    llm = client.get_llm_for_task()
    structured_llm = llm.with_structured_output(NormalizeOutput)
    chain = prompt | structured_llm

    user_content = prompts.normalize_user(request.question)
    try:
        result = cast(
            "NormalizeOutput",
            await _invoke_llm_with_timeout(
                client,
                chain.ainvoke({"question": user_content}, config=get_callback_config()),
                "twentyq.normalize",
            ),
        )
        normalized = result.normalized
    except Exception:  # noqa: BLE001 fallback to original question
        normalized = request.question

    response = NormalizeResponse(normalized=normalized, original=request.question)
    return response.model_dump()


@app.post("/api/twentyq/synonym-checks", tags=["TwentyQ"])
async def api_twentyq_synonym(request: TwentyQSynonymRequest) -> dict[str, Any]:
    """Check if target and guess are semantic equivalents

    Returns:
        dict with 'result' (EQUIVALENT/NOT_EQUIVALENT), 'raw_text'
    """
    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.domains.twentyq.models import (
        SynonymOutput,
        SynonymResponse,
        SynonymResult,
    )
    from mcp_llm_server.domains.twentyq.prompts import get_twentyq_prompts

    client = get_gemini_client()
    prompts = get_twentyq_prompts()

    system = prompts.synonym_system()
    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    llm = client.get_llm_for_task()
    structured_llm = llm.with_structured_output(SynonymOutput)
    chain = prompt | structured_llm

    user_content = prompts.synonym_user(request.target, request.guess)
    try:
        result = cast(
            "SynonymOutput",
            await _invoke_llm_with_timeout(
                client,
                chain.ainvoke(
                    {"user_prompt": user_content}, config=get_callback_config()
                ),
                "twentyq.synonym",
            ),
        )
        response = SynonymResponse(
            result=result.result.value, raw_text=result.result.value
        )
    except Exception:  # noqa: BLE001 fallback to plain chain
        plain_chain = prompt | llm
        plain_result = await _invoke_llm_with_timeout(
            client,
            plain_chain.ainvoke(
                {"user_prompt": user_content}, config=get_callback_config()
            ),
            "twentyq.synonym.fallback",
        )
        raw = client.extract_text(plain_result).upper()
        parsed = None
        for r in SynonymResult:
            if r.value in raw:
                parsed = r.value
                break
        response = SynonymResponse(result=parsed, raw_text=raw)

    return response.model_dump()


# =============================================================================
# REST API - Turtle Soup (Domain-Specific)
# =============================================================================


@app.post("/api/turtle-soup/answers", tags=["TurtleSoup"])
async def api_turtle_answer(request: TurtleAnswerRequest) -> dict[str, Any]:
    """Answer a player's yes/no question for Turtle Soup game

    Returns:
        dict with 'answer', 'raw_text', 'question_count', 'history'
    """

    client = get_gemini_client()
    prompts = get_turtle_soup_prompts()
    manager = await get_langgraph_manager()

    # session_id가 있으면 히스토리 사용, 없으면 stateless
    history_context = ""
    history: list[BaseMessage] = []
    session_id = resolve_session_id(
        session_id=request.session_id,
        chat_id=request.chat_id,
        namespace=request.namespace,
        default_namespace="turtle-soup",
    )
    if session_id and request.session_id is None:
        settings = get_settings()
        await manager.create_session(
            session_id=session_id,
            model=settings.gemini.default_model,
        )

    if session_id:
        history = await manager.get_history(session_id)
        history_context = _build_recent_qa_history_context(
            history,
            header="[이전 질문/답변 기록]",
            max_pairs=get_settings().session.history_max_pairs,
        )

    log.info(
        "TURTLE_ANSWER session={}, count={}",
        session_id or "stateless",
        len(history),
    )

    # TOON 형식으로 puzzle 인코딩
    puzzle_toon = encode_puzzle(request.scenario, request.solution)

    system = prompts.answer_system()
    user_content = prompts.answer_user(puzzle_toon, request.question, history_context)

    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    llm = client.get_llm_for_task("answer")
    chain = prompt | llm

    result = await _invoke_llm_with_timeout(
        client,
        chain.ainvoke({"user_prompt": user_content}, config=get_callback_config()),
        "turtle_soup.answer",
        session_id,
    )
    raw_text = client.extract_text(result)

    normalized_text = raw_text.replace(" ", "")
    is_important = (
        "중요한질문입니다" in normalized_text or "중요합니다" in normalized_text
    )

    answer = AnswerType.from_text(raw_text)
    if answer == AnswerType.IMPORTANT:
        answer = AnswerType.YES
    if is_important and answer is None:
        answer = AnswerType.YES
    answer_text = format_answer_text(
        answer=answer,
        is_important=is_important,
        raw_text=raw_text,
    )

    # LangGraph에 Q&A 저장 (only if session_id provided)
    if session_id:
        await manager.add_messages(
            session_id,
            [
                HumanMessage(content=f"Q: {request.question}"),
                AIMessage(content=f"A: {answer_text}"),
            ],
        )

    # Q&A count
    question_count = len(history) // 2 + 1

    # 히스토리를 QuestionHistoryItem 리스트로 변환
    history_items = _build_turtle_history_items(history, request.question, answer_text)

    response = AnswerQuestionResponse(
        answer=answer_text,
        raw_text=raw_text,
        question_count=question_count,
        history=history_items,
    )
    return response.model_dump()


@app.post("/api/turtle-soup/hints", tags=["TurtleSoup"])
async def api_turtle_hint(request: TurtleHintRequest) -> dict[str, Any]:
    """Generate a hint for Turtle Soup game

    Returns:
        dict with 'hint' text and 'level'
    """
    import json

    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.domains.turtle_soup.models import HintOutput, HintResponse
    from mcp_llm_server.domains.turtle_soup.prompts import get_turtle_soup_prompts
    from mcp_llm_server.utils.toon import encode_puzzle

    client = get_gemini_client()
    prompts = get_turtle_soup_prompts()

    puzzle_toon = encode_puzzle(request.scenario, request.solution)

    system = prompts.hint_system()
    user_content = prompts.hint_user(puzzle_toon, request.level)

    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    llm = client.get_llm_for_task("hints")

    try:
        structured_llm = llm.with_structured_output(HintOutput)
        chain = prompt | structured_llm
        result = cast(
            "HintOutput",
            await _invoke_llm_with_timeout(
                client,
                chain.ainvoke(
                    {"user_prompt": user_content}, config=get_callback_config()
                ),
                "turtle_soup.hints",
                request.session_id,
            ),
        )
        hint = result.hint
    except Exception as e:  # noqa: BLE001 fallback to plain chain
        log.warning("TURTLE_HINT_FALLBACK error={}", e)
        chain = prompt | llm
        raw_result = cast(
            "BaseMessage",
            await _invoke_llm_with_timeout(
                client,
                chain.ainvoke(
                    {"user_prompt": user_content}, config=get_callback_config()
                ),
                "turtle_soup.hints_fallback",
                request.session_id,
            ),
        )
        raw_text = client.extract_text(raw_result).strip()
        if raw_text.startswith("```"):
            raw_text = raw_text.split("```")[1] if "```" in raw_text else raw_text
        try:
            parsed = json.loads(raw_text)
            hint = parsed.get("hint", raw_text)
        except json.JSONDecodeError:
            hint = raw_text

    response = HintResponse(hint=hint, level=request.level)
    return response.model_dump()


@app.post("/api/turtle-soup/validations", tags=["TurtleSoup"])
async def api_turtle_validate(request: TurtleValidateRequest) -> dict[str, Any]:
    """Validate if player's answer matches the solution

    Returns:
        dict with 'result' (YES/NO/CLOSE), 'raw_text'
    """
    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.domains.turtle_soup.models import (
        ValidationResponse,
        ValidationResult,
    )
    from mcp_llm_server.domains.turtle_soup.prompts import get_turtle_soup_prompts

    client = get_gemini_client()
    prompts = get_turtle_soup_prompts()

    system = prompts.validate_system()
    user_content = prompts.validate_user(request.solution, request.player_answer)

    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    llm = client.get_llm_for_task("verify")
    chain = prompt | llm

    result = await _invoke_llm_with_timeout(
        client,
        chain.ainvoke({"user_prompt": user_content}, config=get_callback_config()),
        "turtle_soup.validate",
        request.session_id,
    )
    raw_text = client.extract_text(result)

    validation = ValidationResult.from_text(raw_text)

    response = ValidationResponse(
        result=validation.value if validation else raw_text,
        raw_text=raw_text,
    )
    return response.model_dump()


@app.post("/api/turtle-soup/reveals", tags=["TurtleSoup"])
async def api_turtle_reveal(request: TurtleRevealRequest) -> dict[str, Any]:
    """Reveal the solution with dramatic narrative

    Returns:
        dict with 'narrative' (dramatic Korean explanation)
    """
    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.domains.turtle_soup.models import RevealResponse
    from mcp_llm_server.domains.turtle_soup.prompts import get_turtle_soup_prompts
    from mcp_llm_server.utils.toon import encode_puzzle

    client = get_gemini_client()
    prompts = get_turtle_soup_prompts()

    puzzle_toon = encode_puzzle(request.scenario, request.solution)

    system = prompts.reveal_system()
    user_content = prompts.reveal_user(puzzle_toon)

    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    llm = client.get_llm_for_task()
    chain = prompt | llm

    result = await _invoke_llm_with_timeout(
        client,
        chain.ainvoke({"user_prompt": user_content}, config=get_callback_config()),
        "turtle_soup.reveal",
        request.session_id,
    )
    narrative = client.extract_text(result)

    response = RevealResponse(narrative=narrative)
    return response.model_dump()


@app.post("/api/turtle-soup/puzzles", tags=["TurtleSoup"])
async def api_turtle_generate(request: TurtleGenerateRequest) -> dict[str, Any]:
    """Generate a new lateral thinking puzzle

    Returns:
        dict with 'title', 'scenario', 'solution', 'category', 'difficulty', 'hints'
    """
    import json

    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.domains.turtle_soup.models import (
        GeneratePuzzleResponse,
        PuzzleOutput,
    )
    from mcp_llm_server.domains.turtle_soup.prompts import get_turtle_soup_prompts
    from mcp_llm_server.domains.turtle_soup.puzzle_loader import get_puzzle_loader

    client = get_gemini_client()
    prompts = get_turtle_soup_prompts()
    loader = get_puzzle_loader()

    # 1) 프리셋 JSON 퍼즐 우선 반환
    try:
        preset = loader.get_random_puzzle_by_difficulty(request.difficulty)
        log.info(
            "TURTLE_GENERATE_PRESET_FOUND id=%s difficulty=%s",
            preset.id,
            preset.difficulty,
        )
        return GeneratePuzzleResponse(
            title=preset.title,
            scenario=preset.question,
            solution=preset.answer,
            category=request.category or "PRESET",
            difficulty=preset.difficulty,
            hints=[],
            puzzle_id=preset.id,
        ).model_dump()
    except ValueError:
        log.info(
            "TURTLE_GENERATE_PRESET_MISS difficulty=%s -> fallback_to_llm",
            request.difficulty,
        )

    # 2) 프리셋이 없을 때에만 LLM 생성 (기존 로직)
    examples = loader.get_examples(difficulty=request.difficulty, max_examples=3)
    example_lines = [
        "\n".join(
            [
                f"- 제목: {puzzle.title}",
                f"  시나리오: {puzzle.question}",
                f"  정답: {puzzle.answer}",
                f"  난이도: {puzzle.difficulty}",
            ]
        )
        for puzzle in examples
    ]
    examples_block = "\n\n".join(example_lines)

    system = prompts.generate_system()
    user_content = prompts.generate_user(
        request.category, request.difficulty, request.theme, examples_block
    )

    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    llm = client.get_llm_for_task("hints")

    try:
        structured_llm = llm.with_structured_output(PuzzleOutput)
        chain = prompt | structured_llm
        result = cast(
            "PuzzleOutput",
            await _invoke_llm_with_timeout(
                client,
                chain.ainvoke(
                    {"user_prompt": user_content}, config=get_callback_config()
                ),
                "turtle_soup.generate",
            ),
        )

        response = GeneratePuzzleResponse(
            title=result.title,
            scenario=result.scenario,
            solution=result.solution,
            category=request.category,
            difficulty=request.difficulty,
            hints=result.hints,
            puzzle_id=None,
        )
    except Exception as e:  # noqa: BLE001 fallback to plain chain
        log.warning("TURTLE_GENERATE_FALLBACK error={}", e)
        chain = prompt | llm
        raw_result = cast(
            "BaseMessage",
            await _invoke_llm_with_timeout(
                client,
                chain.ainvoke(
                    {"user_prompt": user_content}, config=get_callback_config()
                ),
                "turtle_soup.generate_fallback",
            ),
        )
        raw_text = client.extract_text(raw_result).strip()
        if raw_text.startswith("```"):
            raw_text = (
                raw_text.split("```json")[-1].split("```")[0]
                if "```" in raw_text
                else raw_text
            )
        parsed = json.loads(raw_text)

        response = GeneratePuzzleResponse(
            title=parsed.get("title", "무제"),
            scenario=parsed.get("scenario", ""),
            solution=parsed.get("solution", ""),
            category=request.category,
            difficulty=request.difficulty,
            hints=parsed.get("hints", []),
            puzzle_id=None,
        )

    return response.model_dump()


@app.post("/api/turtle-soup/rewrites", tags=["TurtleSoup"])
async def api_turtle_rewrite(request: TurtleRewriteRequest) -> dict[str, Any]:
    """Rewrite a puzzle scenario and solution while preserving the core logic

    Returns:
        dict with 'scenario', 'solution', 'original_scenario', 'original_solution'
    """
    import json

    from langchain_core.prompts import ChatPromptTemplate

    from mcp_llm_server.domains.turtle_soup.models import (
        RewriteOutput,
        RewriteScenarioResponse,
    )
    from mcp_llm_server.domains.turtle_soup.prompts import get_turtle_soup_prompts

    client = get_gemini_client()
    prompts = get_turtle_soup_prompts()

    system = prompts.rewrite_system()
    user_content = prompts.rewrite_user(
        request.title, request.scenario, request.solution, request.difficulty
    )

    prompt = ChatPromptTemplate.from_messages(
        [
            ("system", system),
            ("human", "{user_prompt}"),
        ]
    )

    llm = client.get_llm_for_task()

    try:
        structured_llm = llm.with_structured_output(RewriteOutput)
        chain = prompt | structured_llm
        result = cast(
            "RewriteOutput",
            await _invoke_llm_with_timeout(
                client,
                chain.ainvoke(
                    {"user_prompt": user_content}, config=get_callback_config()
                ),
                "turtle_soup.rewrite",
            ),
        )

        response = RewriteScenarioResponse(
            scenario=result.scenario.strip(),
            solution=result.solution.strip(),
            original_scenario=request.scenario,
            original_solution=request.solution,
        )
    except Exception as e:  # noqa: BLE001 fallback to plain chain
        log.warning("TURTLE_REWRITE_FALLBACK error={}", e)
        chain = prompt | llm
        raw_result = cast(
            "BaseMessage",
            await _invoke_llm_with_timeout(
                client,
                chain.ainvoke(
                    {"user_prompt": user_content}, config=get_callback_config()
                ),
                "turtle_soup.rewrite_fallback",
            ),
        )
        raw_text = client.extract_text(raw_result).strip()
        if raw_text.startswith("```"):
            raw_text = (
                raw_text.split("```json")[-1].split("```")[0]
                if "```" in raw_text
                else raw_text
            )
        try:
            parsed = json.loads(raw_text)
            response = RewriteScenarioResponse(
                scenario=parsed.get("scenario", request.scenario).strip(),
                solution=parsed.get("solution", request.solution).strip(),
                original_scenario=request.scenario,
                original_solution=request.solution,
            )
        except json.JSONDecodeError:
            response = RewriteScenarioResponse(
                scenario=request.scenario,
                solution=request.solution,
                original_scenario=request.scenario,
                original_solution=request.solution,
            )

    return response.model_dump()


# === Turtle Soup Puzzle APIs ===


@app.get("/api/turtle-soup/puzzles", tags=["TurtleSoup"])
async def api_turtle_puzzles() -> dict[str, Any]:
    """Get all puzzles with statistics.

    Returns:
        dict with 'puzzles' list and 'stats'
    """
    from mcp_llm_server.domains.turtle_soup.puzzle_loader import get_puzzle_loader

    loader = get_puzzle_loader()
    puzzles = loader.get_all_puzzles()

    return {
        "puzzles": [p.model_dump() for p in puzzles],
        "stats": {
            "total": len(puzzles),
            "by_difficulty": loader.get_puzzle_count_by_difficulty(),
        },
    }


@app.get("/api/turtle-soup/puzzles/random", tags=["TurtleSoup"])
async def api_turtle_puzzle_random(difficulty: int | None = None) -> dict[str, Any]:
    """Get a random puzzle.

    Args:
        difficulty: Optional difficulty filter (1-5)

    Returns:
        Random puzzle data
    """
    from mcp_llm_server.domains.turtle_soup.puzzle_loader import get_puzzle_loader

    loader = get_puzzle_loader()

    if difficulty is not None:
        puzzle = loader.get_random_puzzle_by_difficulty(difficulty)
    else:
        puzzle = loader.get_random_puzzle()

    return puzzle.model_dump()


@app.get("/api/turtle-soup/puzzles/{puzzle_id}", tags=["TurtleSoup"])
async def api_turtle_puzzle_by_id(puzzle_id: int) -> dict[str, Any]:
    """Get puzzle by ID.

    Args:
        puzzle_id: Puzzle ID

    Returns:
        Puzzle data

    Raises:
        HTTPException: If puzzle not found
    """
    from fastapi import HTTPException

    from mcp_llm_server.domains.turtle_soup.puzzle_loader import get_puzzle_loader

    loader = get_puzzle_loader()
    puzzle = loader.get_puzzle_by_id(puzzle_id)

    if puzzle is None:
        raise HTTPException(status_code=404, detail=f"Puzzle {puzzle_id} not found")

    return puzzle.model_dump()


@app.post("/api/turtle-soup/puzzles/reload", tags=["TurtleSoup"])
async def api_turtle_puzzles_reload() -> dict[str, Any]:
    """Reload puzzles from files (hot reload).

    Returns:
        Reload result with count
    """
    from mcp_llm_server.domains.turtle_soup.puzzle_loader import get_puzzle_loader

    loader = get_puzzle_loader()
    count = loader.reload()

    return {
        "success": True,
        "count": count,
        "by_difficulty": loader.get_puzzle_count_by_difficulty(),
    }


# =============================================================================
# Entry Point
# =============================================================================


def main() -> None:
    """Run HTTP server"""
    settings = get_settings()

    hyper_config = HyperConfig()
    hyper_config.bind = [f"{settings.http.host}:{settings.http.port}"]
    reload_enabled = os.getenv("HTTP_RELOAD", "false").lower() == "true"
    hyper_config.use_reloader = reload_enabled
    hyper_config.alpn_protocols = (
        ["h2", "http/1.1"] if settings.http.http2_enabled else ["http/1.1"]
    )
    log.info(
        "HTTP_SERVER_START mode=h2c host={} port={} http2={}",
        settings.http.host,
        settings.http.port,
        settings.http.http2_enabled,
    )

    asyncio.run(serve(cast("ASGIFramework", app), hyper_config))


if __name__ == "__main__":
    main()
