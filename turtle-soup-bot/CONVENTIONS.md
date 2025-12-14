# TURTLE-SOUP-BOT CONVENTIONS (Kotlin/Ktor)
> SYM: !=CRIT, X=PROHIB, ?=OPT, →=to, [N]=count

## CORE: DRY
```
!=NO duplicate code
FLOW[5]: SEARCH existing → CHECK doc → REUSE → EXTRACT if 2+ → DOCUMENT
```

---

## 1. REUSABLE COMPONENTS (!=MANDATORY)

### 1.1 Extensions (utils/*Extensions.kt)

**String (StringExtensions.kt)**
```kotlin
fun String.isValidQuestion(): Boolean
fun String.isValidAnswer(): Boolean
```

**Duration (DurationExtensions.kt)**
```kotlin
fun Duration.toKoreanString(): String
```

**GameState (GameStateExtensions.kt)**
```kotlin
fun GameState.canUseHint(): Boolean
fun GameState.remainingHints(): Int
```

### 1.2 Exceptions (utils/Exceptions.kt) !=ALWAYS
```kotlin
// Base
open class TurtleSoupException(message: String, cause: Throwable? = null)

// Domain
class SessionNotFoundException(sessionId: String)
class InvalidQuestionException(message: String)
class GameAlreadyStartedException(sessionId: String)
class GameAlreadySolvedException(sessionId: String)
class MaxHintsReachedException
class PuzzleGenerationException(message: String, cause: Throwable?)
class RedisException(message: String, cause: Throwable?)
class LockException(message: String)

// Usage
throw SessionNotFoundException(sessionId)  // X Exception, X IllegalArgumentException
```

### 1.3 Constants (config/Constants.kt) !=ALWAYS
```kotlin
object GameConstants {
    const val MAX_HINTS = 3
    const val SESSION_TTL_MINUTES = 30
}

object ValidationConstants {
    const val MIN_QUESTION_LENGTH = 2
    const val MAX_QUESTION_LENGTH = 200
}

object RedisKeys {
    private const val APP_PREFIX = "turtle"
    const val SESSION = "$APP_PREFIX:session"
    const val CHAT = "$APP_PREFIX:chat"
    const val LOCK = "$APP_PREFIX:lock"
}

// Usage
if (state.hintsUsed >= GameConstants.MAX_HINTS)  // X magic numbers
```

### 1.4 Redis (redis/)

**SessionStore**
```kotlin
suspend fun saveGameState(state: GameState)
suspend fun loadGameState(sessionId: String): GameState?
suspend fun deleteSession(sessionId: String)
suspend fun sessionExists(sessionId: String): Boolean
suspend fun refreshTtl(sessionId: String): Boolean
```

**LockManager**
```kotlin
suspend fun <T> withLock(sessionId: String, block: suspend () -> T): T

// Extension
suspend fun <T> LockManager.withSessionLock(
    sessionId: String,
    block: suspend () -> T
): T
```

### 1.5 LangChain4j AiServices (llm/)

**GameMasterService** (!=CORE)
```kotlin
interface GameMasterService {
    @SystemMessage("...")
    @UserMessage("Question: {{question}}")
    fun answerQuestion(
        @MemoryId sessionId: String,
        @V("puzzle") puzzle: Puzzle,
        @V("question") question: String
    ): String
    
    fun validateSolution(...): SolutionValidation
    fun generateHint(...): String
}
```

### 1.6 Logging (!=MANDATORY)
```kotlin
import io.github.oshai.kotlinlogging.KotlinLogging

private val logger = KotlinLogging.logger {}

logger.info { "event_name key1=$value1 key2=$value2" }  // X f-strings style
logger.error(exception) { "error_description" }
```

---

## 2. CODE PATTERNS

### 2.1 Data Class (Immutability)
```kotlin
@Serializable
data class GameState(
    val sessionId: String,
    val questions: List<QuestionHistory> = emptyList(),
    val hintsUsed: Int = 0
) {
    init {
        require(hintsUsed >= 0) { "Hints used cannot be negative" }
    }
    
    val questionCount: Int
        get() = questions.size
    
    fun addQuestion(q: String, a: String): GameState = copy(
        questions = questions + QuestionHistory(q, a),
        lastActivityAt = Instant.now()
    )
}

// X var, X mutable collections
```

### 2.2 Service
```kotlin
class GameService(
    private val gameMaster: GameMasterService,  // Constructor injection
    private val sessionStore: SessionStore,
    private val lockManager: LockManager
) {
    companion object {
        private val logger = KotlinLogging.logger {}
        private const val TIMEOUT = 5000L  // Constants in companion
    }
    
    suspend fun startGame(sessionId: String, ...): GameState {
        return lockManager.withSessionLock(sessionId) {
            // Business logic
            logger.info { "game_started session_id=$sessionId" }
            state
        }
    }
}
```

### 2.3 Ktor Routes
```kotlin
fun Application.configureGameRoutes() {
    val gameService: GameService by inject()  // Koin DI
    
    routing {
        route("/api/game") {
            post("/start") {
                try {
                    val req = call.receive<StartGameRequest>()
                    val state = gameService.startGame(...)
                    call.respond(HttpStatusCode.OK, state.toDto())
                } catch (e: GameAlreadyStartedException) {
                    call.respond(HttpStatusCode.Conflict, ErrorResponse(...))
                } catch (e: TurtleSoupException) {
                    logger.error(e) { "start_game_failed" }
                    call.respond(HttpStatusCode.BadRequest, ErrorResponse(...))
                }
            }
        }
    }
}
```

### 2.4 Koin DI
```kotlin
val appModule = module {
    // Settings
    single { Settings.load() }
    
    // Redis
    single { RedisClient.create(...) }
    single { get<RedisClient>().connect() }
    
    // LangChain4j
    single<ChatLanguageModel> {
        GoogleAiGeminiChatModel.builder()
            .apiKey(get<Settings>().gemini.apiKey)
            .build()
    }
    
    single {
        AiServices.builder(GameMasterService::class.java)
            .chatLanguageModel(get())
            .chatMemoryProvider { ... }
            .build()
    }
    
    // Services
    singleOf(::SessionStore)
    singleOf(::LockManager)
    singleOf(::GameService)
}
```

---

## 3. NAMING

- **Files**: PascalCase.kt (GameService.kt, StringExtensions.kt)
- **Packages**: lowercase (io.github.kapu.turtlesoup.service)
- **Classes**: PascalCase (GameService, SessionStore)
- **Functions**: camelCase (startGame, validateSolution)
  - Boolean: is/has/can (isValid, hasPermission, canUseHint)
  - Collections: plural (questions, items)
- **Constants**: SCREAMING_SNAKE_CASE (MAX_QUESTIONS, SESSION_TTL_MINUTES)
- **Properties**: camelCase (questionCount, hintsUsed)
- **Companion object**: companion object (no name)
- **Extensions**: descriptive receiver type (String.isValidQuestion, GameState.canUseHint)

---

## 4. FORBIDDEN (!=CRITICAL)

1. X `var` in data models → `val` + copy()
2. X Mutable collections → immutable (List, Set, Map)
3. X `println()` → logger
4. X Magic numbers/strings → Constants
5. X Blocking I/O in suspend → async libraries (Lettuce coroutines)
6. X String interpolation in logs → structured logging
7. X Emoji in code/logs → UI only
8. X Generic exceptions → TurtleSoupException hierarchy
9. X Manual state management → LangChain4j AiServices
10. X Reflection-based serialization → kotlinx.serialization
11. X Manual DI → Koin
12. X `!!` (not-null assertion) → safe calls or require()

---

## 5. TESTING (Kotest)

```kotlin
import io.kotest.core.spec.style.StringSpec
import io.kotest.matchers.shouldBe
import io.mockk.coEvery
import io.mockk.mockk

class GameServiceTest : StringSpec({
    
    "should start game successfully" {
        // Given
        val gameMaster = mockk<GameMasterService>()
        val sessionStore = mockk<SessionStore>()
        val lockManager = mockk<LockManager>()
        val service = GameService(gameMaster, sessionStore, lockManager)
        
        coEvery { sessionStore.loadGameState(any()) } returns null
        coEvery { sessionStore.saveGameState(any()) } returns Unit
        
        // When
        val state = service.startGame("session-1", "user-1", "chat-1")
        
        // Then
        state.sessionId shouldBe "session-1"
        state.questionCount shouldBe 0
    }
})
```

- Coverage ≥80%
- Kotest StringSpec (BDD style)
- MockK for mocking (coEvery for suspend)

---

## 6. LINTING & FORMATTING (!=MANDATORY)

```bash
# ktlint: code style
./gradlew ktlintCheck
./gradlew ktlintFormat  # Auto-fix

# detekt: static analysis
./gradlew detekt

# All checks
make lint
```

**build.gradle.kts:**
```kotlin
ktlint {
    version.set("1.0.1")
    android.set(false)
}

detekt {
    buildUponDefaultConfig = true
    config.setFrom(files("detekt.yml"))
}
```

**Rules:**
- Max line length: 120
- Indent: 4 spaces
- Trailing commas: allowed
- No wildcard imports

---

## 7. COMMIT CHECKLIST

**Code**
- [ ] X duplicate, use extensions/utils
- [ ] TurtleSoupException hierarchy
- [ ] Constants, no magic numbers
- [ ] Immutable data classes (val + copy)

**Async/Log**
- [ ] All I/O = suspend
- [ ] Structured logging (logger.info { "key=$value" })
- [ ] X blocking calls

**DI/Config**
- [ ] Constructor injection (Koin)
- [ ] Settings from application.conf + .env

**Lint/Test**
- [ ] `make lint` passes (!=MANDATORY)
- [ ] `./gradlew test` coverage ≥80%
- [ ] X emoji

---

## 8. COMMON MISTAKES

1. Implementing existing → Check extensions/utils
2. New exception → Use TurtleSoupException
3. Hardcoded values → Constants
4. Magic numbers → GameConstants/ValidationConstants
5. String interpolation in logs → Structured
6. Blocking in suspend → Async libraries
7. Manual DI → Koin module
8. Mutable state → Immutable data classes
9. println() → logger
10. var in models → val + copy()

---

## 9. SERIALIZATION (kotlinx.serialization)

```kotlin
@Serializable
data class GameState(
    val sessionId: String,
    @Serializable(with = InstantSerializer::class)
    val startedAt: Instant = Instant.now()
)

// Custom serializer
object InstantSerializer : KSerializer<Instant> {
    override val descriptor = PrimitiveSerialDescriptor("Instant", PrimitiveKind.STRING)
    
    override fun serialize(encoder: Encoder, value: Instant) {
        encoder.encodeString(value.toString())
    }
    
    override fun deserialize(decoder: Decoder): Instant {
        return Instant.parse(decoder.decodeString())
    }
}

// JSON config
val json = Json {
    prettyPrint = true
    ignoreUnknownKeys = true
}
```

---

## 10. PERFORMANCE (!=CRITICAL)

### 10.1 Coroutines (Structured Concurrency)
```kotlin
// Suspend functions
suspend fun fetchData(): Data = withContext(Dispatchers.IO) {
    // I/O operations
}

// Parallel execution
suspend fun fetchAll(): List<Data> = coroutineScope {
    val deferred1 = async { fetch1() }
    val deferred2 = async { fetch2() }
    listOf(deferred1.await(), deferred2.await())
}

// X runBlocking in production
```

### 10.2 Collections
```kotlin
// Immutable collections (more efficient)
val list = listOf(1, 2, 3)  // ✓
val mutableList = mutableListOf(1, 2, 3)  // X in data models

// Sequence for large data
val result = items.asSequence()
    .filter { it.isValid }
    .map { it.transform() }
    .toList()

// Standard library (optimized)
questions.count { it.isAnswered }  // ✓
questions.filter { it.isAnswered }.size  // X (creates intermediate list)
```

### 10.3 Redis (Lettuce)
```kotlin
// Coroutines API
val commands: RedisCoroutinesCommands<String, String> = connection.coroutines()
val value = commands.get(key)

// Pipeline for batch
connection.async().use { async ->
    async.setAutoFlushCommands(false)
    val futures = keys.map { async.get(it) }
    async.flushCommands()
    futures.map { it.await() }
}
```

### 10.4 ZGC Configuration
```properties
# gradle.properties
org.gradle.jvmargs=-Xmx2g -XX:+UseZGC

# application.conf
ktor {
    deployment {
        jvmArgs = ["-XX:+UseZGC", "-Xmx2g"]
    }
}
```

---

## 11. LANGCHAIN4J PATTERNS

### 11.1 AiServices Interface
```kotlin
interface GameMasterService {
    @SystemMessage("""
        System instructions here.
        Variables: {{puzzle.scenario}}, {{puzzle.solution}}
    """)
    @UserMessage("User message: {{question}}")
    fun methodName(
        @MemoryId sessionId: String,  // Required for chat memory
        @V("puzzle") puzzle: Puzzle,  // Template variable
        @V("question") question: String
    ): String  // Or structured output (data class)
}
```

### 11.2 ChatMemoryStore (Redis)
```kotlin
// LangChain4j handles serialization automatically
val memoryStore = RedisChatMemoryStore.builder()
    .redisClient(lettuceConnection)
    .ttlSeconds(1800)
    .build()

val chatMemory = MessageWindowChatMemory.builder()
    .chatMemoryStore(memoryStore)
    .id(sessionId)
    .maxMessages(100)
    .build()
```

### 11.3 Structured Output
```kotlin
@Serializable
data class SolutionValidation(
    val isCorrect: Boolean,
    val explanation: String,
    val confidence: Double
)

interface GameMasterService {
    @UserMessage("Validate: {{answer}}")
    fun validateSolution(
        @MemoryId sessionId: String,
        @V("answer") answer: String
    ): SolutionValidation  // LangChain4j auto-parses JSON
}
```

---

## 12. KTOR SPECIFIC

### 12.1 Content Negotiation
```kotlin
install(ContentNegotiation) {
    json(Json {
        prettyPrint = true
        ignoreUnknownKeys = true
    })
}
```

### 12.2 Error Handling
```kotlin
install(StatusPages) {
    exception<TurtleSoupException> { call, cause ->
        logger.error(cause) { "turtle_soup_exception" }
        call.respond(
            HttpStatusCode.BadRequest,
            ErrorResponse(cause::class.simpleName ?: "ERROR", cause.message ?: "")
        )
    }
    
    exception<Throwable> { call, cause ->
        logger.error(cause) { "unhandled_exception" }
        call.respond(
            HttpStatusCode.InternalServerError,
            ErrorResponse("INTERNAL_ERROR", cause.message ?: "Unknown error")
        )
    }
}
```

### 12.3 CORS
```kotlin
install(CORS) {
    allowMethod(HttpMethod.Options)
    allowMethod(HttpMethod.Post)
    allowHeader(HttpHeaders.ContentType)
    anyHost()  // X in production, specify allowed hosts
}
```

---

## FINAL CHECKLIST

**Before Commit:**
1. [ ] X duplicate code (search existing utils/extensions)
2. [ ] Immutable data classes (val + copy)
3. [ ] TurtleSoupException hierarchy
4. [ ] Constants (no magic numbers)
5. [ ] Structured logging
6. [ ] Suspend functions for I/O
7. [ ] Koin DI (constructor injection)
8. [ ] `make lint` passes
9. [ ] `./gradlew test` ≥80% coverage
10. [ ] X emoji in code/logs

**Architecture:**
11. [ ] LangChain4j AiServices for AI logic
12. [ ] Redis ChatMemoryStore for chat history
13. [ ] Lettuce coroutines for Redis I/O
14. [ ] kotlinx.serialization for JSON
15. [ ] Ktor routes for API

**Performance:**
16. [ ] Coroutines for concurrency
17. [ ] Sequence for large collections
18. [ ] Redis pipeline for batch
19. [ ] ZGC configuration
20. [ ] X blocking calls in suspend
