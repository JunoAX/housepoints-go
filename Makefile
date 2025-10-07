.PHONY: help run build test docker-build docker-push migrate-up migrate-down lint fmt clean

# Variables
BINARY_NAME=housepoints-go
DOCKER_IMAGE=ghcr.io/junoax/housepoints-go
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
PORT?=8080

## help: Display this help message
help:
	@echo "Available commands:"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

## run: Run the server locally
run:
	go run cmd/server/main.go

## build: Build the binary
build:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-X main.Version=${VERSION}" -o bin/${BINARY_NAME} cmd/server/main.go

## test: Run all tests
test:
	go test -v -race -coverprofile=coverage.out ./...

## test-integration: Run integration tests
test-integration:
	go test -v -tags=integration ./tests/integration/...

## test-coverage: Run tests with coverage report
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

## docker-build: Build Docker image
docker-build:
	docker build -t ${DOCKER_IMAGE}:${VERSION} -t ${DOCKER_IMAGE}:latest .

## docker-push: Push Docker image to registry
docker-push:
	docker push ${DOCKER_IMAGE}:${VERSION}
	docker push ${DOCKER_IMAGE}:latest

## migrate-up: Run database migrations
migrate-up:
	@echo "Running migrations..."
	@echo "TODO: Implement migration tool"

## migrate-down: Rollback database migrations
migrate-down:
	@echo "Rolling back migrations..."
	@echo "TODO: Implement migration tool"

## lint: Run linters
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	go fmt ./...
	goimports -w .

## clean: Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

## dev: Run with hot reload
dev:
	air

## deps: Download dependencies
deps:
	go mod download
	go mod tidy
