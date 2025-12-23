# go-redis → valkey-go 마이그레이션 계획

## 개요

game-bot-go 프로젝트의 Redis 클라이언트를 `github.com/redis/go-redis/v9`에서 `github.com/valkey-io/valkey-go`로 전환.

**목표**:
- mcp-llm-server-go와 통일된 Redis 클라이언트
- RESP3 프로토콜 및 클라이언트 사이드 캐싱 지원
- 코드 재사용성 향상

---

## 영향 범위 분석

### 파일 수: ~47개

| 카테고리 | 파일 수 | 주요 파일 |
|---------|--------|----------|
| Core 클라이언트 | 3 | `redisx/client.go`, `di/redis.go`, `bootstrap/redis.go` |
| MQ (Streams) | 5 | `mq/streams.go`, `mq/publisher.go`, `mq/factory.go` 등 |
| twentyq/redis | 18 | `session_store.go`, `lock_manager.go` 등 |
| turtlesoup/redis | 10 | 유사 구조 |
| common/pending | 2 | `store.go`, `store_test.go` |
| 테스트 파일 | ~15 | miniredis 사용 중 |

---

## API 변환 가이드

### 기본 명령어

| go-redis | valkey-go |
|----------|-----------|
| `client.Set(ctx, key, val, ttl).Err()` | `client.Do(ctx, client.B().Set().Key(key).Value(val).Ex(ttl).Build()).Error()` |
| `client.Get(ctx, key).Result()` | `client.Do(ctx, client.B().Get().Key(key).Build()).ToString()` |
| `client.Get(ctx, key).Bytes()` | `client.Do(ctx, client.B().Get().Key(key).Build()).AsBytes()` |
| `client.Del(ctx, keys...).Err()` | `client.Do(ctx, client.B().Del().Key(keys...).Build()).Error()` |
| `client.Exists(ctx, key).Result()` | `client.Do(ctx, client.B().Exists().Key(key).Build()).AsInt64()` |
| `client.Expire(ctx, key, ttl).Result()` | `client.Do(ctx, client.B().Expire().Key(key).Seconds(int64(ttl.Seconds())).Build()).AsBool()` |
| `client.Ping(ctx).Err()` | `client.Do(ctx, client.B().Ping().Build()).Error()` |

### List 명령어

| go-redis | valkey-go |
|----------|-----------|
| `client.RPush(ctx, key, val).Err()` | `client.Do(ctx, client.B().Rpush().Key(key).Element(val).Build()).Error()` |
| `client.LRange(ctx, key, 0, -1).Result()` | `client.Do(ctx, client.B().Lrange().Key(key).Start(0).Stop(-1).Build()).AsStrSlice()` |
| `client.LTrim(ctx, key, start, stop).Err()` | `client.Do(ctx, client.B().Ltrim().Key(key).Start(start).Stop(stop).Build()).Error()` |

### Hash 명령어

| go-redis | valkey-go |
|----------|-----------|
| `client.HSet(ctx, key, field, val).Err()` | `client.Do(ctx, client.B().Hset().Key(key).FieldValue().FieldValue(field, val).Build()).Error()` |
| `client.HGet(ctx, key, field).Result()` | `client.Do(ctx, client.B().Hget().Key(key).Field(field).Build()).ToString()` |
| `client.HGetAll(ctx, key).Result()` | `client.Do(ctx, client.B().Hgetall().Key(key).Build()).AsStrMap()` |
| `client.HIncrBy(ctx, key, field, incr).Result()` | `client.Do(ctx, client.B().Hincrby().Key(key).Field(field).Increment(incr).Build()).AsInt64()` |

### Set 명령어

| go-redis | valkey-go |
|----------|-----------|
| `client.SAdd(ctx, key, members...).Err()` | `client.Do(ctx, client.B().Sadd().Key(key).Member(members...).Build()).Error()` |
| `client.SMembers(ctx, key).Result()` | `client.Do(ctx, client.B().Smembers().Key(key).Build()).AsStrSlice()` |
| `client.SRem(ctx, key, members...).Err()` | `client.Do(ctx, client.B().Srem().Key(key).Member(members...).Build()).Error()` |
| `client.SCard(ctx, key).Result()` | `client.Do(ctx, client.B().Scard().Key(key).Build()).AsInt64()` |

### Stream 명령어 (MQ용)

| go-redis | valkey-go |
|----------|-----------|
| `client.XAdd(ctx, &redis.XAddArgs{...}).Result()` | `client.Do(ctx, client.B().Xadd().Key(key).Id("*").FieldValue().FieldValue(k, v).Build()).ToString()` |
| `client.XReadGroup(ctx, &redis.XReadGroupArgs{...}).Result()` | 별도 helper 필요 (복잡) |
| `client.XAck(ctx, stream, group, ids...).Err()` | `client.Do(ctx, client.B().Xack().Key(stream).Group(group).Id(ids...).Build()).Error()` |
| `client.XGroupCreateMkStream(ctx, stream, group, start).Err()` | `client.Do(ctx, client.B().XgroupCreate().Key(stream).Group(group).Id(start).Mkstream().Build()).Error()` |
| `client.XGroupDestroy(ctx, stream, group).Err()` | `client.Do(ctx, client.B().XgroupDestroy().Key(stream).Group(group).Build()).Error()` |

### Nil 체크

| go-redis | valkey-go |
|----------|-----------|
| `errors.Is(err, redis.Nil)` | `valkey.IsValkeyNil(err)` |

---

## 마이그레이션 단계

### Phase 1: 핵심 인프라 (우선순위 P0)

- [ ] 1.1 `go.mod`에 `valkey-io/valkey-go` 추가
- [ ] 1.2 `internal/common/valkeyx/client.go` 생성
- [ ] 1.3 `internal/common/di/valkey.go` 생성
- [ ] 1.4 `internal/common/bootstrap/valkey.go` 생성

### Phase 2: twentyq 스토어 전환 (P0)

- [ ] 2.1 `twentyq/redis/session_store.go` → valkey
- [ ] 2.2 `twentyq/redis/history_store.go` → valkey
- [ ] 2.3 `twentyq/redis/player_store.go` → valkey
- [ ] 2.4 `twentyq/redis/hint_count_store.go` → valkey
- [ ] 2.5 `twentyq/redis/category_store.go` → valkey
- [ ] 2.6 `twentyq/redis/wrong_guess_store.go` → valkey
- [ ] 2.7 `twentyq/redis/topic_history_store.go` → valkey
- [ ] 2.8 `twentyq/redis/surrender_vote_store.go` → valkey
- [ ] 2.9 `twentyq/redis/pending_message_store.go` → valkey
- [ ] 2.10 `twentyq/redis/lock_manager.go` → valkey
- [ ] 2.11 `twentyq/redis/processing_lock.go` → valkey

### Phase 3: turtlesoup 스토어 전환 (P0)

- [ ] 3.1 `turtlesoup/redis/session_store.go` → valkey
- [ ] 3.2 `turtlesoup/redis/lock_manager.go` → valkey
- [ ] 3.3 `turtlesoup/redis/processing_lock.go` → valkey
- [ ] 3.4 `turtlesoup/redis/pending_message_store.go` → valkey
- [ ] 3.5 `turtlesoup/redis/surrender_vote_store.go` → valkey
- [ ] 3.6 `turtlesoup/redis/puzzle_dedup_store.go` → valkey

### Phase 4: common 모듈 전환 (P0)

- [ ] 4.1 `common/pending/store.go` → valkey
- [ ] 4.2 `common/processinglock/service.go` → valkey

### Phase 5: MQ (Streams) 전환 (P1)

- [ ] 5.1 `common/mq/streams.go` → valkey (XReadGroup 복잡)
- [ ] 5.2 `common/mq/publisher.go` → valkey
- [ ] 5.3 `common/mq/factory.go` → valkey
- [ ] 5.4 `common/mq/reply_publisher.go` → valkey
- [ ] 5.5 `common/mq/stream_message_handler.go` → valkey

### Phase 6: 정리 및 테스트 (P1)

- [ ] 6.1 `go.mod`에서 `go-redis/v9` 제거
- [ ] 6.2 `internal/common/redisx/` 삭제
- [ ] 6.3 테스트 업데이트 (miniredis → valkey mock)
- [ ] 6.4 전체 테스트 실행 및 검증
- [ ] 6.5 `go mod tidy`

---

## 테스트 전략

### miniredis 호환성 문제

`alicebob/miniredis`는 RESP2만 지원하므로 valkey-go의 RESP3와 호환 불가.

**해결 방안**:
1. **DisableCache=true 옵션**: 테스트 시 클라이언트 캐싱 비활성화
2. **valkey-go의 RESP2 호환 모드**: `DisableCache: true` 설정으로 miniredis 사용 가능
3. 테스트용 helper 함수 생성

---

## 예상 일정

| Phase | 예상 시간 |
|-------|----------|
| Phase 1 | 0.5일 |
| Phase 2 | 1일 |
| Phase 3 | 0.5일 |
| Phase 4 | 0.5일 |
| Phase 5 | 0.5일 |
| Phase 6 | 0.5일 |
| **합계** | **3.5일** |

---

*생성일: 2025-12-23*
