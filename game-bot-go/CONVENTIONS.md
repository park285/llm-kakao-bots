# game-bot-go CONVENTIONS (Go)

> CORE: DRY (no duplication) / REUSE FIRST

## 0. Principles

- No duplicate implementations: search with `rg` before writing code.
- Prefer shared components under `internal/common`.
- Avoid hardcoding: do not embed user-facing text, constants, keys, or thresholds in code; follow existing config/message systems.

## 1. Reusable Components (Required)

### 1.1 Message Templates (`internal/common/messageprovider`)

- Do not hardcode user-facing text in code. Use message templates (YAML) + keys.
- Keep the `messageprovider.Provider.Get(key, params...)` pattern.
- Message YAML locations:
  - `internal/twentyq/assets/messages/game-messages.yml`
  - `internal/turtlesoup/assets/messages/game-messages.yml`

### 1.2 MQ / Message Types (`internal/common/mq`, `internal/common/mqmsg`)

- Prefer `internal/common/mq` for stream publish/consume.
- Use `mqmsg.OutboundMessage` for outbound types (`waiting/final/error`) and stream fields.

### 1.3 Queue / Dedup Queue (`internal/common/pending`)

- Do not re-implement queue ordering/dedup/storage; reuse `pending.Store`.
- If a domain wrapper is required, keep it minimal and thin.

### 1.4 Redis/Valkey Client (`internal/common/valkeyx`)

- Use `valkeyx` to create Valkey clients and perform ping checks.
- DI wrappers: `di.DataValkeyClient`, `di.MQValkeyClient`

### 1.5 Text Chunking (`internal/common/textutil`)

- For Kakao message length limits, prefer `textutil.ChunkByLines`.

## 2. Error Handling (Required)

- Always wrap errors returned from other packages. (`wrapcheck`)
- Example:

```go
value, err := dep.Do(ctx)
if err != nil {
	return fmt.Errorf("do failed: %w", err)
}
```

## 3. Context Rules (Required)

- `context.Context` is always the first parameter.
- In HTTP/MQ handlers, pass through the incoming context; avoid sprinkling `context.Background()`.

## 4. Logging Rules

- Use `slog` key-value logging: `logger.Info("event_name", "key", value, ...)`.
- Avoid string interpolation/concatenation (consistency/perf).

## 5. Testing

- Target `go test ./...`.
- Prefer table-driven tests.
- If Redis is required, use `miniredis` or test doubles (keep I/O isolated when possible).

## 6. Naming

- Exported: `PascalCase`, internal: `camelCase`
- Booleans: `is/has/can` prefix
- Errors: `ErrXxx` or `XxxError` (provide `Unwrap()` when appropriate)
- Constants: prefer domain `config` constants; otherwise `SCREAMING_SNAKE_CASE`

## 7. REST API

- Prefer plural resources (e.g., `/riddles`, `/hints`).
- Express actions via HTTP verbs; do not add verbs to paths (`/create`, `/update`).
- Follow existing route/DTO patterns under `internal/*/httpapi`.

## 8. Forbidden / Required (Enforced)

- No hardcoded messages → use message templates.
- No magic numbers → use `internal/*/config` constants.
- No returning external-package errors as-is → always wrap.
- No emojis in code/logs/docs (message templates are the only exception).
