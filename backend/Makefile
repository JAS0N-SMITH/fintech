.PHONY: run dev test test-unit test-integration lint vet fmt vuln semgrep gosec

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

vuln:
	govulncheck ./...

semgrep:
	semgrep --config ../.semgrep.yml ./...

gosec:
	gosec ./...

help:
	@echo "Available commands:"
	@echo "  make run                - Run the API server once"
	@echo "  make dev                - Run with hot-reload (requires air: go install github.com/air-verse/air@latest)"
	@echo "  make test               - Run all tests"
	@echo "  make test-unit          - Run unit tests only"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make lint               - Run linter (includes gosec)"
	@echo "  make vet                - Run go vet"
	@echo "  make fmt                - Format code"
	@echo "  make vuln               - Check for known vulnerabilities (govulncheck)"
	@echo "  make semgrep            - Run semgrep static analysis (OWASP rules)"
	@echo "  make gosec              - Run gosec security scanner"
