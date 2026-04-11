.PHONY: run dev test test-unit test-integration lint vet fmt

run:
	go run cmd/api/main.go

dev:
	air

test:
	go test ./...

test-unit:
	go test ./... -short

test-integration:
	go test -tags=integration ./...

lint:
	golangci-lint run

vet:
	go vet ./...

fmt:
	go fmt ./...

help:
	@echo "Available commands:"
	@echo "  make run                - Run the API server once"
	@echo "  make dev                - Run with hot-reload (requires air)"
	@echo "  make test               - Run all tests"
	@echo "  make test-unit          - Run unit tests only"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make lint               - Run linter"
	@echo "  make vet                - Run go vet"
	@echo "  make fmt                - Format code"
