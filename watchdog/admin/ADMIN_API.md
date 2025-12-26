# Watchdog Admin API (Backend)

이 문서는 `/home/kapu/gemini/llm/watchdog` 워치독 프로세스에 내장된 **관리용 HTTP API**(프론트엔드가 호출할 API) 스펙입니다.

## 0. 보안 전제 (Cloudflare Tunnel + Cloudflare Access)

- Admin 서버는 기본적으로 `127.0.0.1`에 바인딩해서 **로컬 전용**으로 실행하는 것을 전제로 합니다.
- 외부 서빙은 Cloudflare Tunnel로만 하고, Access 정책으로 인증/인가를 처리합니다.
- API는 **세션/로그인 없이** Cloudflare Access JWT(`Cf-Access-Jwt-Assertion`)를 검증합니다.

## 1. 환경 변수

### 1.1 Admin 서버

- `WATCHDOG_ADMIN_ENABLED` (bool, default: `false`): Admin API 서버 실행 여부
- `WATCHDOG_ADMIN_ADDR` (string, default: `127.0.0.1:30002`): 바인딩 주소
- `WATCHDOG_ADMIN_H2C` (bool, default: `true`): HTTP/2 cleartext(h2c) 활성화

Timeouts:
- `WATCHDOG_ADMIN_READ_HEADER_TIMEOUT_SECONDS` (int, default: `5`)
- `WATCHDOG_ADMIN_READ_TIMEOUT_SECONDS` (int, default: `30`)
- `WATCHDOG_ADMIN_WRITE_TIMEOUT_SECONDS` (int, default: `60`)
- `WATCHDOG_ADMIN_IDLE_TIMEOUT_SECONDS` (int, default: `120`)
- `WATCHDOG_ADMIN_SHUTDOWN_TIMEOUT_SECONDS` (int, default: `10`)

### 1.2 Cloudflare Access 검증(필수)

- `WATCHDOG_ADMIN_CF_ACCESS_TEAM_DOMAIN` (string, required)
  - 예: `myteam` 또는 `myteam.cloudflareaccess.com`
- `WATCHDOG_ADMIN_CF_ACCESS_AUD` (string, required)
- `WATCHDOG_ADMIN_ALLOWED_EMAILS` (string list, optional)
  - Access는 통과했더라도, 이 allowlist가 설정되면 해당 이메일만 허용합니다.
  - 구분자: 공백/쉼표 (예: `a@x.com,b@y.com`)

### 1.3 내부 서비스 인증

- `WATCHDOG_INTERNAL_SERVICE_TOKEN` (string, optional)
  - Docker 네트워크 내 서비스 간 인증에 사용되는 토큰
  - `X-Internal-Service-Token` 헤더로 전달
  - 설정 시 CF Access 인증을 우회할 수 있음

- `WATCHDOG_SKIP_AUTH_MODE` (string, default: `token_only`)
  - `skip_auth=true` 쿼리 파라미터의 허용 범위를 제어
  - 값:
    - `disabled`: skip_auth 완전 비활성화 (프로덕션 권장)
    - `token_only`: X-Internal-Service-Token 헤더 필수 (기본값)
    - `docker_network`: Docker 네트워크 IP (172.x, 10.x, 192.168.x)에서만 허용
    - `local_only`: localhost (127.0.0.1, ::1)에서만 허용

### 1.4 워치독 설정 리로드(옵션)

- `WATCHDOG_CONFIG_PATH` (string, optional): JSON 설정 파일 경로
  - 설정되면 **기동 시에도** 파일을 읽어 ENV 기본값 위에 덮어씁니다.

## 2. 인증 방식

### 2.1 Cloudflare Access (기본)

- 모든 `/admin/api/v1/*` 요청은 아래 헤더가 필요합니다.
  - `Cf-Access-Jwt-Assertion: <JWT>`
- 워치독은 아래 URL의 JWKS를 주기적으로 가져와 JWT 서명을 검증합니다.
  - `https://<TEAM_DOMAIN>/cdn-cgi/access/certs`
- 검증 항목:
  - `alg=RS256`
  - `aud` = `WATCHDOG_ADMIN_CF_ACCESS_AUD`
  - `iss` = `https://<TEAM_DOMAIN>`
  - `email` claim 존재 (없으면 거부)

### 2.2 Internal Service Token (서비스 간 호출)

- Docker 네트워크 내 서비스 간 호출 시 사용
- 요청 헤더에 `X-Internal-Service-Token: <TOKEN>` 추가
- `WATCHDOG_INTERNAL_SERVICE_TOKEN` 환경변수와 일치하면 인증 성공
- 예시:
  ```bash
  curl -H "X-Internal-Service-Token: your-secret-token" \
       http://watchdog:30002/admin/api/v1/targets
  ```

### 2.3 skip_auth (레거시/개발용, 권장하지 않음)

- `skip_auth=true` 쿼리 파라미터는 `WATCHDOG_SKIP_AUTH_MODE`에 따라 동작합니다.
- **주의**: 프로덕션에서는 `disabled` 또는 `token_only` 모드를 권장합니다.
- 모든 경우에 `WATCHDOG_ADMIN_ALLOWED_IPS` IP allowlist는 적용됩니다.

## 3. 공통 응답/에러

에러 응답은 아래 포맷을 사용합니다.

```json
{
  "error": { "code": "invalid_token", "message": "유효하지 않은 토큰입니다." }
}
```

## 4. 엔드포인트

### 4.1 Health (무인증)

- `GET /health`

### 4.2 워치독 상태

- `GET /admin/api/v1/watchdog/status`
  - 워치독 업타임/현재 설정 요약 반환

- `POST /admin/api/v1/watchdog/check-now`
  - 즉시 헬스체크를 트리거합니다(큐잉).

- `POST /admin/api/v1/watchdog/reload-config`
  - `WATCHDOG_CONFIG_PATH` 기반으로 설정을 재로딩합니다.
  - 응답에는 `appliedFields`, `requiresRestartFields`가 포함됩니다.

### 4.3 Docker 컨테이너 목록(Inventory)

- `GET /admin/api/v1/docker/containers`
  - Docker의 전체 컨테이너 목록 + `managed` 여부를 반환합니다.
  - 응답에는 `generatedAt`, `containers`가 포함됩니다.

### 4.4 대상 컨테이너 상태

- `GET /admin/api/v1/targets`
  - 관리 대상 컨테이너 목록 + 상태 스냅샷

- `GET /admin/api/v1/targets/:name`
  - 단일 컨테이너 상세 상태

### 4.5 관리대상 on/off (영구 반영)

`pause/resume`는 **모니터링만** 제어하며, “관리대상 포함/제외”를 의미하지 않습니다.

- `PUT /admin/api/v1/targets/:name/managed`
  - body:
    - `managed` (bool, required): `true`면 관리대상 포함, `false`면 제외
    - `reason` (string, optional): 감사/추적용 사유
  - 동작:
    - `WATCHDOG_CONFIG_PATH` JSON 파일의 `containers` 배열을 수정(append/remove)합니다.
    - 수정 후 `reload-config`와 동일하게 런타임 설정도 즉시 반영합니다.
  - 요구사항:
    - `WATCHDOG_CONFIG_PATH`가 설정되어 있어야 합니다.
    - 워치독 프로세스가 설정 파일에 **쓰기 권한**을 가져야 합니다.

### 4.6 컨테이너 제어

주의: 워치독이 자동 재시작을 수행하므로, **의도적으로 중지**하려면 monitoring pause가 필요합니다.

- `POST /admin/api/v1/targets/:name/restart`
  - body:
    - `reason` (string, optional)
    - `force` (bool, optional): 쿨다운/진행중 상태를 무시하고 재시작 시도

- `POST /admin/api/v1/targets/:name/stop`
  - 동작: **monitoring을 pause**한 뒤 컨테이너 stop을 호출합니다.
  - body:
    - `timeoutSeconds` (int, optional, default 10)
    - `reason` (string, optional)

- `POST /admin/api/v1/targets/:name/start`
  - 동작: 컨테이너 start 후 monitoring을 resume 합니다.
  - body:
    - `reason` (string, optional)

- `POST /admin/api/v1/targets/:name/pause`
  - monitoring만 pause (컨테이너는 그대로)

- `POST /admin/api/v1/targets/:name/resume`
  - monitoring resume + 즉시 헬스체크 트리거

### 4.7 로그

- `GET /admin/api/v1/targets/:name/logs`
  - query:
    - `tail` (int, default 200, max 2000)
    - `timestamps` (bool, default true, `false` 가능)
  - 응답: `text/plain`

- `GET /admin/api/v1/targets/:name/logs/stream`
  - SSE 스트림(`text/event-stream`)
  - query:
    - `tail` (int, default 200, max 2000)
  - 이벤트 payload는 JSON 한 줄입니다.

### 4.8 이벤트(히스토리)

- `GET /admin/api/v1/events?limit=200`
  - 최근 워치독 이벤트/관리 액션 버퍼를 반환합니다.

## 5. 설정 파일(JSON) 예시

```json
{
  "enabled": true,
  "containers": ["twentyq", "turtlesoup"],
  "intervalSeconds": 30,
  "maxFailures": 1,
  "cooldownSeconds": 120,
  "restartTimeoutSec": 30,
  "dockerSocket": "/var/run/docker.sock",
  "useEvents": true,
  "eventMinIntervalSec": 1,
  "statusReportSeconds": 60,
  "verboseLogging": false
}
```

참고:
- `containers`는 빈 배열(`[]`)이 될 수 있습니다(관리 대상 없음).
- 다만 `WATCHDOG_ADMIN_ENABLED=false` 상태에서 `containers`가 비어있으면 워치독은 기동 시 종료합니다(설정 누락을 조기에 감지하기 위함).
