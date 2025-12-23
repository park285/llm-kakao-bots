ARG GO_VERSION=1.25.4

FROM golang:${GO_VERSION}-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags go_json -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /app/bin/bot ./cmd/bot

FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1000 appuser \
    && adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appuser /app/bin/bot ./bin/bot
COPY --from=builder --chown=appuser:appuser /app/admin-ui/dist ./admin-ui/dist
COPY --from=builder --chown=appuser:appuser /app/internal/domain/data ./internal/domain/data
COPY --from=builder --chown=appuser:appuser /app/internal/adapter/templates ./internal/adapter/templates
COPY --from=builder --chown=appuser:appuser /app/config.example.json ./config.example.json
COPY --from=builder --chown=appuser:appuser /app/config.json ./config.json
COPY --from=builder --chown=appuser:appuser /app/official_japanese_names.json ./official_japanese_names.json

RUN mkdir -p /app/logs && chown -R appuser:appuser /app

ENV GIN_MODE=release
ENV GOGC=60
EXPOSE 30001

USER appuser

CMD ["./bin/bot"]
