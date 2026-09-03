.PHONY: help build run test clean migrate-up migrate-down docker-build docker-run

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the API server
	go build -o bin/api cmd/api/main.go

run: ## Run the API server
	go run cmd/api/main.go

test: ## Run all tests
	go test ./... -v

test-short: ## Run unit tests only (no database required)
	go test ./... -short -v

test-coverage: ## Run tests with coverage
	go test ./... -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

migrate-up: ## Apply database migrations
	go run cmd/migrate/main.go -command up

migrate-down: ## Rollback database migrations
	go run cmd/migrate/main.go -command down

migrate-version: ## Show current migration version
	go run cmd/migrate/main.go -command version

fmt: ## Format code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

lint: ## Run staticcheck (requires installation)
	staticcheck ./...

clean: ## Clean build artifacts
	rm -rf bin/
	rm -f coverage.out coverage.html

docker-build: ## Build Docker image
	docker build -t key-cabinet-server:latest .

docker-run: ## Run Docker container
	docker run -p 8080:8080 --env-file .env key-cabinet-server:latest

deps: ## Download dependencies
	go mod download

tidy: ## Tidy dependencies
	go mod tidy

install-tools: ## Install development tools
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
