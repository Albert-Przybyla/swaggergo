BINARY_NAME = swaggergo
MAIN_PATH = ./cmd/swaggergo
MODULE = github.com/Albert-Przybyla/swaggergo

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  = $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

BUILD_DIR = dist
INSTALL_DIR = $(GOPATH)/bin

GOFLAGS = -trimpath -buildvcs=false

LDFLAGS = -ldflags "\
-X $(MODULE)/internal/build.Version=$(VERSION) \
-X $(MODULE)/internal/build.Commit=$(COMMIT) \
-X $(MODULE)/internal/build.Date=$(DATE) \
-s -w"

.PHONY: all build install clean uninstall test tidy build-all package run

## default
all: build

## build: compile CLI
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

## run: run without building
run:
	go run $(MAIN_PATH)

## install: install to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME) to $(INSTALL_DIR)..."
	CGO_ENABLED=0 go install $(GOFLAGS) $(LDFLAGS) $(MAIN_PATH)
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
	@echo "Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

## package: archive binaries for release
package: build-all
	@echo "Packaging binaries..."
	cd $(BUILD_DIR) && \
	tar -czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
	tar -czf $(BINARY_NAME)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64 && \
	tar -czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
	tar -czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64 && \
	zip -q $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe

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
