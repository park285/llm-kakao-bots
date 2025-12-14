# Turtle Soup Bot (Kotlin/Ktor)

바다거북숲(Lateral Thinking Puzzle) 게임 챗봇 - LangChain4j + Ktor 기반

## Tech Stack

| 레이어 | 기술 | 역할 |
|--------|------|------|
| Runtime | Java 21 + Kotlin 2.3 | ZGC(저지연 GC) + 최신 Coroutines |
| Server | Ktor Server (Netty) | 웹 요청 처리. Spring 대비 압도적 가벼움 |
| DI | Koin | Ktor의 표준 DI |
| AI Framework | LangChain4j (Core + Google AI Gemini 2.5 Flash) | Spring Starter 제외. 순수 라이브러리 사용 |
| AI Logic | AiServices (Interface) | 게임 마스터 로직을 선언형 인터페이스로 구현 |
| Data Model | kotlinx.serialization | Pydantic의 완벽한 대체재. Reflection 없이 동작 |
| Memory | Redis (via LangChain4j) | 대화 맥락(Context) 저장소 |
| Validation | Konform | Pure Kotlin DSL 검증 라이브러리 |
| Logging | kotlin-logging | Structured logging |
| Testing | Kotest + MockK | BDD 스타일 테스트 |
| Linting | ktlint + detekt | 코드 품질 관리 |

## Project Structure

```
turtle-soup-bot/
├── src/main/kotlin/io/github/kapu/turtlesoup/
│   ├── Application.kt                   # Ktor main
│   ├── config/
│   │   ├── KoinModule.kt               # DI 설정
│   │   ├── Settings.kt                 # 환경 설정
│   │   ├── Constants.kt                # 상수
│   ├── models/                          # kotlinx.serialization
│   │   ├── GameState.kt
│   │   ├── Puzzle.kt
│   │   ├── QuestionHistory.kt
│   │   └── PuzzleCategory.kt
│   ├── llm/                             # LangChain4j
│   │   ├── GameMasterService.kt        # AiServices interface
│   │   ├── PuzzleGeneratorService.kt   # 퍼즐 생성 AI
│   │   └── PromptLoader.kt
│   ├── redis/                           # Lettuce + LangChain4j
│   │   ├── SessionStore.kt             # GameState 저장
│   │   └── LockManager.kt
│   ├── service/
│   │   ├── GameService.kt              # 게임 로직
│   │   └── PuzzleService.kt
│   ├── api/                             # Ktor routes
│   │   ├── GameRoutes.kt
│   │   └── dto/
│   └── utils/                           # Extensions
│       ├── StringExtensions.kt
│       ├── DurationExtensions.kt
│       ├── GameStateExtensions.kt
│       └── Exceptions.kt
└── src/main/resources/
    ├── application.conf                 # Ktor 설정
    ├── logback.xml
    ├── lua/                             # Redis Lua 스크립트
    └── messages/                        # 메시지 번역 리소스
```

## Installation

### Prerequisites

- Docker 24+
- (로컬 개발 시) Java 21+, Gradle 8.5+, Redis/Valkey 9.0

### Quick Start (Docker 권장)

```bash
# 1. Clone repository
git clone <repository-url>
cd turtle-soup-bot

# 2. Environment setup
cp .env.example .env
# Edit .env with your API keys

# 3. Build image (Dockerfile 사용)
docker build -t turtle-soup-bot .
# 또는 Ktor Docker 플러그인
./gradlew buildImage

# 4. Run (UDS 마운트 + 포트 노출)
docker run --rm --name turtle-soup-bot \
  --env-file .env \
  --tmpfs /tmp \
  -v turtle-soup-logs:/app/logs \
  -v /run/mcp-llm:/run/mcp-llm \
  -p 40257:40257 \
  turtle-soup-bot

# 종료
docker stop turtle-soup-bot
```

> LLM MCP 서버는 이제 h2c(HTTP/2 cleartext)로 통신합니다. 기본 엔드포인트는 `.env`의 `LLM_REST_BASE_URL`(예: `http://localhost:40527`)이며, 봇 서비스 포트는 `SERVER_PORT`(기본 40257)로 노출됩니다.

### Local Dev (옵션)

```bash
# Fat JAR 빌드
./gradlew shadowJar

# 로컬 실행 (PID/로그 관리 스크립트)
./bot-start.sh
./bot-status.sh
./bot-restart.sh
./bot-stop.sh

# 개발용 핫리로드
./gradlew run --continuous
```

## Configuration

Edit `.env`:

```bash
GOOGLE_API_KEY=your-google-api-key-here
GEMINI_MODEL=gemini-2.5-flash-preview-09-2025
REDIS_HOST=localhost
REDIS_PORT=6379
```

## API Endpoints

### Start Game
```bash
POST /api/game/start
{
  "sessionId": "session-123",
  "userId": "user-456",
  "chatId": "chat-789",
  "category": "MYSTERY",  // optional
  "difficulty": 3         // optional
}
```

### Ask Question
```bash
POST /api/game/question
{
  "sessionId": "session-123",
  "question": "이것은 살인 사건인가요?"
}
```

### Submit Solution
```bash
POST /api/game/solution
{
  "sessionId": "session-123",
  "answer": "범인은 쌍둥이 형제였다"
}
```

### Request Hint
```bash
POST /api/game/hint
{
  "sessionId": "session-123"
}
```

### Get Game Status
```bash
GET /api/game/status/{sessionId}
```

### End Game
```bash
DELETE /api/game/{sessionId}
```

## Development

### Run Tests
```bash
./gradlew test
```

### Lint & Format
```bash
# Check code style
./gradlew ktlintCheck
./gradlew detekt

# Auto-fix
./gradlew ktlintFormat
```

### Build
```bash
# Build JAR
./gradlew build

# Build Docker image
./gradlew buildImage
```

## LangChain4j Architecture

### 핵심 개념: AiServices (선언형 인터페이스)

Python LangGraph의 노드 기반 상태 머신 대신, LangChain4j는 **선언형 인터페이스**를 사용합니다:

```kotlin
interface GameMasterService {
    @SystemMessage("You are a game master...")
    @UserMessage("Question: {{question}}")
    fun answerQuestion(
        @MemoryId sessionId: String,
        @V("question") question: String
    ): String
}

// LangChain4j가 자동으로 구현체 생성
val gameMaster = AiServices.builder(GameMasterService::class.java)
    .chatLanguageModel(geminiModel)
    .chatMemoryProvider { sessionId -> ... }
    .build()
```

### 대화 맥락 관리 (Redis ChatMemoryStore)

```kotlin
// LangChain4j가 자동으로 Redis에 대화 히스토리 저장/복원
val memoryStore = RedisChatMemoryStore.builder()
    .redisClient(lettuceClient)
    .ttlSeconds(1800)
    .build()

val chatMemory = MessageWindowChatMemory.builder()
    .chatMemoryStore(memoryStore)
    .id(sessionId)
    .maxMessages(100)
    .build()
```

**장점:**
- 상태 머신 없이 간단한 인터페이스만으로 구현
- `@MemoryId`로 세션별 독립적인 대화 맥락 자동 관리
- Redis 통합으로 분산 환경에서도 세션 공유 가능

## Performance

- **ZGC (Z Garbage Collector)**: 초저지연 GC, 대부분 10ms 이하
- **Ktor Netty**: Spring MVC 대비 10배 빠른 기동 속도
- **kotlinx.serialization**: Reflection 없이 컴파일 타임 최적화
- **Lettuce**: Netty 기반 비동기 Redis 클라이언트
- **LangChain4j**: Spring Starter 없이 순수 라이브러리로 최소 의존성

## License

Proprietary
