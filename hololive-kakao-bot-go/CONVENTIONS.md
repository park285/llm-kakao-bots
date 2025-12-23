# Hololive Kakao Bot Go - Claude 참조 가이드

INHERIT: /home/kapu/gemini/CLAUDE.md (L0-L3)

## Architecture

**Binary**:
- `cmd/bot`: Main bot (HTTP webhook + alarm checker + YouTube scheduler)

## 1. 유틸리티 함수 위치 및 사용법

### 문자열 (internal/util/string.go)
```go
util.Normalize(s)              // 소문자 + trim
util.NormalizeSuffix(s)        // "짱", "쨩" 제거
util.TruncateString(s, maxLen) // rune 기준 자르기
util.NormalizeKey(s)           // 특수문자 제거
```

### 숫자 (internal/util/math.go)
```go
util.FormatKoreanNumber(12345)  // "1만 2345" [자주 사용!]
util.Min(a, b)
util.Max(a, b)
```

### 시간 (internal/util/time.go)
```go
util.FormatKST(t, "01/02 15:04")  // KST 포맷
util.ToKST(t)                      // UTC → KST
util.MinutesUntilCeil(target, now) // 남은 분(올림)
```

### 에러 타입 (pkg/errors/errors.go)
```go
errors.NewAPIError(msg, statusCode, context)
errors.NewCacheError(msg, operation, key, cause)
errors.NewServiceError(msg, service, operation, cause)
errors.NewValidationError(msg, field, value)
errors.NewKeyRotationError(msg, statusCode, context)

// 에러 체크
import stdErrors "errors"
valkey.IsValkeyNil(err)
stdErrors.As(err, &apiErr)
```

## 2. 서비스 메서드 [자주 사용]

### CacheService (internal/service/cache)
```go
cache.Get(ctx, key, &dest)           // JSON auto unmarshal
cache.Set(ctx, key, value, ttl)
cache.GetStreams(ctx, key)           // context 필수!
cache.SetStreams(ctx, key, data, ttl)
cache.SAdd/SRem/SMembers/SIsMember   // Set 연산
cache.HSet/HGet/HMSet/HGetAll        // Hash 연산
```

### Holodex (internal/service/holodex)
```go
holodex.GetLiveStreams(ctx)
holodex.GetUpcomingStreams(ctx, hours)
holodex.GetChannelSchedule(ctx, channelID, hours, includeLive)
holodex.SearchChannels(ctx, query)
holodex.GetChannel(ctx, channelID)
```

### Alarm (internal/service/notification)
```go
alarm.AddAlarm(ctx, roomID, userID, channelID, memberName)
alarm.RemoveAlarm(ctx, roomID, userID, channelID)
alarm.GetUserAlarms(ctx, roomID, userID)
alarm.CheckUpcomingStreams(ctx)
alarm.GetNextStreamInfo(ctx, channelID)
```

### Formatter (internal/adapter)
```go
// 스트림 포맷팅
formatter.FormatLiveStreams(streams)
formatter.UpcomingStreams(streams, hours)
formatter.ChannelSchedule(channel, streams, days)

// 알림 메시지
formatter.FormatAlarmAdded(memberName, added, nextStreamInfo)
formatter.FormatAlarmRemoved(memberName, removed)
formatter.FormatAlarmList(alarms)  // []AlarmListEntry
formatter.FormatAlarmCleared(count)
formatter.AlarmNotification(notification)
formatter.AlarmNotificationGroup(minutesUntil, notifications)

// 멤버 정보
formatter.MemberDirectory(groups, total)  // []MemberDirectoryGroup
formatter.MemberNotFound(memberName)
formatter.FormatTalentProfile(rawProfile, translated)

// 기타
formatter.FormatHelp()
formatter.FormatError(message)
```

## 3. 상수 위치

### TTL (internal/constants)
```go
constants.CacheTTL.LiveStreams      // 5분
constants.CacheTTL.UpcomingStreams  // 5분
constants.CacheTTL.ChannelSchedule  // 5분
constants.CacheTTL.ChannelInfo      // 20분
constants.CacheTTL.NextStreamInfo   // 60분
```

### API (internal/constants)
```go
constants.APIConfig.HolodexBaseURL    // "https://holodex.net/api/v2"
constants.APIConfig.HolodexTimeout    // 10초
constants.RetryConfig.MaxAttempts     // 3
constants.CircuitBreakerConfig.FailureThreshold  // 3
```

## 4. 표준 패턴

### 캐시 사용
```go
var cached DataType
if err := cache.Get(ctx, key, &cached); err == nil && cached != nil {
	return cached, nil
}
data, err := fetchData(ctx)
_ = cache.Set(ctx, key, data, constants.CacheTTL.SomeData)
return data, nil
```

### Helper 추출 시점
- 함수 100줄 초과
- Complexity > 20
- 동일 로직 2번 이상

### 필수 Import Alias
```go
import stdErrors "errors"  // pkg/errors와 구분 필수!
```

## 5. 필수 규칙 (MANDATORY)

### 5.1 문자열 정규화 [CRITICAL]

**금지**: 직접 구현
```go
strings.ToLower(strings.TrimSpace(s))  // BAD
strings.TrimSpace(s)                   // BAD (util.TrimSpace / util.Normalize 사용)
strings.ToLower(s)                     // BAD (util.Normalize 사용)
```

**필수**: util 함수 사용
```go
util.Normalize(s)              // GOOD - 소문자 + trim
util.TrimSpace(s)              // GOOD - trim only (대소문자 보존)
util.TruncateString(s, max)    // GOOD - rune 기준 자르기
```

**가이드**: 값 성격에 따라 선택
- 사용자 입력 비교/키 생성: util.Normalize / util.NormalizeKey
- 케이스 보존이 필요한 값(API 키/식별자/URL 등): util.TrimSpace

**예외**: 특수한 케이스만 처리가 필요할 때
```go
strings.HasPrefix(...)   // OK
strings.Contains(...)    // OK
strings.Split(...)       // OK
```

### 5.2 하드코딩 금지 [CRITICAL]

**금지**: 상수, 타임아웃, 메시지 하드코딩
```go
const streamKey = "kakao:bot:reply"           // BAD
ConnWriteTimeout: 3 * time.Second,            // BAD
BlockingPoolSize: 50,                         // BAD
Block(5000)                                   // BAD
return "멤버를 찾을 수 없습니다."                // BAD (한국어 메시지)
```

**필수**: constants 또는 config 사용
```go
constants.MQ.ReplyStreamKey                   // GOOD
constants.MQ.ConnWriteTimeout                 // GOOD
constants.MQ.BlockingPoolSize                 // GOOD
constants.MQ.BlockTimeout                     // GOOD
formatter.MemberNotFound(memberName)          // GOOD (메시지 함수화)
```

### 5.3 중복 코드 제거 [HIGH]

**금지**: 동일 로직 2회 이상 반복
```go
// BAD - 두 곳에서 동일한 클라이언트 생성 코드
client, err := valkey.NewClient(valkey.ClientOption{...})
```

**필수**: 헬퍼 함수 추출
```go
// GOOD
func newValkeyClient(cfg ValkeyConfig) (valkey.Client, error) {
    // 공통 로직
}
```

### 5.4 Valkey/MQ 원자성 보장 [CRITICAL]

**금지**: 메시지 처리와 ACK 분리
```go
// BAD - 처리 성공 후 ACK 실패 가능
bot.HandleMessage(ctx, msg)
if err := ackMessage(ctx, msgID); err != nil {
    logger.Error("ACK failed")  // 메시지 중복 처리 위험
}
```

**필수**: 처리 실패 시 ACK 스킵 또는 멱등성 보장
```go
// GOOD - 처리 성공 시에만 ACK
if err := bot.HandleMessage(ctx, msg); err != nil {
    logger.Error("Handle failed", zap.Error(err))
    return // ACK 하지 않음, 재시도
}
// 처리 성공 시에만 ACK
if err := ackMessage(ctx, msgID); err != nil {
    logger.Warn("ACK failed but message processed", zap.Error(err))
}
```

또는 멱등성 키 사용:
```go
// GOOD - 중복 처리 방지
idempotencyKey := fmt.Sprintf("processed:%s", msg.ID)
if cache.Exists(ctx, idempotencyKey) {
    _ = ackMessage(ctx, msgID)
    return // 이미 처리됨
}

if err := bot.HandleMessage(ctx, msg); err != nil {
    return err
}

cache.Set(ctx, idempotencyKey, true, 24*time.Hour)
_ = ackMessage(ctx, msgID)
```

## 6. 금지 패턴 (PROHIBITED)

### fmt.Sprintf 남용
```go
// BAD
key := fmt.Sprintf("cache:%s", id)
url := fmt.Sprintf("https://example.com/%s", path)

// GOOD - 상수 정의 또는 헬퍼 함수
const cacheKeyPrefix = "cache:"
key := cacheKeyPrefix + id

func buildURL(path string) string {
    return constants.API.BaseURL + path
}
```

### 에러 메시지에 변수값 노출
```go
// BAD - 민감 정보 노출 위험
return fmt.Errorf("user %s not found", userID)

// GOOD - 로그에만 기록, 에러는 일반화
logger.Error("User not found", zap.String("user_id", userID))
return errors.NewValidationError("user not found", "user_id", "")
```

---

**Updated**: 2025-11-21 | **Lint**: OK 0 issues
