GO ?= go
GOLANGCI_LINT ?= golangci-lint
IMAGE_NAME ?= hololive-kakao-bot
IMAGE_TAG ?= latest

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run ./...

.PHONY: fmt
fmt:
	$(GOLANGCI_LINT) run --fix ./...

.PHONY: test
test:
	$(GO) test ./...

.PHONY: test-coverage
test-coverage:
	$(GO) test ./... -coverprofile=coverage.out
	$(GO) tool cover -html=coverage.out -o coverage.html

.PHONY: wire
wire:
	$(GO) generate -tags=wireinject ./...

.PHONY: build
build: lint build-bin

.PHONY: build-bin
build-bin:
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -tags go_json -trimpath -o bin/bot ./cmd/bot

.PHONY: docker-build
docker-build:
	docker build -f Dockerfile.prod -t $(IMAGE_NAME):$(IMAGE_TAG) .

.PHONY: docker-push
docker-push:
	docker push $(IMAGE_NAME):$(IMAGE_TAG)

.PHONY: clean
clean:
	rm -rf bin/ coverage.out coverage.html
