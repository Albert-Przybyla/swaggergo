BINARY_NAME = swaggergo
MAIN_PATH = ./cmd/swaggergo

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  = $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

BUILD_DIR = dist
INSTALL_DIR = $(GOPATH)/bin

LDFLAGS = -ldflags "\
-X swaggergo/internal/build.Version=$(VERSION) \
-X swaggergo/internal/build.Commit=$(COMMIT) \
-X swaggergo/internal/build.Date=$(DATE) \
-s -w"

.PHONY: all build install clean uninstall test tidy build-all run

all: build

## build: compile CLI
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

## run: run without building
run:
	go run $(MAIN_PATH)

## install: install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	go install $(LDFLAGS) $(MAIN_PATH)
	@echo "✅ Installed."

## install-global: install to /usr/local/bin (requires sudo)
install-global: build
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Installed globally"

## uninstall: remove binary from system
uninstall:
	@echo "Removing $(BINARY_NAME)..."
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Removed (if existed)"

## build-all: cross-compile
build-all:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

## tidy
tidy:
	go mod tidy

## test
test:
	go test ./...

## clean: remove build artifacts
clean:
	rm -rf $(BUILD_DIR)

## help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
