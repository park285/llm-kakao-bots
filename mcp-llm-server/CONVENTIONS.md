# MCP LLM Server Conventions

코드 작성 전 반드시 확인. 규칙 위반 시 PR 거부.

## Code Quality (강제)

```bash
# 설치
pip install -e ".[dev]"

# 커밋 전 필수 실행
ruff check src/ --fix && ruff format src/ && black src/ && mypy src/ && pytest
```

| 도구 | 용도 | 실패 시 |
|------|------|---------|
| Ruff | 린팅 + 포맷팅 | 커밋 금지 |
| Black | 포맷팅 보조 | 커밋 금지 |
| Mypy | 타입 체크 (strict) | 커밋 금지 |
| Pytest | 테스트 | 커밋 금지 |

## Environment Variables

```bash
cp .env.example .env  # GOOGLE_API_KEY 설정 필수
```

| 변수 | 설명 | 기본값 |
|------|------|--------|
| `GOOGLE_API_KEY` | API 키 (필수) | - |
| `GOOGLE_API_KEYS` | 복수 키 (콤마/스페이스) | - |
| `GEMINI_MODEL` | 기본 모델 | `gemini-2.5-flash-preview-09-2025` |
| `GEMINI_*_MODEL` | 태스크별 (HINTS/ANSWER/VERIFY) | - |
| `GEMINI_THINKING_LEVEL_*` | 3.0 사고 레벨 | `low` |
| `GEMINI_THINKING_BUDGET_*` | 2.5 사고 토큰 | 다양 |
| `GUARD_THRESHOLD` | 인젝션 임계값 | `0.85` |
| `SESSION_HISTORY_MAX_PAIRS` | 프롬프트 히스토리 Q/A 페어 제한 | `10` |

**환경 변수 추가 시**: `.env.example` → `.env` → `settings.py` → `README.md`

## Naming

```python
variable_name = "snake_case"       # 변수/함수
CONSTANT_VALUE = 100               # 상수
ClassName = ...                    # 클래스
_private_field = None              # 프라이빗
```

## Type Hints (필수)

```python
# 모든 함수에 타입 필수
def process(text: str, count: int = 0) -> str: ...

# 컬렉션 (소문자)
def analyze(items: list[str]) -> dict[str, int]: ...

# Optional
from typing import Optional
def get(id: str) -> Optional[User]: ...  # or User | None
```

## Async

```python
# 비동기 함수
async def chat(prompt: str) -> str:
    return await self._llm.ainvoke(messages)

# AsyncIterator
async def stream(prompt: str) -> AsyncIterator[str]:
    async for chunk in self._llm.astream(messages):
        yield chunk
```

## Dataclass

```python
# Immutable (설정용)
@dataclass(frozen=True)
class Settings:
    api_key: str = field(default_factory=lambda: os.getenv("KEY", ""))

# Mutable (데이터용)
@dataclass
class Session:
    messages: list[Message] = field(default_factory=list)
```

## Singleton

```python
_instance: Optional[MyClass] = None

def get_instance() -> MyClass:
    global _instance
    if _instance is None:
        _instance = MyClass()
    return _instance
```

## FastAPI Route

```python
from fastapi import APIRouter
from pydantic import BaseModel


router = APIRouter(prefix="/api/example", tags=["Example"])


class ExampleRequest(BaseModel):
    """요청 바디"""

    text: str


class ExampleResponse(BaseModel):
    """응답 바디"""

    ok: bool


@router.post("/echo", response_model=ExampleResponse)
async def api_echo(request: ExampleRequest) -> ExampleResponse:
    """한 줄 설명.

    Args:
        request: 요청 바디.

    Returns:
        응답 바디.
    """
    return ExampleResponse(ok=bool(request.text.strip()))
```

## Import Order

```python
# 1. 표준 라이브러리
import os
from typing import Optional

# 2. 서드파티
from fastapi import APIRouter

# 3. 로컬
from mcp_llm_server.config.settings import get_settings
```

## Logging

```python
import logging
log = logging.getLogger(__name__)

log.info("메시지: %s", data)      # placeholder 사용
log.exception("예외 발생")        # traceback 포함
```

## Line Length

**88자** 

```toml
# pyproject.toml
[tool.ruff]
line-length = 88

[tool.black]
line-length = 88
```

## Docstring (Google Style)

```python
def calc(text: str, threshold: float = 0.7) -> float:
    """점수 계산.

    Args:
        text: 분석 텍스트
        threshold: 임계값

    Returns:
        0.0~1.0 점수
    """
```

**타입 중복 금지**: 타입은 annotation에만 작성, docstring에서 타입 설명 생략

```python
# GOOD: 타입은 annotation에만
def process(text: str, count: int) -> list[str]:
    """텍스트 처리.

    Args:
        text: 처리할 텍스트
        count: 반복 횟수

    Returns:
        처리된 문자열 목록
    """

# BAD: docstring에 타입 중복
def process(text: str, count: int) -> list[str]:
    """텍스트 처리.

    Args:
        text (str): 처리할 텍스트  # X 타입 중복
        count (int): 반복 횟수     # X 타입 중복

    Returns:
        list[str]: 처리된 문자열 목록  # X 타입 중복
    """
```

## TODO Comments (Google Style)

**형식**: `# TODO: <issue-url> - <explanation>` 또는 `# TODO(owner): <explanation>`

```python
# GOOD: 참조 URL/이슈 포함
# TODO: https://github.com/langchain-ai/langchain-google/pull/1330 - thinking_level 지원 시 전환
# TODO: #123 - 성능 최적화 필요
# TODO(kapu): 다음 릴리스에서 deprecated 예정

# BAD: 참조 없는 TODO
# TODO: 나중에 수정
# TODO: LangChain 업데이트 후 전환
```

## 금지 사항

| 절대 금지 | 지양 |
|-----------|------|
| 타입 힌트 없는 함수 | 과도한 주석 |
| `Any` 타입 남용 | 중첩 3단계 이상 |
| 테스트 없는 기능 | 함수 50줄 초과 |
| `.env` 커밋 | 전역 변수 |
| 린팅 에러 무시 | `# type: ignore` 남용 |
