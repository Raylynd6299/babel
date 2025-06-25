# Makefile
.PHONY: help build run test clean docker-build docker-up docker-down migrate seed

# Default help command
help:
	@echo "Available commands:"
	@echo "  build        - Build all services"
	@echo "  run          - Run specific service (make run SERVICE=auth-service)"
	@echo "  test         - Run tests"
	@echo "  docker-build - Build Docker images"
	@echo "  docker-up    - Start Docker Compose stack"
	@echo "  docker-down  - Stop Docker Compose stack"
	@echo "  migrate      - Run database migrations"
	@echo "  seed         - Seed database with initial data"
	@echo "  clean        - Clean build artifacts"

# Build all services
build:
	@echo "Building all services..."
	go build -o bin/api-gateway ./cmd/api-gateway
	go build -o bin/auth-service ./cmd/auth-service
	go build -o bin/content-service ./cmd/content-service
	go build -o bin/progress-service ./cmd/progress-service
	go build -o bin/vocabulary-service ./cmd/vocabulary-service
	go build -o bin/phonetic-service ./cmd/phonetic-service
	go build -o bin/social-service ./cmd/social-service
	go build -o bin/gamification-service ./cmd/gamification-service
	go build -o bin/analytics-service ./cmd/analytics-service

# Run specific service
run:
ifndef SERVICE
	@echo "Please specify SERVICE. Example: make run SERVICE=auth-service"
	@exit 1
endif
	@echo "Running $(SERVICE)..."
	go run ./cmd/$(SERVICE)

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Build Docker images
docker-build:
	@echo "Building Docker images..."
	docker-compose build

# Start Docker Compose stack
docker-up:
	@echo "Starting Docker Compose stack..."
	docker-compose up -d
	@echo "Services started. API Gateway available at http://localhost:8080"

# Stop Docker Compose stack
docker-down:
	@echo "Stopping Docker Compose stack..."
	docker-compose down

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	docker system prune -f

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	go mod tidy
	go mod download
	mkdir -p bin/

# Install tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/air-verse/air@latest
	go install github.com/golang/mock/mockgen@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run with hot reload
dev:
ifndef SERVICE
	@echo "Please specify SERVICE. Example: make dev SERVICE=auth-service"
	@exit 1
endif
	@echo "Running $(SERVICE) with hot reload..."
	air -c .air.$(SERVICE).toml

# Linting
lint:
	@echo "Running linter..."
	golangci-lint run

# Generate mocks
generate-mocks:
	@echo "Generating mocks..."
	go generate ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Security scan
security:
	@echo "Running security scan..."
	gosec ./...

# Build for production
build-prod:
	@echo "Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o bin/api-gateway ./cmd/api-gateway
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o bin/auth-service ./cmd/auth-service
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o bin/content-service ./cmd/content-service
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o bin/progress-service ./cmd/progress-service
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-w -s' -o bin/vocabulary-service ./cmd/vocabulary-service