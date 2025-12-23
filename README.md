# llm 워크스페이스 안내

## 개요
- 이 루트는 LLM 공통 인프라와 두 봇 프로젝트를 모아 둔 상위 워크스페이스입니다.
- 운영 기준 공통 LLM 서버는 `mcp-llm-server-go/`(Go)이며, `mcp-llm-server/`(Python)는 레거시/개발용으로 유지합니다.
- LLM 추론·가드·NLP 로직은 전부 `mcp-llm-server-go`가 담당하고, 봇들은 HTTP REST API를 통해 이 서버에 의존합니다. 봇 디렉터리에는 LLM 프롬프트/로직을 직접 두지 않습니다.

## 프로젝트 맵
- `mcp-llm-server-go/` : Go(Gin + h2c) LLM REST 서버 (google.golang.org/genai, guard, session, usage). 품질도구: gofmt/goimports/gci/golangci-lint.
- `mcp-llm-server/` : Python 3.13 FastAPI + Hypercorn(h2c) REST 서버 (LangChain Google GenAI ≥3.2.0, mcp ≥1.22.0, kiwipiepy ≥0.22.1, pyahocorasick). 레거시/개발용. 품질도구: ruff/black/mypy/pytest.
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
- `mcp-llm-server-go` (Go):
  - 품질: `make lint`, `make fmt`
  - 테스트: `make test` (또는 `go test ./...`)
  - 빌드: `make build` (출력: `bin/server`)
  - 실행: `./bin/server` 또는 `go run ./cmd/server`
  - 스모크(LLM 미호출): `bash scripts/smoke_test.sh`
  - LLM 실동작: `bash scripts/llm_live_test.sh /home/kapu/gemini/llm/.env`
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
- Docker/Compose: 운영은 루트의 `docker-compose.prod.yml`만 사용합니다.

## Docker / Compose (운영)
- 운영 스택 파일: `docker-compose.prod.yml` (Go 기반 `mcp-llm-server-go` 사용)
- 컨테이너 이름이 고정(`container_name`)이므로 프로젝트명은 `-p 20q-kakao-bot`로 통일합니다.

### 환경 변수 (SSOT)
- 운영 스택 설정의 단일 소스는 루트 `./.env` 입니다.
- `docker-compose.prod.yml`은 `mcp-llm-server-go`/봇 컨테이너에 `env_file: ./.env`로 주입하며, `${VAR}` 치환에도 동일 파일을 사용합니다.
- `mcp-llm-server/.env`는 로컬에서 서버를 단독 실행할 때만 사용(Compose 운영 스택에서는 미사용)합니다.
- (옵션) `HTTP_API_KEY`를 설정하면 `mcp-llm-server-go`의 `/api/*`가 인증 모드로 동작하며, Go 봇은 동일 키를 자동 전송합니다.
- (옵션) `HTTP_RATE_LIMIT_RPM`로 `mcp-llm-server-go`의 `/api/*` 레이트리밋을 활성화할 수 있습니다.
- 세션 스토어는 `SESSION_STORE_URL`, `SESSION_STORE_ENABLED`, `SESSION_STORE_REQUIRED`로 제어하며, 운영은 `valkey-cache`(+AOF) 기준입니다.
- 로그 파일 설정은 `LOG_DIR`, `LOG_FILE_MAX_SIZE_MB`, `LOG_FILE_MAX_BACKUPS`, `LOG_FILE_MAX_AGE_DAYS`, `LOG_FILE_COMPRESS`를 사용합니다.
- (Gemini3) `GEMINI_TEMPERATURE`는 **실제 적용 값이 1.0 미만으로 내려가지 않으며**, `/health/models`의 `temperature`로 확인할 수 있습니다.

### 실행/재기동
```bash
docker compose -p 20q-kakao-bot -f docker-compose.prod.yml up -d --force-recreate
```

### 중지
```bash
docker compose -p 20q-kakao-bot -f docker-compose.prod.yml down --remove-orphans
```

### 로그
- 호스트의 `./logs/`에 파일로 기록됩니다. (예: `server.log`, `twentyq.log`, `turtlesoup.log`)
