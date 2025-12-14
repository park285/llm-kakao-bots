.PHONY: help build run test lint format clean docker

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build project
	./gradlew build

run: ## Run application
	./gradlew run

test: ## Run tests
	./gradlew test

lint: ## Run linters (ktlint + detekt)
	./gradlew ktlintCheck detekt

format: ## Format code
	./gradlew ktlintFormat

clean: ## Clean build artifacts
	./gradlew clean

docker: ## Build Docker image
	./gradlew buildImage

check: lint test ## Run all checks (lint + test)

dev: ## Run in development mode
	./gradlew run --continuous
