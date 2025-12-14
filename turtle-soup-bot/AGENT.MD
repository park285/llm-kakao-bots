# TURTLE-SOUP-BOT AGENT RULES (Kotlin/Ktor)
> Project-Specific | 2025-11-23 | SYM: !=CRIT, X=PROHIB, ?=OPT, →=to, [N]=count

INHERIT:
  chain: Global AGENT → gemini/AGENT.md → turtle-soup-bot/AGENT.md
  ref: /home/kapu/.claude/CLAUDE.MD (fallback: /home/kapu/.claude/CLAUDE.MD), /home/kapu/gemini/CLAUD.MD
  scope: turtle-soup-bot project-specific rules only
  conventions!=CRITICAL: CONVENTIONS.md MANDATORY before ANY code write/edit

PROJECT_ROOT:
  absolute: /home/kapu/gemini/turtle-soup-bot
  working_dir: Always verify with pwd before operations

MCP_TOOLS:
  ref: /home/kapu/gemini/AGENT.md MCP_STACK section
  primary!=serena: Semantic code toolkit
    activate!=FIRST: activate_project(turtle-soup-bot) OR activate_project(/home/kapu/gemini/turtle-soup-bot)
    paths: RELATIVE from project root (e.g., src/main/kotlin/..., NOT /home/kapu/...)
    use: ALL code reading/editing/refactoring operations
  secondary[3]: context7 (library docs), ide (diagnostics), sequentialthinking (analysis)

STACK:
  lang: Kotlin 2.3.0-RC3 + Java 25
  framework: Ktor 3.3.2 (Netty engine)
  ai: LangChain4j 0.36.2 + Google AI Gemini 2.5 Flash (Java SDK)
  workflow: LangChain4j AiServices (declarative interfaces)
  cache: Redis/Valkey 9.0 (Lettuce client)
  di: Koin 4.0.0
  serialization: kotlinx.serialization 1.7.3
  validation: Konform 0.6.1
  logging: kotlin-logging 7.0.0 + Logback 1.5.12
  test: Kotest 5.9.1 + MockK 1.13.13
  lint: ktlint 1.0.1 + detekt 1.23.4
  build: Gradle 8.5+ with Kotlin DSL

STRUCTURE:
  src/main/kotlin/io/github/kapu/turtlesoup/: config/, models/, redis/, rest/, service/, api/, bridge/, mq/, security/, utils/
  src/main/resources/: application.conf, logback.xml, messages/, lua/
  src/test/kotlin/: mirrors main structure
  build.gradle.kts: Gradle build config
  detekt.yml: Static analysis config
  .editorconfig: Code style

COMMANDS:
  build: ./gradlew build
  run: ./gradlew run
  test: ./gradlew test (-v for verbose, --tests for specific)
  lint: ./gradlew ktlintCheck detekt
  format: ./gradlew ktlintFormat
  all_checks: make lint && ./gradlew test (!=pre-commit)
  clean: ./gradlew clean
  docker: ./gradlew buildImage

RULES:
  workflow[4]!=CRIT: REPORT → PROPOSAL → APPROVAL → EXECUTION
    principle: X execute without explicit 사용자님 approval
    scope: ALL significant ops (file edits, config, deployments, git)
    evidence_driven!: base on actual file contents/logs/test results (X assumptions)
    session_cache: approved tools/patterns in same session = reusable (X re-approval)

  git!=CRIT: MANDATORY user approval + impact scope
    prohibited_without_approval[8]: git add, commit, push, rebase, reset --hard, merge, cherry-pick, branch -D
    approval_flow[4]: ANALYZE (status/diff) → REPORT (scope) → AWAIT approval → EXECUTE
    impact_scope: files (list + summary), branch (current/target), history (commits affected), risk
    read_only_allowed[4]: status, log, diff, show (X approval needed)

  security!=CRITICAL: NEVER commit to GitHub
    X[4]: LLM traces (AI/LLM/Claude/Gemini in commits/PRs/docs), Sensitive (.env, API keys, tokens, credentials), Artifacts (.claude/, *.local.json, session logs), Personal data
    verify: grep Claude|Gemini|API.*key|secret|token before git add

  format: 4-space indent (handled by ktlint), PascalCase files (GameService.kt)
  dto: kotlinx.serialization data classes, immutable (val + copy)
  logging: kotlin-logging (structured), X string interpolation
  test: descriptive names (should_generate_hint_when_valid), ≥80% coverage
  commit: Conventional Commits (feat:, fix:, refactor:, test:, chore:)
    X LLM! NEVER mention AI/LLM/Claude/Gemini in commits/PRs/docs to GitHub!

  naming!=MANDATORY:
    follow: codebase patterns FIRST!
    files: PascalCase.kt (GameService.kt, StringExtensions.kt)
    packages: lowercase (io.github.kapu.turtlesoup.service)
    classes: PascalCase (GameService, SessionStore)
    functions: camelCase (startGame, validateSolution), concise!, feature-focused
      bool: is*/has*/can*, collection: plural, converter: to*
    constants: SCREAMING_SNAKE_CASE (MAX_QUESTIONS, SESSION_TTL_MINUTES)
    properties: camelCase (questionCount, hintsUsed)
    type_hints: MANDATORY for ALL function signatures

  comments!=MINIMAL:
    docstrings: 간결한 한국어로 핵심 기능만 (X English, X verbose!)
      use: public classes, services, major components
      format: /** 클래스/함수의 핵심 기능 한 줄 설명 */
      GOOD: /** 힌트 생성 및 관리 서비스 */
      BAD: /** 이 서비스는 사용자의 게임 진행도를 분석하여 적절한 난이도의 힌트를 생성하고... */
    inline: compressed Korean, action-only, NO subject
      use: complex logic, edge cases, non-obvious behavior ONLY
      GOOD: // 사용자 입력 검증, // edge case: 빈 리스트
      BAD: // 이 함수는 클라이언트로부터 받은 사용자 입력을 검증합니다
    minimal: prefer self-documenting code (clear naming) over comments

  emoji!=STRICT:
    allowed: UI text for end users ONLY (message_provider messages)
    X internal: code, logs, comments, commits, PRs, docs, stack traces
    rationale: professional codebase, avoid encoding issues, log parsability

  LLM: REST API 호출로 외부 프롬프트/모델 관리 (로컬 YAML 프롬프트 없음)
  api: /api/{service}/{resource}, Ktor routes
  ktlint_detekt!=MANDATORY: pass before commit
    ktlint: ./gradlew ktlintCheck && ./gradlew ktlintFormat
    detekt: ./gradlew detekt (strict mode enabled)

CONTEXT_COMPRESSION:
  trigger: 사용자님 → 압축 or context approaching limit
  flow[4]!=MANDATORY: ANALYZE current conversation, SUMMARIZE (7-section format), SAVE!=CRITICAL write_memory(session-YYYYMMDD-HHMM), CONFIRM report saved
  cleanup!=CRITICAL: DELETE session memory after reading with delete_memory
  rationale: Preserve work context across sessions
  memory_naming: session-{timestamp} or feature-{name}-{date}

ENV_KEYS:
  required[4]: GOOGLE_API_KEY, GEMINI_MODEL, REDIS_HOST, REDIS_PORT
  gemini: GEMINI_MODEL, (temperature/maxTokens in application.conf)
  file: .env (X commit!)
  template: .env.example

DOMAIN_PACKAGES:
  config: Settings (Typesafe Config), Constants, Koin DI module
  models: kotlinx.serialization data classes (GameState, Puzzle, etc.)
  rest: LLM REST client + DTOs
  redis: Lettuce client, SessionStore, LockManager, ChatMemoryStore integration
  service: Business logic (GameService, PuzzleService)
  api: Ktor routes + DTOs
  bridge/mq/security/utils: message parsing, MQ, guards, helpers
  api: Ktor routes, DTOs, error handling
  utils: Extension functions (String, Duration, GameState), Exceptions

LANGCHAIN4J_PATTERNS!=CRITICAL:
  ai_services: Declarative interfaces with @SystemMessage, @UserMessage, @MemoryId, @V
    example:
      interface GameMasterService {
          @SystemMessage("System instructions...")
          @UserMessage("Question: {{question}}")
          fun answerQuestion(
              @MemoryId sessionId: String,
              @V("puzzle") puzzle: Puzzle,
              @V("question") question: String
          ): String
      }

  chat_memory: Redis ChatMemoryStore auto-manages conversation history
    setup:
      val memoryStore = RedisChatMemoryStore.builder()
          .redisClient(lettuceConnection)
          .ttlSeconds(1800)
          .build()

      val chatMemory = MessageWindowChatMemory.builder()
          .chatMemoryStore(memoryStore)
          .id(sessionId)
          .maxMessages(100)
          .build()

  structured_output: Return Kotlin data classes, LangChain4j auto-parses JSON
    example:
      @Serializable
      data class SolutionValidation(val isCorrect: Boolean, val explanation: String)

      fun validateSolution(...): SolutionValidation  // Auto-parsed from JSON

KTOR_PATTERNS!=CRITICAL:
  routes: Extension functions on Application
    fun Application.configureGameRoutes() {
        val service: GameService by inject()
        routing {
            route("/api/game") {
                post("/start") { ... }
            }
        }
    }

  error_handling: StatusPages plugin
    install(StatusPages) {
        exception<TurtleSoupException> { call, cause ->
            call.respond(HttpStatusCode.BadRequest, ErrorResponse(...))
        }
    }

  di: Koin
    val service: GameService by inject()

PERFORMANCE!=CRITICAL:
  coroutines: Suspend functions for I/O, structured concurrency
  collections: Immutable (List, Set, Map), sequence for large data
  redis: Lettuce coroutines API (commands.get, commands.setex)
  serialization: kotlinx.serialization (compile-time, no reflection)
  gc: ZGC configuration in gradle.properties

PROMPT_POLICY!=CRITICAL:
  system_prompt: 영어 (English) - AI가 이해하기 쉬운 언어
  ai_response: 한국어 (Korean) - 사용자에게 제공되는 최종 답변
  rationale: 프롬프트 효율성(영어) + 사용자 경험(한국어) 최적화
  example:
    system: "You are a lateral thinking puzzle game master. Generate creative hints based on user progress."
    ai_output: "힌트: 이 퍼즐은 시간 역행과 관련이 있습니다."
