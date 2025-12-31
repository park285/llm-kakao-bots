---
description: 한국어 주석 작성 규칙 - Korean Comment Style Guide
---

# 주석 작성 규칙 (Comment Style Guide)

## 1. 기본 원칙 (Core Principles)

### 1.1 한국어 작성 필수
모든 주석은 한국어로 작성하는 것을 원칙으로 합니다.

### 1.2 Why 중심의 서술
- 코드가 **'무엇(What)'**을 하는지는 코드로 표현함
- 주석은 **'왜(Why)'** 그렇게 구현했는지(의도, 맥락, 정책)를 설명함

### 1.3 하이브리드 어조 전략
주석의 위치와 목적에 따라 어조를 구분:

| 구분 | 용도 | 어조 | 마침표 |
|------|------|------|--------|
| **문서화 주석** | 외부 공개용 (API, Public Interface) | 친절한 경어체 (~합니다, ~입니다) | **사용** (완전한 문장) |
| **구현부 주석** | 내부 설명용 (Implementation) | 간결한 명사형 (~함, ~음, ~것) | **생략** (개조식 표현) |

> **Note:** 문서화 주석의 마침표는 GoDoc, JSDoc 등 자동화 도구가 문장 경계를 인식하는 기준이 됩니다.

---

## 2. 세부 작성 규칙 (Detailed Rules)

### 2.1 문서화 주석 (Documentation Comments)

**적용 대상:**
- 파일 상단 (Package 설명)
- Class, Struct, Interface 정의
- 함수(메서드) 상단
- Type Definition

**어조:** 경어체 / 해요체 (~합니다, ~입니다)

**형식:** 언어별 표준 문서화 스타일 준수 (GoDoc, JSDoc, JavaDoc, Docstring 등)

**첫 문장 규칙:** 문서화 주석의 **첫 문장은 해당 기능의 핵심 요약**이어야 하며, 반드시 **마침표(.)로 끝나야** 합니다.
> GoDoc, JSDoc 등 자동화 도구는 첫 문장을 Summary로 추출합니다.

#### Go 예시:
```go
// PostgresService: PostgreSQL 데이터베이스 연결 및 GORM 인스턴스를 관리하는 서비스입니다.
// 연결 풀 설정과 헬스 체크를 담당합니다.
type PostgresService struct {
    db     *sql.DB
    gormDB *gorm.DB
}

// NewPostgresService: 주어진 설정을 사용하여 PostgreSQL 연결을 수립하고 서비스를 초기화합니다.
// 연결 풀 설정 및 초기 헬스 체크(Ping)를 수행하며, GORM 인스턴스도 함께 초기화합니다.
//
// Parameters:
//   - cfg: PostgreSQL 접속 정보 (Host, Port, User, Password, Database)
//
// Returns:
//   - *PostgresService: 초기화된 서비스 인스턴스
//   - error: 연결 실패 시 에러
func NewPostgresService(cfg *PostgresConfig) (*PostgresService, error) { ... }
```

#### TypeScript 예시:
```typescript
/**
 * 사용자의 등급에 따른 최종 할인 금액을 계산합니다.
 * 할인 정책 변경(2023.12)에 따라 VIP 등급은 추가 5%가 적용됩니다.
 *
 * @param user - 구매를 진행하는 사용자 객체
 * @param price - 상품의 원래 가격
 * @returns 할인이 적용된 최종 가격 (절사 처리됨)
 */
function calculateFinalPrice(user: User, price: number): number { ... }
```

---

### 2.2 내부 구현 주석 (Inline Comments)

**적용 대상:**
- 함수 내부 로직
- 변수 옆
- 조건문 분기
- 복잡한 알고리즘 설명

**어조:** 명사형 / 개조식 (~함, ~음, ~것, ~처리)

**명사형 어미 구분:**
| 어미 | 용도 | 예시 |
|------|------|------|
| **~함** | 동작/행위 (Action) | 계산함, 호출함, 반환함, 전송함 |
| **~음** | 상태/사실 (State) | 설정됨, 비어있음, 완료됨, 존재함 |

**형식:** `// ` 뒤에 공백 한 칸을 두고 작성

#### 예시:
```go
func calculateFinalPrice(user *User, price int) int {
    // 유효하지 않은 가격인 경우 0원 반환함
    if price < 0 {
        return 0
    }

    // NOTE: DB 부하를 줄이기 위해 로컬 캐시를 우선 조회함
    cachedRate := getCachedDiscountRate(user.Grade)
    
    // VIP 등급 추가 할인 적용함 (정책: 2023.12)
    if user.Grade == GradeVIP {
        cachedRate += 0.05
    }
    
    return int(float64(price) * (1 - cachedRate))
}
```

---

### 2.3 기술 용어 표기

모호함을 없애기 위해 핵심 기술 용어, 클래스명, 변수명, 라이브러리 이름은 **번역하지 않고 영문 그대로** 표기합니다.

| ❌ 피해야 할 표현 | ✅ 권장 표현 |
|------------------|-------------|
| 요청을 보냄 | Request를 전송함 |
| 널 익셉션 방지 | NullPointerException 방지 |
| 컨텍스트 취소됨 | context가 취소됨 |
| 고루틴에서 실행 | goroutine에서 실행함 |
| 제이슨 파싱 | JSON 파싱 |

#### 영문 용어와 한글 조사 결합 규칙

**표기법:** 영문 단어와 한글 조사는 **붙여 씁니다.** (국립국어원 규범 준수)
| ❌ 잘못된 예 | ✅ 올바른 예 |
|-------------|-------------|
| JSON 을 파싱함 | JSON을 파싱함 |
| Server 가 응답함 | Server가 응답함 |

**발음 기준:** 조사는 영문 단어의 **실제 발음**을 기준으로 선택합니다.
| 영문 용어 | 발음 | 조사 선택 |
|----------|------|----------|
| Server | 서버 (모음 끝) | ~는, ~를, ~가 |
| JSON | 제이슨 (자음 끝) | ~은, ~을, ~이 |
| API | 에이피아이 (모음 끝) | ~는, ~를, ~가 |
| Request | 리퀘스트 (자음 끝) | ~은, ~을, ~이 |

---

### 2.4 주석 내 코드 요소 인용

주석 내에서 변수명, 함수명, 값 등을 언급할 때 일반 텍스트와 구분하기 위해 **백틱(\`) 또는 작은따옴표('')**로 감쌉니다.

> **Note:** GoDoc, JSDoc 등 Markdown을 지원하는 문서화 도구에서는 백틱을 권장합니다.

| ❌ 구분 없음 | ✅ 명확한 구분 |
|-------------|---------------|
| user가 nil일 경우 | `user`가 `nil`일 경우 |
| timeout 값 초과 시 | `timeout` 값 초과 시 |
| GetUser 호출함 | `GetUser` 호출함 |

---

### 2.5 특수 태그 활용

협업을 위한 특수 태그는 **대문자**로 표기하며, 설명은 **명사형 한국어**로 작성합니다.

| 태그 | 의미 | 작성 예시 |
|------|------|----------|
| `TODO` | 추후 해야 할 작업 | `// TODO: 성능 이슈로 인해 비동기 처리로 변경 필요` |
| `FIXME` | 수정이 시급한 오류 | `// FIXME: 특수문자 입력 시 정규식 에러 발생함` |
| `HACK` | 임시방편 코드 | `// HACK: 라이브러리 버그 회피를 위해 강제 형변환함` |
| `NOTE` | 주의사항 또는 참고 | `// NOTE: DB 부하 감소를 위해 캐시 우선 조회` |
| `XXX` | 위험하거나 문제있는 코드 | `// XXX: 동시성 이슈 가능성 있음` |
| `PERF` | 성능 관련 주석 | `// PERF: O(n²) → O(n) 최적화 필요` |

---

## 3. 종합 예시 (Complete Example)

```go
// Package notification: 알림 서비스의 핵심 기능을 제공합니다.
// 외부 API를 통한 환율 조회와 DB 동기화를 담당합니다.
package notification

// SyncExchangeRate: 외부 API를 통해 환율 정보를 조회하고 DB에 동기화합니다.
// API 호출 실패 시, 기존에 저장된 최근 환율을 반환합니다.
//
// Parameters:
//   - ctx: context.Context (타임아웃 및 취소 지원)
//   - currencyCode: 조회할 통화 코드 (예: "USD", "KRW")
//
// Returns:
//   - float64: 해당 통화의 현재 환율
//   - error: 조회 실패 시 에러
func (s *Service) SyncExchangeRate(ctx context.Context, currencyCode string) (float64, error) {
    // 입력값 유효성 검증함
    if currencyCode == "" {
        return 0, fmt.Errorf("통화 코드는 필수입니다")
    }

    // NOTE: SLA 규정에 따라 외부 API 실패 시 빠른 Fail-over를 위해 타임아웃을 짧게 설정함
    rate, err := s.externalAPI.FetchRate(ctx, currencyCode, 5*time.Second)
    if err != nil {
        // FIXME: 에러 로그 포맷 통일 필요
        slog.Error("[Sync Error]", slog.String("currency", currencyCode), slog.Any("error", err))
        
        // 에러 발생 시 최신 저장 데이터 반환함 (Fail-over)
        return s.db.GetLatestRate(ctx, currencyCode)
    }
    
    // DB 업데이트 후 값 반환함
    if err := s.db.UpdateRate(ctx, currencyCode, rate); err != nil {
        slog.Warn("DB 업데이트 실패, 조회된 값은 정상 반환함", slog.Any("error", err))
    }
    
    return rate, nil
}
```

---

## 4. Anti-Patterns (피해야 할 패턴)

### ❌ What(무엇) 설명 - 코드가 이미 표현하는 내용
```go
// i를 1씩 증가시킴
i++

// 사용자 목록을 순회함
for _, user := range users {
```

### ✅ Why(왜) 설명 - 의도와 맥락
```go
// 마지막 요소 접근을 위해 인덱스 미리 증가함
i++

// 비활성 사용자 제외를 위해 필터링함 (정책: 30일 미접속)
for _, user := range users {
```

---

## 5. Checklist

주석 작성 시 다음을 확인하세요:

- [ ] 한국어로 작성했는가?
- [ ] 문서화 주석은 경어체(~합니다)와 **마침표**를 사용했는가?
- [ ] 문서화 주석의 **첫 문장이 해당 기능의 핵심 요약**인가?
- [ ] 내부 주석은 명사형(~함, ~음)을 사용하고 **마침표를 생략**했는가?
- [ ] 동작은 ~함, 상태는 ~음으로 구분했는가?
- [ ] 기술 용어는 영문 그대로 표기했는가?
- [ ] 영문 용어와 조사는 **붙여 쓰고**, 발음 기준으로 조사를 선택했는가?
- [ ] 코드 요소(변수명, 함수명)는 백틱 또는 따옴표로 구분했는가?
- [ ] Why(왜)를 설명하고 있는가? (What은 코드로)
- [ ] 특수 태그(TODO, FIXME 등)는 대문자로 표기했는가?