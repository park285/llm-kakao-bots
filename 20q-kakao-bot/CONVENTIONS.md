# 20Q-KAKAO-BOT CONVENTIONS
> SYM: !=CRIT, X=PROHIB, ?=OPT, →=to, ∴=result, [N]=count
## CORE: DRY!=ABSOLUTE
FLOW[5]: SEARCH existing → CHECK doc → REUSE → EXTRACT if 2+ → DOCUMENT

---
## 1. REUSABLE UTILS!=MANDATORY (src/main/kotlin/party/qwer/twentyq/util/)
### Extensions
```kotlin
// String (common/extensions/StringExtensions.kt)
text.isValidQuestion(), text.normalizeForComparison()
userId.maskSensitive(), apiKey.maskToken()
text.toCategoryIcon(), text.parseYesNo(), template.fillTemplate(mapOf("key" to val))

// Duration (common/extensions/DurationExtensions.kt)
5.seconds, 10.minutes, 24.hours, 7.days  // X raw ms!
duration.toKoreanString(), elapsed isLongerThan 30.seconds

// List (common/extensions/ListSafetyExtensions.kt)
list.takeUpTo(10), list.dropUpTo(5), list.takeLastUpTo(3)
list.safeGet(5), list.getOrDefault(5, default)  // X throw
list.nullIfEmpty(), list.chunkedSafely(10)

// Session (game/extensions/SessionStateExtensions.kt) [10 funcs]
session.isExpired(max), session.isActive(max), session.remainingQuestions(max)
session.canUseHint(max), session.remainingHints(max)
session.progress(max), session.progressPercent(max)
session.hasCategory(), session.hasNoCategory(), session.isCategory(cat)

// Message (game/extensions/MessageContextExtensions.kt)
context.displayName(anonymousName)
```
### Core Components
```kotlin
// Exception (service/exception/GameException.kt)
throw SessionNotFoundException(), HintLimitExceededException()
throw DuplicateQuestionException(), InvalidQuestionException()
// X RuntimeException!

// Cache (cache/CacheBuilders.kt)
private val cache = CacheBuilders.expireAfterWrite<K, V>(
    maxSize = 10_000L, ttl = 5.minutes, recordStats = true
)
// X ConcurrentHashMap!

// Lock (redis/LockCoordinator.kt)
lockCoordinator.withLock(chatId, userId, requiresWrite = true) {
    // critical section
}

// Message Provider (game/GameMessageProvider.kt)
messageProvider.get("start.ready", "name" to userName)
// X "하드코딩된 메시지"

// JSON (common/json/)
JsonResponseParser.parseToType<T>(llmOutput)
MarkdownJsonExtractor.extractJsonFromMarkdown(markdown)

// Security (common/security/Guards.kt)
val eval = guard.evaluate(userInput)
if (eval.malicious) throw InvalidQuestionException()
requireSessionOrThrow(session, chatId)

// Formatting (common/formatting/UserIdFormatter.kt)
UserIdFormatter.displayName(userId, sender, chatId, anonymousName)
```
### Constants (!=ALWAYS use, X magic numbers!)
```kotlin
GameConstants.MAX_QUESTIONS              // game/constants/
ValidationConstants.KAKAO_MESSAGE_MAX_LENGTH
AIConstants.MAX_ANSWER_TOKENS            // ai/
HttpConstants.HTTP_RETRY_ATTEMPTS        // http/
LoggingConstants.LOG_SAMPLE_LIMIT_HIGH   // logging/
RedisKeys.SESSION                        // redis/
```
### Logging Patterns
```kotlin
// Structured (!=MANDATORY): log.info("EVENT chatId={}, userId={}", chatId, userId)
// Lazy: log.debugL { "Result: ${expensiveOp()}" }
// Sampled: log.sampled("key", limit=10, windowMillis=1000L) { it.debug("MSG") }
// X log.info("User $userId started")  // NO string interpolation!
```

---
## 2. PATTERNS
### Service Structure
```kotlin
@Service
class XxxService(private val dep: Dep) {  // !=constructor injection
    companion object {
        private val log = LoggerFactory.getLogger(XxxService::class.java)
        private const val TIMEOUT_MS = 5000L
    }
    suspend fun publicMethod() { }  // public first, private last
}
```
### Immutability & Async
```kotlin
data class RiddleSession(val count: Int = 0)
fun RiddleSession.increment() = copy(count = count + 1)  // X mutate!

suspend fun processAnswer(): Result  // !=ALL async MUST be suspend
coroutineScope { launch { task() } }  // parallel
bucket.get().awaitSingleOrNull()  // Reactor→Coroutine
```

---
## 3. REST API (Level 2)
```
복수형 리소스: /riddles (X /riddle), HTTP 메서드=액션 (X /create, /update)
POST /riddles, GET /riddles, POST /riddles/hints, POST /riddles/answers
```

---
## 4. NAMING
**Files**: PascalCase.kt, **Funcs**: verbNoun (processVote), is/has/can (bool), to* (convert)
**Kotlin**: X I-prefix, X get/set (use properties), **KDoc**: 간결한 한국어 (X English)

---
## 5. FORBIDDEN!=CRITICAL
X ConcurrentHashMap → CacheBuilders | X Field injection → Constructor | X Magic numbers → Constants
X Blocking in suspend → awaitSingleOrNull() | X String interpolation in logs → {}
X Emoji in code/commits → UI only | X Hardcoded messages → GameMessageProvider
X Mutating data classes → .copy() | X Excessive @Suppress → Refactor | X English KDoc → Korean
X kotlinx.serialization → Jackson 3 | X Jackson 2 core (com.fasterxml.jackson.core/databind) → Jackson 3 (tools.jackson.*) | ?Jackson 2 annotations (com.fasterxml.jackson.annotation) OK

---
## 6. TESTING
```kotlin
@Test fun `should return true for valid Korean question`()
coEvery { repo.get(any()) } returns result
coVerify(exactly = 1) { repo.save(any()) }
// Coverage ≥80%, ./gradlew detekt passes
```

---
## 7. CHECKLIST
- [ ] X duplicate, use extensions+constants, GameException, CacheBuilders, GameMessageProvider
- [ ] suspend for async, structured logging {}, sampled in hot paths
- [ ] ./gradlew detekt passes, ./gradlew test passes, coverage≥80%
