"""Tests for HTTP server endpoints."""

from collections.abc import Generator
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
from fastapi.testclient import TestClient


# Fixtures need to mock dependencies before importing app
@pytest.fixture
def mock_gemini_client() -> Generator[MagicMock]:
    """Mock Gemini client."""
    with patch("mcp_llm_server.routes.llm.get_gemini_client") as mock:
        client = MagicMock()
        client.chat = AsyncMock(return_value="Test response")
        mock.return_value = client
        yield client


@pytest.fixture
def mock_langgraph_manager() -> Generator[MagicMock]:
    """Mock LangGraph session manager."""
    with patch("mcp_llm_server.routes.session.get_langgraph_manager") as mock:
        manager = MagicMock()
        manager.get_or_create_session = AsyncMock()
        manager.get_session = AsyncMock()
        manager.end_session = AsyncMock(return_value=True)
        manager.get_session_info = MagicMock()
        manager.get_history = AsyncMock(return_value=[])
        manager.add_message = AsyncMock()
        manager.add_messages = AsyncMock()
        manager.get_history_as_dicts = AsyncMock(return_value=[])
        manager.clear_history = AsyncMock()
        mock.return_value = manager
        yield manager


@pytest.fixture
def mock_guard() -> Generator[MagicMock]:
    """Mock injection guard."""
    with (
        patch("mcp_llm_server.http_server.get_injection_guard") as mock_guard_http,
        patch(
            "mcp_llm_server.routes.dependencies.get_injection_guard"
        ) as mock_guard_dep,
        patch("mcp_llm_server.routes.guard.get_injection_guard") as mock_guard_route,
    ):
        guard = MagicMock()
        evaluation = SimpleNamespace(score=0.1, threshold=1.0, malicious=False, hits=[])
        guard.evaluate = AsyncMock(return_value=evaluation)
        guard.is_malicious = AsyncMock(return_value=False)
        mock_guard_http.return_value = guard
        mock_guard_dep.return_value = guard
        mock_guard_route.return_value = guard
        yield guard


@pytest.fixture
def mock_nlp() -> Generator[MagicMock]:
    """Mock Korean NLP service."""
    with (
        patch("mcp_llm_server.http_server.get_korean_nlp_service") as mock_http,
        patch("mcp_llm_server.routes.nlp.get_korean_nlp_service") as mock_route,
    ):
        nlp = MagicMock()
        nlp.analyze_async = AsyncMock()
        nlp.calculate_anomaly_score_async = AsyncMock()
        nlp.analyze_heuristics_async = AsyncMock()
        mock_http.return_value = nlp
        mock_route.return_value = nlp
        yield nlp


@pytest.fixture
def mock_metrics_handler() -> Generator[MagicMock]:
    """Mock metrics handler."""
    with patch("mcp_llm_server.routes.llm.get_metrics_handler") as mock:
        handler = MagicMock()
        handler.get_metrics.return_value = {
            "total_calls": 10,
            "total_input_tokens": 100,
            "total_output_tokens": 50,
            "total_tokens": 150,
            "total_reasoning_tokens": 0,
            "total_duration_ms": 1000,
        }
        mock.return_value = handler
        yield handler


@pytest.fixture
def client(
    mock_gemini_client: MagicMock,
    mock_langgraph_manager: MagicMock,
    mock_guard: MagicMock,
    mock_nlp: MagicMock,
    mock_metrics_handler: MagicMock,
) -> TestClient:
    """Create test client with mocked dependencies."""
    from mcp_llm_server.http_server import app

    return TestClient(app)


class TestHealthEndpoint:
    """Tests for /health endpoint."""

    def test_health_check_basic(self, client: TestClient) -> None:
        with patch(
            "mcp_llm_server.routes.health.collect_health",
            AsyncMock(return_value={"status": "ok", "components": {}}),
        ) as mock_health:
            response = client.get("/health")

        mock_health.assert_awaited_once_with(deep_checks=True)
        assert response.status_code == 200
        assert response.json()["status"] == "ok"

    def test_health_includes_langgraph(self, client: TestClient) -> None:
        health_payload = {
            "status": "ok",
            "components": {
                "langgraph": {
                    "status": "ok",
                    "detail": {
                        "redis_enabled": True,
                        "redis_connected": True,
                        "backend": "redis",
                        "session_count": 3,
                        "redis_url": "redis://localhost:46379",
                        "session_ttl_minutes": 60,
                        "deep_checked": True,
                    },
                }
            },
            "langgraph": {
                "redis_enabled": True,
                "redis_connected": True,
                "backend": "redis",
                "session_count": 3,
                "redis_url": "redis://localhost:46379",
                "session_ttl_minutes": 60,
                "deep_checked": True,
            },
        }

        with patch(
            "mcp_llm_server.routes.health.collect_health",
            AsyncMock(return_value=health_payload),
        ) as mock_health:
            response = client.get("/health/ready")

        mock_health.assert_awaited_once_with(deep_checks=True)
        assert response.status_code == 200
        data = response.json()
        assert data["langgraph"]["backend"] == "redis"
        assert data["langgraph"]["session_count"] == 3
        assert data["components"]["langgraph"]["status"] == "ok"
        assert data["components"]["langgraph"]["detail"]["deep_checked"] is True

    def test_health_live_shallow_check(self, client: TestClient) -> None:
        with patch(
            "mcp_llm_server.routes.health.collect_health",
            AsyncMock(return_value={"status": "ok", "components": {}}),
        ) as mock_health:
            response = client.get("/health/live")

        mock_health.assert_awaited_once_with(deep_checks=False)
        assert response.status_code == 200


class TestLlmChatEndpoint:
    """Tests for /api/llm/chat endpoint."""

    def test_chat_basic(
        self, client: TestClient, mock_gemini_client: MagicMock
    ) -> None:
        mock_gemini_client.chat = AsyncMock(return_value="Hello!")

        response = client.post("/api/llm/chat", json={"prompt": "Hi"})

        assert response.status_code == 200
        assert response.json()["response"] == "Hello!"

    def test_chat_with_system_prompt(
        self, client: TestClient, mock_gemini_client: MagicMock
    ) -> None:
        mock_gemini_client.chat = AsyncMock(return_value="Response")

        response = client.post(
            "/api/llm/chat", json={"prompt": "Test", "system_prompt": "Be helpful"}
        )

        assert response.status_code == 200
        mock_gemini_client.chat.assert_called_once()


class TestLlmUsageEndpoint:
    """Tests for /api/llm/usage endpoint."""

    def test_get_usage(
        self, client: TestClient, mock_metrics_handler: MagicMock
    ) -> None:
        mock_metrics_handler.get_metrics.return_value = {
            "total_calls": 10,
            "total_input_tokens": 100,
            "total_output_tokens": 50,
            "total_tokens": 150,
            "total_reasoning_tokens": 0,
            "total_duration_ms": 1000,
        }

        response = client.get("/api/llm/usage")

        assert response.status_code == 200
        data = response.json()
        assert data["input_tokens"] == 100
        assert data["output_tokens"] == 50

    def test_get_usage_zero(
        self, client: TestClient, mock_metrics_handler: MagicMock
    ) -> None:
        mock_metrics_handler.get_metrics.return_value = {
            "total_calls": 0,
            "total_input_tokens": 0,
            "total_output_tokens": 0,
            "total_tokens": 0,
            "total_reasoning_tokens": 0,
            "total_duration_ms": 0,
        }

        response = client.get("/api/llm/usage")

        assert response.status_code == 200
        data = response.json()
        assert data["input_tokens"] == 0


class TestSessionEndpoints:
    """Tests for /api/session/* endpoints."""

    def test_create_session(
        self, client: TestClient, mock_langgraph_manager: MagicMock
    ) -> None:
        mock_session = MagicMock()
        mock_session.session_id = "test123"
        mock_session.model = "gemini-2.5-flash"
        mock_langgraph_manager.create_fresh_session = AsyncMock(
            return_value=mock_session
        )

        response = client.post("/api/sessions", json={"session_id": "test123"})

        assert response.status_code == 200
        assert response.json()["session_id"] == "test123"

    def test_create_session_auto_uuid(
        self, client: TestClient, mock_langgraph_manager: MagicMock
    ) -> None:
        """Test session creation with auto-generated UUID when session_id not provided."""
        mock_session = MagicMock()
        mock_session.session_id = "auto-generated-uuid"
        mock_session.model = "gemini-2.5-flash"
        mock_langgraph_manager.create_fresh_session = AsyncMock(
            return_value=mock_session
        )

        response = client.post("/api/sessions", json={})

        assert response.status_code == 200
        assert "session_id" in response.json()

    def test_end_session(
        self, client: TestClient, mock_langgraph_manager: MagicMock
    ) -> None:
        mock_langgraph_manager.end_session = AsyncMock(return_value=True)

        response = client.delete("/api/sessions/test123")

        assert response.status_code == 200
        assert response.json()["removed"] is True

    def test_get_session_info(
        self, client: TestClient, mock_langgraph_manager: MagicMock
    ) -> None:
        mock_langgraph_manager.get_session_info.return_value = {
            "session_id": "test",
            "model": "gemini-2.5-flash",
            "message_count": 5,
        }

        response = client.get("/api/sessions/test")

        assert response.status_code == 200
        assert response.json()["message_count"] == 5

    def test_get_session_info_not_found(
        self, client: TestClient, mock_langgraph_manager: MagicMock
    ) -> None:
        mock_langgraph_manager.get_session_info.return_value = None

        response = client.get("/api/sessions/nonexistent")

        assert response.status_code == 404


class TestGuardEndpoints:
    """Tests for /api/guard/* endpoints."""

    def test_evaluate_safe(self, client: TestClient, mock_guard: MagicMock) -> None:
        from mcp_llm_server.models.guard import GuardEvaluation

        mock_guard.evaluate = AsyncMock(
            return_value=GuardEvaluation(score=0.1, hits=[], threshold=0.7)
        )

        response = client.post("/api/guard/evaluations", json={"input_text": "hello"})

        assert response.status_code == 200
        data = response.json()
        assert data["malicious"] is False
        assert data["score"] == 0.1

    def test_is_malicious(self, client: TestClient, mock_guard: MagicMock) -> None:
        mock_guard.is_malicious = AsyncMock(return_value=True)

        response = client.post("/api/guard/checks", json={"input_text": "attack"})

        assert response.status_code == 200
        assert response.json()["malicious"] is True


class TestNlpEndpoints:
    """Tests for /api/nlp/* endpoints."""

    def test_analyze(self, client: TestClient, mock_nlp: MagicMock) -> None:
        from mcp_llm_server.infra.korean_nlp import NlpToken

        mock_nlp.analyze_async.return_value = [
            NlpToken(form="안녕", tag="NNG", position=0, length=2)
        ]

        response = client.post("/api/nlp/analyses", json={"text": "안녕"})

        assert response.status_code == 200
        data = response.json()
        assert len(data) == 1
        assert data[0]["form"] == "안녕"
        mock_nlp.analyze_async.assert_awaited_once_with("안녕")

    def test_anomaly_score(self, client: TestClient, mock_nlp: MagicMock) -> None:
        mock_nlp.calculate_anomaly_score_async.return_value = 0.3

        response = client.post("/api/nlp/anomaly-scores", json={"text": "테스트"})

        assert response.status_code == 200
        assert response.json()["score"] == 0.3
        mock_nlp.calculate_anomaly_score_async.assert_awaited_once_with("테스트")

    def test_heuristics(self, client: TestClient, mock_nlp: MagicMock) -> None:
        from mcp_llm_server.infra.korean_nlp import NlpHeuristics

        mock_nlp.analyze_heuristics_async.return_value = NlpHeuristics(
            numeric_quantifier=True,
            unit_noun=False,
            boundary_ref=False,
            comparison_word=False,
        )

        response = client.post("/api/nlp/heuristics", json={"text": "3개"})

        assert response.status_code == 200
        assert response.json()["numeric_quantifier"] is True
        mock_nlp.analyze_heuristics_async.assert_awaited_once_with("3개")


class TestRequestModels:
    """Tests for request/response Pydantic models."""

    def test_chat_request_validation(self) -> None:
        from mcp_llm_server.http_server import ChatRequest

        req = ChatRequest(prompt="test", system_prompt="system")
        assert req.prompt == "test"
        assert req.system_prompt == "system"

    def test_guard_request_validation(self) -> None:
        from mcp_llm_server.http_server import GuardRequest

        req = GuardRequest(input_text="test input")
        assert req.input_text == "test input"

    def test_nlp_request_validation(self) -> None:
        from mcp_llm_server.http_server import NlpRequest

        req = NlpRequest(text="한글 텍스트")
        assert req.text == "한글 텍스트"

    def test_session_create_request(self) -> None:
        from mcp_llm_server.http_server import SessionCreateRequest

        req = SessionCreateRequest(
            session_id="test", model="gemini-2.5-flash", system_prompt="Be helpful"
        )
        assert req.session_id == "test"
        assert req.model == "gemini-2.5-flash"

    def test_twentyq_hints_request(self) -> None:
        from mcp_llm_server.http_server import TwentyQHintsRequest

        req = TwentyQHintsRequest(
            target="사과", category="음식", details={"color": "red"}
        )
        assert req.target == "사과"
        assert req.category == "음식"


class TestUsageResponse:
    """Tests for UsageResponse model."""

    def test_usage_response(self) -> None:
        from mcp_llm_server.http_server import UsageResponse

        resp = UsageResponse(
            input_tokens=100, output_tokens=50, total_tokens=150, reasoning_tokens=None
        )
        assert resp.input_tokens == 100
        assert resp.output_tokens == 50
        assert resp.total_tokens == 150
        assert resp.reasoning_tokens is None
        assert resp.model is None

    def test_build_daily_usage_response_sets_model(self) -> None:
        from datetime import date

        from mcp_llm_server.http_server import _build_daily_usage_response
        from mcp_llm_server.infra.usage_repository import DailyUsage

        usage = DailyUsage(
            usage_date=date(2025, 1, 1),
            input_tokens=10,
            output_tokens=20,
            reasoning_tokens=0,
            request_count=2,
        )

        response = _build_daily_usage_response(
            usage, model="gemini-2.5-flash-preview-09-2025"
        )

        assert response.model == "gemini-2.5-flash-preview-09-2025"
        assert response.total_tokens == 30
        assert response.usage_date == "2025-01-01"

    def test_build_usage_list_response_propagates_model(self) -> None:
        from datetime import date

        from mcp_llm_server.http_server import _build_usage_list_response
        from mcp_llm_server.infra.usage_repository import DailyUsage

        usages = [
            DailyUsage(
                usage_date=date(2025, 1, 1),
                input_tokens=5,
                output_tokens=5,
                reasoning_tokens=0,
                request_count=1,
            )
        ]

        response = _build_usage_list_response(usages, model="gemini-2.5-pro")

        assert response.model == "gemini-2.5-pro"
        assert response.total_tokens == 10
        assert response.usages[0].model == "gemini-2.5-pro"
