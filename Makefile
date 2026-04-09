# Makefile for grafana-govee-datasource
# Targets: build, test, lint, dev, clean

PLUGIN_ID := timlevett-govee-datasource
BINARY    := gpx_govee_datasource
PKG_DIR   := ./pkg

.PHONY: all build build-frontend build-backend test test-frontend test-backend \
        lint lint-frontend lint-backend dev clean install typecheck

# Default: full build
all: build

## install — install Node dependencies
install:
	npm ci

## build — build both frontend and backend
build: build-frontend build-backend

## build-frontend — compile TypeScript/React with webpack
build-frontend: install
	npm run build

## build-backend — compile the Go plugin binary for the host OS/arch
build-backend:
	@echo "Building Go backend..."
	go build -o $(BINARY) $(PKG_DIR)/main.go

## build-backend-linux — cross-compile for Linux amd64 (for Grafana on Linux)
build-backend-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BINARY)_linux_amd64 $(PKG_DIR)/main.go

## build-backend-all — compile for all platforms Grafana supports
build-backend-all:
	GOOS=linux   GOARCH=amd64  go build -o dist/gpx_govee_datasource_linux_amd64   $(PKG_DIR)/main.go
	GOOS=linux   GOARCH=arm64  go build -o dist/gpx_govee_datasource_linux_arm64   $(PKG_DIR)/main.go
	GOOS=windows GOARCH=amd64  go build -o dist/gpx_govee_datasource_windows_amd64.exe $(PKG_DIR)/main.go
	GOOS=darwin  GOARCH=amd64  go build -o dist/gpx_govee_datasource_darwin_amd64  $(PKG_DIR)/main.go
	GOOS=darwin  GOARCH=arm64  go build -o dist/gpx_govee_datasource_darwin_arm64  $(PKG_DIR)/main.go

## test — run all tests
test: test-frontend test-backend

## test-frontend — run Jest tests
test-frontend: install
	npm test

## test-backend — run Go tests
test-backend:
	go test ./...

## lint — run all linters
lint: lint-frontend lint-backend

## lint-frontend — run ESLint
lint-frontend: install
	npm run lint

## lint-backend — run golangci-lint (requires golangci-lint to be installed)
lint-backend:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	}
	golangci-lint run ./...

## typecheck — TypeScript type-check without emitting files
typecheck: install
	npm run typecheck

## dev — start webpack in watch mode for frontend development
dev: install
	npm run dev

## clean — remove build artefacts
clean:
	rm -rf dist/ node_modules/.cache/ coverage/ $(BINARY) $(BINARY)_*
	go clean ./...

## help — print this help
help:
	@echo "Available targets:"
	@grep -E '^## ' Makefile | sed 's/## /  /'
