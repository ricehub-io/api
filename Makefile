BINARY  := api
BUILD   := build
CMD     := .

GOFLAGS := -trimpath
LDFLAGS := -ldflags="-s -w"

.PHONY: all build run test lint fmt vet tidy security check clean swagger install-tools

all: check build

## build: compile the binary to ./build/api
build:
	@mkdir -p $(BUILD)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD)/$(BINARY) $(CMD)

## run: run the application without compiling a binary
run:
	go run $(CMD)

## test: run all tests with race detector
test:
	go test ./... -v -race -count=1

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## fmt: check if codebase is compliant with goimports' formatting
fmt:
	goimports -l .

## vet: run go vet
vet:
	go vet ./...

## tidy: tidy and verify go modules
tidy:
	go mod tidy
	go mod verify

## security: scan for vulnerabilities
security:
	govulncheck ./...
	gosec -exclude-generated ./...

## check: run fmt, vet, lint, security, and test
check: fmt vet lint security test

## clean: remove build artifacts
clean:
	rm -rf $(BUILD)

## swagger: generate swagger docs
swagger:
	swag init

## install-tools: install required dev tools
install-tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/swaggo/swag/cmd/swag@latest

## help: list available targets
help:
	@grep -E "^##" Makefile | sed "s/## //"
