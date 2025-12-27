# Hololive KakaoTalk Bot (Go)

> 홀로라이브 VTuber 스케줄, 정보 검색 및 알림을 제공하는 고성능 카카오톡 봇 (Go 버전)

카카오톡을 통해 홀로라이브 소속 VTuber들의 실시간 방송 현황, 예정된 스케줄, 공식 프로필 정보를 빠르고 편리하게 제공합니다. Go 1.25의 최신 기능과 Valkey 기반의 다층 캐싱 시스템, GORM 기반의 안정적인 데이터 관리를 통해 높은 성능과 확장성을 보장합니다.

## ✨ 주요 기능

-   **실시간 방송 조회 (`!라이브`)**: Holodex API와 연동하여 현재 방송 중인 멤버 확인
-   **스케줄 정보 (`!예정`)**: 향후 24시간 내의 방송 예정 스케줄 조회
-   **멤버 정보 & 검색**: 공식 프로필 데이터 기반 상세 정보 제공 (한국어 번역 포함)
-   **스마트 알림 시스템**:
    -   방송 시작 전 알림 (5분, 15분, 30분 전 등 설정 가능)
    -   개인화된 멤버별 알림 구독/해제 (`!알람`)
-   **관리자 대시보드**: 웹 기반 관리자 패널을 통한 봇 상태 모니터링 및 설정 관리
-   **동적 ACL (접근 제어)**: 카카오톡 채팅방 별 접근 허용/차단 동적 관리
-   **강력한 성능**:
    -   **HTTP/2 (H2C)**: 멀티플렉싱 지원으로 통신 효율 극대화
    -   **Valkey Caching**: API 호출 비용 절감 및 응답 속도 최적화
    -   **ValkeyMQ**: 안정적인 메시지 큐 기반 비동기 처리
    -   **Circuit Breaker**: 외부 API 장애 시 자동 차단 및 복구

## 🛠 기술 스택

이 프로젝트는 최신 Go 생태계와 안정적인 오픈소스를 활용하여 구축되었습니다.

-   **Language**: [Go](https://go.dev/) 1.25.0
-   **Web Framework**: [Gin](https://github.com/gin-gonic/gin) 1.11.0 (High-performance HTTP web framework)
    -   **Protocol**: HTTP/2 Cleartext (H2C) via `golang.org/x/net/http2/h2c`
-   **Database**: PostgreSQL 16+
    -   **ORM**: [GORM](https://gorm.io/) (PostgreSQL Driver)
-   **Cache & MQ**: [Valkey](https://valkey.io/) (Open Source Redis Monitor)
    -   **Client**: `valkey-io/valkey-go`
-   **Logging**: `log/slog` (Go Standard Library)
    -   **Handler**: `lmittmann/tint` (Colorized output), `natefinch/lumberjack` (Log rotation)
-   **Concurrency**: `sourcegraph/conc` (Structured concurrency)
-   **Infrastructure**:
    -   **Messenger**: Iris (카카오톡 연동 미들웨어)
    -   **Monitoring**: Deunhealth (컨테이너 상태 모니터링 및 자동 복구)
    -   **Deployment**: Docker & Docker Compose

## 📂 프로젝트 구조

```
hololive-kakao-bot-go/
├── cmd/
│   ├── bot/                      # Main Bot Entrypoint
│   └── tools/                    # 데이터 관리 및 유틸리티 도구
├── internal/
│   ├── adapter/                  # 메시지 포맷팅 및 외부 인터페이스 어댑터
│   ├── app/                      # 애플리케이션 라이프사이클 및 DI (Manual Injection)
│   ├── bot/                      # 봇 핵심 로직 및 오케스트레이션
│   ├── command/                  # 명령어 핸들러 (!라이브, !예정 등)
│   ├── config/                   # 환경 설정 관리
│   ├── domain/                   # 도메인 모델 정의 (GORM Models)
│   ├── mq/                       # ValkeyMQ 메시지 수신/발신
│   ├── server/                   # HTTP/H2C 서버 및 미들웨어
│   ├── service/                  # 비즈니스 로직 (YouTube, Schedule, Alarm 등)
│   └── platform/                 # 인프라 스트럭처 (DB, Cache 연결 등)
├── data/                         # 임베디드 정적 데이터 (번역된 프로필 등)
├── scripts/                      # 배포 및 실행 스크립트
└── Dockerfile                    # 프로덕션 배포용 Docker 설정
```

## 🚀 시작하기

### 사전 요구사항

-   Go 1.25 이상
-   Valkey (또는 Redis) 서버
-   PostgreSQL 데이터베이스
-   Iris 메신저 서버 (카카오톡 연동용)
-   Holodex API Key

### 로컬 실행 (개발용)

1.  **환경 변수 설정**:
    ```bash
    cp .env.example .env
    # .env 파일을 열어 필요한 설정(API Key, DB 정보 등)을 입력하세요.
    ```

2.  **데이터베이스 초기화**:
    GORM Auto Migration을 통해 테이블이 자동으로 생성됩니다.

3.  **빌드 및 실행**:
    ```bash
    # 의존성 설치
    go mod download

    # 실행
    go run ./cmd/bot
    
    # 또는 스크립트 사용
    ./scripts/start-bots.sh --foreground
    ```

### Docker Compose 배포 (프로덕션)

이 프로젝트는 `docker-compose`를 통한 통합 배포를 권장합니다.

```yaml
# docker-compose.prod.yml 예시 (메인 레포지토리 참조)
services:
  hololive-bot:
    image: hololive-kakao-bot-go:latest
    environment:
      - SERVER_PORT=30001
      - POSTGRES_HOST=postgres
      - CACHE_HOST=valkey-cache
      - MQ_HOST=valkey-mq
    deploy:
      resources:
        limits:
          memory: 512m
    labels:
      deunhealth.restart.on.unhealthy: "true" # 헬스 체크 실패 시 자동 재시작
```

## ⚙️ 환경 변수 설정 (`.env`)

주요 설정 항목은 다음과 같습니다.

| 카테고리 | 변수명 | 설명 | 기본값 |
| :--- | :--- | :--- | :--- |
| **서버** | `SERVER_PORT` | 봇 웹 서버 포트 | `30001` |
| | `ADMIN_PASS_HASH` | 관리자 패널 비밀번호 (Bcrypt 해시) | **필수** |
| | `SESSION_SECRET` | 세션 보안을 위한 시크릿 키 | **필수** |
| | `ADMIN_ALLOWED_IPS` | 관리자 페이지 접근 허용 IP (쉼표 구분) | (전체 허용) |
| **Holodex** | `HOLODEX_API_KEY_1` | Holodex API 키 (여러 개 등록 가능 _1~_5) | **필수** |
| **YouTube** | `YOUTUBE_API_KEY` | YouTube Data API 키 (구독자 수 조회용) | - |
| **Kakao** | `KAKAO_ROOMS` | 봇이 응답할 카카오톡 방 이름 목록 (쉼표 구분) | `홀로라이브 알림방` |
| | `KAKAO_ACL_ENABLED` | ACL(접근 제어) 활성화 여부 | `true` |
| **Iris** | `IRIS_BASE_URL` | Iris 메신저 서버 주소 | `http://localhost:3000` |
| **DB** | `POSTGRES_HOST`, `_PORT`, ... | PostgreSQL 연결 정보 | `localhost`, `5432` |
| **Cache** | `CACHE_HOST`, `_PORT` | Valkey(Redis) 캐시 서버 정보 | `localhost`, `6379` |
| **MQ** | `MQ_HOST`, `_PORT` | ValkeyMQ 서버 정보 | `localhost`, `1833` |
| **Logging** | `LOG_LEVEL` | 로그 레벨 (`debug`, `info`, `warn`, `error`) | `info` |

## 🕹 명령어 목록

봇이 있는 채팅방에서 아래 명령어를 사용할 수 있습니다. (`!` 접두사 기준)

-   **방송 확인**
    -   `!라이브`: 현재 방송 중인 모든 멤버 목록
    -   `!라이브 [멤버명]`: 특정 멤버의 생방송 여부 확인
    -   `!예정`: 향후 24시간 내 예정된 방송 목록
    -   `!멤버 [이름]`: 해당 멤버의 주간 스케줄 확인

-   **정보 조회**
    -   `!정보 [멤버명]`: 멤버 프로필 및 상세 정보 (예: `!정보 미코`)
    -   `!구독자순위`: 멤버들의 구독자 수 및 최근 급상승 순위 (TOP 10)

-   **알림 관리**
    -   `!알람 추가 [멤버명]`: 해당 멤버의 방송 알림 받기
    -   `!알람 제거 [멤버명]`: 해당 멤버의 알림 끄기
    -   `!알람 목록`: 현재 구독 중인 알림 목록 확인
    -   `!알람 초기화`: 모든 알림 설정 초기화

-   **기타**
    -   `!도움말`: 명령어 도움말 확인

## 🛡 관리 및 모니터링

-   **Health Check**: `/health` 엔드포인트를 통해 봇의 상태를 확인할 수 있습니다.
-   **Deunhealth**: 컨테이너가 멈추거나 헬스 체크에 실패하면 `deunhealth`가 자동으로 이를 감지하고 재시작하여 가용성을 유지합니다.
-   **Graceful Shutdown**: 종료 시그널(SIGTERM) 수신 시 진행 중인 작업을 안전하게 마무리하고 종료합니다.

## 📝 라이선스

MIT License. See [LICENSE](LICENSE) for details.
