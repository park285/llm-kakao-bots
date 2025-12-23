# Hololive KakaoTalk Bot (Go)

> 홀로라이브 VTuber 스케줄 및 정보 제공 카카오톡 봇 (Go 버전)

카카오톡을 통해 홀로라이브 소속 VTuber들의 스케줄, 프로필 정보를 제공하는 봇입니다.

## 주요 기능

- **스케줄 조회**: Holodex API 연동으로 실시간 방송 및 예정 스케줄 확인
- **멤버 정보**: 공식 프로필 데이터 제공 (사전 번역 데이터 내장)
- **알림 설정**: 사용자별 멤버 알림 관리
- **캐싱**: Valkey 기반 고성능 캐싱
- **Circuit Breaker**: AI API 장애 대응

## 빠른 시작

### 요구사항

- Go 1.24+
- Valkey 서버
- Iris Messenger 서버

### 빌드

```bash
cd /home/kapu/gemini/hololive-kakao-bot-go
CGO_ENABLED=0 go build -tags go_json -o bin/bot ./cmd/bot  # Main Bot
```

### 실행

```bash
# .env 파일 생성 (템플릿 참고)
cp .env.example .env
nano .env  # API 키 및 설정 입력

# Bot 실행 (foreground)
./scripts/start-bots.sh --foreground

# Bot 실행 (background)
./scripts/start-bots.sh
```

스크립트는 `logs/` 디렉터리에 표준 출력과 오류를 기록하고, 백그라운드 실행 시 PID를 저장합니다. 종료는 `./scripts/stop-bots.sh`를 사용하세요.

### 운영 스크립트

```bash
./scripts/start-bots.sh   # Bot 시작 (background)
./scripts/restart-bots.sh # Bot 재시작
./scripts/stop-bots.sh    # Bot 종료
./scripts/status-bots.sh  # 서비스 상태 및 의존성 확인
```

## 프로젝트 구조

```
hololive-kakao-bot-go/
├── cmd/
│   ├── bot/                      # Main Bot (HTTP webhook + alarm + YouTube)
│   └── tools/                    # 데이터 관리 도구
│       ├── fetch_profiles/       # 공식 프로필 fetch
│       └── warm_member_cache/    # 멤버 캐시 워밍업
├── internal/
│   ├── adapter/                  # 메시지 포맷팅
│   ├── bot/                      # 봇 오케스트레이션
│   ├── command/                  # 명령어 핸들러
│   ├── config/                   # 환경 설정
│   ├── constants/                # 상수 정의
│   ├── domain/                   # 도메인 모델
│   ├── iris/                     # Iris Messenger 클라이언트
│   ├── service/                  # 비즈니스 로직
│   ├── platform/                 # 공통 부트스트랩/초기화 로직
│   ├── mq/                       # Message Queue (ValkeyMQ)
│   └── util/                     # 헬퍼 함수
├── data/                         # 임베디드 정적 데이터
│   ├── members.json              # 멤버 정보
│   ├── official_profiles/*.json  # 공식 프로필
│   └── official_translated/*.json# 번역된 프로필
├── scripts/                      # 운영 스크립트
└── bin/                          # 빌드된 바이너리 (gitignored)
    └── bot
```

## 환경 변수 설정

`.env.example`을 기준으로 `.env` 파일 또는 시스템 환경 변수로 설정:

```env
# Iris Server Configuration
IRIS_BASE_URL=http://localhost:3000

# Bot Webhook Server Configuration
SERVER_PORT=30001

# Admin Panel Credentials
ADMIN_USER=admin
ADMIN_PASS=change_this_password

# KakaoTalk
# 알림을 받을 카카오톡 방 이름들 (쉼표로 구분)
KAKAO_ROOMS=홀로라이브 알림방

# Holodex API (여러 키 로테이션 지원)
HOLODEX_API_KEY_1=your_holodex_api_key_here
HOLODEX_API_KEY_2=
HOLODEX_API_KEY_3=
HOLODEX_API_KEY_4=
HOLODEX_API_KEY_5=

# Valkey - Cache
CACHE_HOST=localhost
CACHE_PORT=6379
CACHE_PASSWORD=
CACHE_DB=0

# Valkey - MQ (카카오 메시지 라우팅용, Docker 포트 1833 사용)
MQ_HOST=localhost
MQ_PORT=1833
MQ_PASSWORD=
MQ_STREAM_KEY=kakao:hololive
MQ_CONSUMER_GROUP=hololive-bot-group
MQ_CONSUMER_NAME=consumer-1

# Notification Settings
NOTIFICATION_ADVANCE_MINUTES=5,15,30
CHECK_INTERVAL_SECONDS=60

# Logging
LOG_LEVEL=info
LOG_FILE=logs/bot.log

# Bot Command Prefix
BOT_PREFIX=!
BOT_SELF_USER=iris
```

## 지원 명령어

📺 방송 확인
  !라이브 - 현재 라이브 중인 방송
  !라이브 [멤버명] - 특정 멤버 라이브 확인
  !예정 - 예정된 방송 (24시간 기준)
  !멤버 [이름] [일수] - 특정 멤버 일정 (기본 7일)

👤 멤버 정보
  !정보 [멤버명] - 멤버 프로필 조회
  예: "!미코 정보", "!아쿠아에 대해 알려줘"

🔔 알람 설정
  !알람 추가 [멤버명]
  !알람 제거 [멤버명]
  !알람 목록
  !알람 초기화

📊 통계
  !구독자순위 - 최근 10일간 구독자 증가 순위 TOP 10
  !구독자순위 [기간]
  자동 알림: 마일스톤 달성 시 (10만, 100만, 500만 등)

❓ 도움말
  !도움말 - 전체 명령어 요약 확인

## 기술 스택

- **언어**: Go 1.24
- **캐시**: Valkey
- **로깅**: Uber Zap
- **메신저**: Iris (KakaoTalk 연동)
- **데이터**: Holodex API

## 아키텍처 특징

- **Rate Limiting**: Holodex API 요청 제한 준수
- **캐싱 전략**: Valkey 다층 캐싱으로 API 호출 최소화
- **임베디드 데이터**: 공식 프로필 정적 데이터 내장

## 테스트

```bash
# 전체 테스트 실행
go test ./internal/...

# 특정 패키지 테스트
go test ./internal/domain -v

# 커버리지 확인
go test -cover ./internal/...
```

## 라이선스

Private Repository

---
