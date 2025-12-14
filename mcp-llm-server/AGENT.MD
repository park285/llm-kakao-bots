INHERIT:
  global: /home/kapu/.claude/CLAUDE.md (L0-L3 standards!)
  parent: /home/kapu/gemini/CLAUDE.md (workspace rules!)
  chain: Global L0-L3 -> gemini/CLAUDE.md -> mcp-llm-server/CLAUDE.md -> CONVENTIONS.md

PROJECT_CONTEXT:
  name: mcp-llm-server
  type: HTTP REST API Server
  purpose: LLM operations for 20q-kakao-bot and turtle-soup-bot
  lang: Python 3.13
  framework: FastAPI + Hypercorn (h2c)

SERENA_ACTIVATION!=FIRST:
  project_path: /home/kapu/gemini/mcp-llm-server
  relative_paths: src/mcp_llm_server/..., tests/...

CODE_QUALITY!=MANDATORY|ENFORCED:
  tools:
    ruff: Primary linter + formatter
    black: Secondary formatter
    mypy: Type checking (strict mode)
  commands:
    lint: ruff check src/ --fix
    format: ruff format src/ && black src/
    typecheck: mypy src/
    test: pytest -v
    all: ruff check src/ && ruff format --check src/ && mypy src/ && pytest
  enforcement!=CRITICAL:
    - 커밋 전 모든 검사 통과 필수
    - 타입 에러 0개 필수
    - 린팅 에러 0개 필수
    - 테스트 실패 시 커밋 금지
  pre_commit_hook:
    command: ruff check src/ && ruff format --check src/ && black --check src/ && mypy src/
    required: true

ENV_CONFIGURATION!=CRITICAL:
  files:
    .env: 실제 설정 (gitignore 대상)
    .env.example: 템플릿 (커밋 대상)
  required_vars:
    - GOOGLE_API_KEY (필수)
  structure:
    gemini_api:
      GOOGLE_API_KEY: 단일 API 키
      GOOGLE_API_KEYS: 복수 키 로테이션 (콤마/스페이스 구분)
    models:
      GEMINI_MODEL: 기본 모델
      GEMINI_HINTS_MODEL: 힌트 생성용
      GEMINI_ANSWER_MODEL: 답변 검증용
      GEMINI_VERIFY_MODEL: 검증용
    thinking_3x:
      GEMINI_THINKING_LEVEL: 기본 레벨 (none/low/medium/high)
      GEMINI_THINKING_LEVEL_HINTS: 힌트용
      GEMINI_THINKING_LEVEL_ANSWER: 답변용
      GEMINI_THINKING_LEVEL_VERIFY: 검증용
    thinking_25:
      GEMINI_THINKING_BUDGET: 기본 토큰 수 (0=비활성)
      GEMINI_THINKING_BUDGET_HINTS: 힌트용 (8192)
      GEMINI_THINKING_BUDGET_ANSWER: 답변용 (4096)
      GEMINI_THINKING_BUDGET_VERIFY: 검증용 (2048)
    generation:
      GEMINI_TEMPERATURE: 온도 (3.0은 1.0 고정)
      GEMINI_MAX_TOKENS: 최대 출력 토큰
    session:
      MAX_SESSIONS: 최대 동시 세션
      SESSION_TTL_MINUTES: 세션 TTL
      SESSION_HISTORY_MAX_PAIRS: 프롬프트 히스토리 Q/A 페어 제한
    guard:
      GUARD_ENABLED: 가드 활성화
      GUARD_THRESHOLD: 탐지 임계값 (0.85)

ARCHITECTURE:
  layers:
    server: http_server.py - FastAPI app + Hypercorn entrypoint
    routes: routes/ - APIRouter modules (llm/session/guard/nlp/usage/health)
    infra: infra/ - Core services
    config: config/ - Settings via dataclass
    models: models/ - Data models
    utils: utils/ - Utilities
  patterns:
    singleton: get_*() functions
    async: All LLM operations
    caching: LRU cache for models

KEY_COMPONENTS:
  Settings:
    path: src/mcp_llm_server/config/settings.py
    classes:
      ThinkingConfig: 태스크별 thinking 설정
      GeminiSettings: API 키, 모델, 온도
      SessionSettings: 세션 관리
      GuardSettings: 인젝션 가드
      Settings: 메인 컨테이너
    methods:
      get_settings() -> Settings (singleton)
      GeminiSettings.get_model(task?) -> str
      GeminiSettings.get_temperature(model) -> float
      ThinkingConfig.get_level(task?) -> str
      ThinkingConfig.get_budget(task?) -> int?

  GeminiClient:
    path: src/mcp_llm_server/infra/gemini_client.py
    methods:
      chat(prompt, system?, history?, model?) -> str
      chat_structured(prompt, schema, ...) -> T
      stream(prompt, ...) -> AsyncIterator[str]
      chat_with_tools(prompt, tools, ...) -> tuple[str, list[ToolCall]]
      get_last_usage() -> TokenUsage
    singleton: get_gemini_client()

  InjectionGuard:
    path: src/mcp_llm_server/infra/injection_guard.py
    methods:
      evaluate(text) -> GuardEvaluation
      is_malicious(text) -> bool
    singleton: get_injection_guard()

  KoreanNlpService:
    path: src/mcp_llm_server/infra/korean_nlp.py
    methods:
      analyze(text) -> list[NlpToken]
      calculate_anomaly_score(text) -> float
      analyze_heuristics(text) -> NlpHeuristics
    singleton: get_korean_nlp_service()

REST_API:
  health: /health, /health/ready, /health/live, /health/models
  llm: /api/llm/*
  session: /api/sessions/*
  guard: /api/guard/*
  nlp: /api/nlp/*
  usage: /api/usage/*
  domains: /api/twentyq/*, /api/turtle-soup/*

PYTHON_PATTERNS!=CRITICAL:
  typing:
    - 모든 함수에 타입 힌트 필수
    - Optional[T] or T | None
    - list[T], dict[K, V] (소문자)
  async:
    - async def for I/O
    - await for async calls
  dataclass:
    - frozen=True for immutable
    - field(default_factory=...) for mutable
  singleton:
    - Module-level _instance
    - get_*() accessor

COMMON_TASKS:
  add_env_var:
    1. .env.example에 추가 (주석 포함)
    2. .env에 추가
    3. settings.py에 필드 추가
    4. README.md 업데이트
  
  add_rest_endpoint:
    1. routes/에 APIRouter 추가 또는 http_server.py에 엔드포인트 추가
    2. 요청/응답 모델(BaseModel) + 타입 힌트 + docstring 필수
    3. http_server.py에 include_router 필요 여부 확인
    4. README.md 엔드포인트 목록 업데이트
  
  modify_settings:
    1. settings.py의 해당 *Settings 클래스에 필드 추가
    2. .env.example과 .env에 환경 변수 추가
    3. README.md Configuration 업데이트

FILE_OPERATIONS:
  before_edit: Read file or use get_symbols_overview
  prefer: Symbol-level edits
  fallback: replace_content with regex

GIT_WORKFLOW:
  pre_commit: ruff check && ruff format --check && mypy && pytest
  branch: feature/*, fix/*
  commit_msg: Conventional commits (feat:, fix:, refactor:)
  X: .env 커밋 금지 (API 키 포함)
