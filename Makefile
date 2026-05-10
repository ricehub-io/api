BIN_NAME  := api
BUILD_DIR := build
CMD       := .

GOFLAGS := -trimpath
LDFLAGS := -ldflags="-s -w"

.PHONY: build run test lint fmt security check swagger install-tools

## build: compile the source code
build:
	@mkdir -p $(BUILD_DIR)
	go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BIN_NAME) $(CMD)

## run: run the app without compiling a binary
run:
	go run $(CMD)

## test: run all tests with race detector
test:
	go test ./... -v -race

## lint: run golangci-lint
lint:
	golangci-lint run ./...

## fmt: check if codebase is compliant with goimports' formatting
fmt:
	golangci-lint fmt --diff ./...

## security: scan for vulnerabilities
security:
	govulncheck ./...
# 	gosec -exclude-generated -conf gosec-config.json ./...

## check: run fmt, lint, security, and test
check: fmt lint security test

## swagger: generate swagger docs
swagger:
	swag init

## install-tools: install required dev tools
install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install github.com/swaggo/swag/cmd/swag@latest

## help: list available targets
help:
	@grep -E "^##" Makefile | sed "s/## //"
