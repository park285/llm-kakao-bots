# Game Bot Go

카카오톡 오픈채팅방에서 동작하는 게임 봇 서비스의 Go 구현체입니다. 고성능과 안정성을 위해 마이크로서비스 아키텍처를 지향하며, LLM(Large Language Model) 서버와 연동하여 다양한 게임 로직을 처리합니다.

##  주요 기능

### 1. TwentyQ (스무고개)
- 사용자가 정답을 맞추거나 봇에게 질문하며 진행하는 추리 게임
- LLM 기반의 유동적인 질문/답변 처리
- 세션 관리 및 게임 상태 유지

### 2. Turtlesoup (바다거북 수프)
- 상황 설명을 듣고 질문을 통해 진상을 파악하는 수평적 사고 퍼즐 게임
- 복잡한 시나리오 처리 및 힌트 시스템 제공

### 3. 관리자 대시보드 (Admin UI)
- 게임 상태 모니터링 및 제어
- Docker 컨테이너 및 시스템 상태 관리 (Watchdog)
- 로그 확인 및 사용자 관리

## 🛠 기술 스택

- **Language**: Go 1.25+ (Experimental Features: GreenTea GC 활용 검토)
- **Database**: Valkey (Redis 대체, 고성능 Key-Value Store)
- **Web Framework**: Gin (경량화된 웹 프레임워크)
- **Architecture**: Clean Architecture, DI (Dependency Injection)
- **Infrastructure**: Docker, Docker Compose

## 📂 프로젝트 구조

```
game-bot-go/
├── cmd/                # 메인 애플리케이션 진입점
├── internal/           # 비공개 애플리케이션 및 라이브러리 코드
│   ├── common/         # 공통 유틸리티 (Valkey, Config, LLM Client 등)
│   ├── twentyq/        # 스무고개 게임 로직
│   └── turtlesoup/     # 바다거북 수프 게임 로직
├── logs/               # 애플리케이션 로그
└── Dockerfile.prod     # 프로덕션 배포용 Docker 설정
```

##  시작하기

### 필수 요구사항
- Go 1.24 이상
- Docker & Docker Compose
- Valkey (또는 Redis)

### 로컬 실행
```bash
# 의존성 설치
go mod download

# 서버 실행
go run ./cmd/server
```

### Docker 실행
```bash
docker compose up -d --build
```

##  설정

주요 설정은 환경 변수 또는 `config.yaml` 파일을 통해 관리됩니다.
- `VALKEY_HOST`: Valkey 서버 주소
- `LLM_SERVER_URL`: LLM 추론 서버 주소
- `API_KEY`: 내부 API 보안 키

