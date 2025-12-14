# mcp-llm-server (HTTP REST API)

스무고개봇(20q-kakao-bot) 및 바다거북스프 봇(turtle-soup-bot)이 호출하는 LLM 백엔드 서버.
FastAPI + Hypercorn(h2c) 기반으로 REST API를 제공하며, LangChain 기반 Gemini 통합/인젝션 가드/한국어 NLP/세션 히스토리를 포함한다.

## Features

- **LLM**: Gemini 2.5/3 모델 호출 + usage 수집
- **Sessions**: LangGraph 기반 세션 히스토리 관리 (Redis checkpointer 지원)
- **Injection Guard**: 규칙 기반 + 형태소 이상치 기반 탐지
- **Korean NLP**: Kiwipiepy 기반 형태소 분석/휴리스틱

## Requirements

- Python 3.13

## Installation

```bash
python -m venv .venv
source .venv/bin/activate
pip install -e ".[dev]"
```

## Configuration

```bash
cp .env.example .env  # GOOGLE_API_KEY 설정 필수
```

주요 환경 변수:

```env
GOOGLE_API_KEY=your_api_key
GEMINI_MODEL=gemini-2.5-flash-preview-09-2025

MAX_SESSIONS=50
SESSION_TTL_MINUTES=1440
SESSION_HISTORY_MAX_PAIRS=10  # 프롬프트에 포함할 히스토리 Q/A 페어 수 (0=미포함)
```

## Running

```bash
mcp-llm-server

# 또는
python -m mcp_llm_server.http_server
```

## REST API

- **OpenAPI**: `/docs` (기본 FastAPI Swagger UI)

**Health**
- `GET /health`, `GET /health/ready`, `GET /health/live`, `GET /health/models`

**LLM (`/api/llm`)**
- `POST /api/llm/chat`
- `POST /api/llm/stream`
- `POST /api/llm/chat-with-usage`
- `POST /api/llm/stream-events`
- `POST /api/llm/structured`
- `GET /api/llm/usage`
- `GET /api/llm/usage/total`
- `GET /api/llm/metrics`

**Sessions (`/api/sessions`)**
- `POST /api/sessions`
- `POST /api/sessions/{session_id}/messages`
- `GET /api/sessions/{session_id}`
- `DELETE /api/sessions/{session_id}`

**Guard (`/api/guard`)**
- `POST /api/guard/evaluations`
- `POST /api/guard/checks`

**NLP (`/api/nlp`)**
- `POST /api/nlp/analyses`
- `POST /api/nlp/anomaly-scores`
- `POST /api/nlp/heuristics`

**Usage (`/api/usage`)**
- `GET /api/usage/daily`
- `GET /api/usage/recent?days=7`
- `GET /api/usage/total?days=30`

**TwentyQ (`/api/twentyq`)**
- `POST /api/twentyq/hints`
- `POST /api/twentyq/answers`
- `POST /api/twentyq/verifications`
- `POST /api/twentyq/normalizations`
- `POST /api/twentyq/synonym-checks`

**TurtleSoup (`/api/turtle-soup`)**
- `POST /api/turtle-soup/answers`
- `POST /api/turtle-soup/hints`
- `POST /api/turtle-soup/validations`
- `POST /api/turtle-soup/reveals`
- `POST /api/turtle-soup/puzzles`
- `POST /api/turtle-soup/rewrites`
- `GET /api/turtle-soup/puzzles`
- `GET /api/turtle-soup/puzzles/random`
- `GET /api/turtle-soup/puzzles/{puzzle_id}`
- `POST /api/turtle-soup/puzzles/reload`

## Project Structure

```
src/mcp_llm_server/
├── http_server.py          # FastAPI + Hypercorn(h2c) 엔트리포인트
├── routes/                 # 공용 API 라우터 (llm/session/guard/nlp/usage/health)
├── domains/                # TwentyQ/TurtleSoup 도메인 모델/프롬프트/리소스
├── infra/                  # GeminiClient, LangGraph 세션, Guard/NLP, DB 등
├── config/                 # settings/logging
├── middleware.py           # request-id 등 공용 미들웨어
└── rulepacks/              # 인젝션 규칙팩
```
