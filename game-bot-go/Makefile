GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run ./...

.PHONY: fmt
fmt:
	$(GOLANGCI_LINT) run --fix ./...

.PHONY: test
test:
	$(GO) test ./...

.PHONY: wire
wire:
	$(GO) generate -tags=wireinject ./...

.PHONY: build
build: lint
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -tags go_json -trimpath -o bin/twentyq ./cmd/twentyq
	CGO_ENABLED=0 $(GO) build -tags go_json -trimpath -o bin/turtlesoup ./cmd/turtlesoup
