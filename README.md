# llm 워크스페이스 안내

## 개요
- 이 루트는 LLM 공통 인프라와 두 봇 프로젝트를 모아 둔 상위 워크스페이스입니다.
- 공통 LLM 서버는 그대로 `mcp-llm-server/` 하위에 유지하며, 봇 코드는 각 디렉터리에서 별도 규칙으로 관리합니다.
- LLM 추론·가드·NLP 로직은 전부 `mcp-llm-server`가 담당하고, 봇들은 HTTP REST API를 통해 이 서버에 의존합니다. 봇 디렉터리에는 LLM 프롬프트/로직을 직접 두지 않습니다.

## 프로젝트 맵
- `mcp-llm-server/` : Python 3.13 FastAPI + Hypercorn(h2c) REST 서버 (LangChain Google GenAI ≥3.2.0, mcp ≥1.22.0, kiwipiepy ≥0.22.1, pyahocorasick). 품질도구: ruff/black/mypy/pytest. 봇들의 LLM 호출을 HTTP REST API로 제공하는 단일 진입점.
- `20q-kakao-bot/` : Kotlin 2.3.0-RC3 + Spring Boot 4.0.0 (WebFlux/Redis/R2DBC). Coroutines BOM 1.10.2, Redisson 3.52.0, detekt. LLM 호출은 mcp-llm-server REST API를 사용하며, 로컬 LLM 로직 없음.
- `turtle-soup-bot/` : Kotlin 2.3.0-RC3 + Ktor 3.3.2, Koin 4.0.0, coroutines 1.9.0, kotlinx.serialization 1.7.3, LangChain4j(0.36.x) + Google Gemini, Redis/Valkey. 품질도구: ktlint 1.0.1, detekt 1.23.4. LLM 호 출은 mcp-llm-server REST API를 경유하며, 로컬에 LLM 로직/프롬프트를 두지 않음.
- `.serena/` : Serena 도구 설정.

## 규칙 파일 진입점
- 공통 체인: `/home/kapu/.claude/CLAUDE.md` → `/home/kapu/gemini/CLAUDE.md` → 각 프로젝트 AGENT/CONVENTIONS.
- `mcp-llm-server/AGENT.MD`, `mcp-llm-server/CONVENTIONS.md`, `mcp-llm-server/CLAUDE.md`
- `20q-kakao-bot/CONVENTIONS.md` (별도 AGENT 없음)
- `turtle-soup-bot/AGENT.MD`, `turtle-soup-bot/CONVENTIONS.md`, `turtle-soup-bot/CLAUDE.md`

## 빠른 명령 요약
- 공통 전제: 각 프로젝트 디렉터리로 이동 후 실행.
- `mcp-llm-server` (Python):
  - 설치: `python -m venv .venv && source .venv/bin/activate && pip install -e ".[dev]"`
  - 품질: `ruff check src/ --fix && ruff format src/ && black src/ && mypy src/ && pytest`
  - 실행: `mcp-llm-server` 또는 `python -m mcp_llm_server.http_server`
- `20q-kakao-bot` (Kotlin):
  - 품질: `./gradlew detekt`
  - 테스트: `./gradlew test`
- `turtle-soup-bot` (Kotlin/Ktor):
  - 품질: `./gradlew ktlintCheck detekt`
  - 포맷: `./gradlew ktlintFormat`
  - 테스트: `./gradlew test`

## 배치 정책
- 공통 인프라(LLM MCP 서버)는 `mcp-llm-server/`에 유지. 경로 변경 시 규칙 체인, 스크립트, 도커 설정 전체 수정이 필요하므로 현 구조를 권장합니다.
- Docker/Compose: 공통 LLM 서버 도커화 지원(아래 참조). 봇들은 각 디렉터리의 Dockerfile/스크립트를 사용합니다.

## Docker / Compose
- `mcp-llm-server/Dockerfile`: python:3.13-slim 기반, 엔트리포인트 `mcp-llm-server`.
  - 통신은 HTTP(기본 40527)이며 `HTTP_HOST`/`HTTP_PORT`로 변경 가능. 외부 노출 시 `-p 40527:40527` 추가.
  - 빌드: `docker build -t mcp-llm-server ./mcp-llm-server`
  - 실행 예: `docker run --rm --tmpfs /tmp -p 40527:40527 -v mcp-llm-logs:/app/logs --env-file mcp-llm-server/.env mcp-llm-server` (HTTP + 로그 볼륨)
- `mcp-llm-server/docker-compose.yml`: Redis Stack(기본 46379→6379) 포함. 포트 충돌 시 `MCP_REDIS_HOST_PORT=46380 docker compose -f mcp-llm-server/docker-compose.yml up -d` (로컬 서버 실행 시 `REDIS_URL=redis://localhost:46380`도 함께 설정).
- 각 봇(`20q-kakao-bot`, `turtle-soup-bot`)도 Dockerfile을 갖고 있으며, 빌드/실행은 해당 디렉터리 README/스크립트 참조.
- 전체 스택(20q + turtle-soup + mcp-llm-server):  
  `docker compose -f 20q-kakao-bot/docker-compose.yml up -d --build`

### 개발(핫리로드)
- 실행:  
  `docker compose -f 20q-kakao-bot/docker-compose.yml -f 20q-kakao-bot/docker-compose.dev.yml up --build`
- 구성/동작:
  - `mcp-llm-server`는 `src/`를 컨테이너에 마운트하고 `HTTP_RELOAD=true`로 실행한다. Python 코드 수정 시 Hypercorn h2c 리로더가 자동 재시작한다.
  - `20q-kakao-bot`은 `20q-kakao-bot/Dockerfile.dev` + 소스 마운트로 `./gradlew bootRun --continuous` 모드에서 동작한다. Kotlin/리소스 변경 시 자동 재컴파일/재시작된다.
  - `turtle-soup-bot`은 `turtle-soup-bot/Dockerfile.dev` + 소스 마운트로 `./gradlew run --continuous` 모드에서 동작한다.
- 중지:  
  `docker compose -f 20q-kakao-bot/docker-compose.yml -f 20q-kakao-bot/docker-compose.dev.yml down`
- 주의:
  - 운영/배치에서는 `docker-compose.dev.yml`을 사용하지 않는다.
  - Gradle continuous 모드는 최초 빌드/재시작이 느릴 수 있으니 개발용으로만 사용한다.
