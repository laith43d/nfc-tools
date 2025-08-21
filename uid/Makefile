# Makefile for NFC UID Service

BINARY_NAME=nfc-uid-service
WINDOWS_BINARY=$(BINARY_NAME).exe
LINUX_BINARY=$(BINARY_NAME)-linux
MACOS_BINARY=$(BINARY_NAME)-macos

# Build targets
.PHONY: all build clean test install uninstall deps

all: build

# Build for current platform
build:
	go build -o $(BINARY_NAME) .

# Build for all platforms
build-all: build-windows build-linux build-macos

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_BINARY) .

build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(LINUX_BINARY) .

build-macos:
	GOOS=darwin GOARCH=amd64 go build -o $(MACOS_BINARY) .

# Install dependencies
deps:
	go mod download
	go mod tidy

# Run tests
test:
	go test -v ./...

# Test the application
test-run: build
	./$(BINARY_NAME) -test

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME) $(WINDOWS_BINARY) $(LINUX_BINARY) $(MACOS_BINARY)

# Install service (platform-specific)
install: build
ifeq ($(OS),Windows_NT)
	@echo "Run install.bat as Administrator to install the Windows service"
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		@echo "Run 'sudo ./install.sh' to install the Linux service"
	endif
	ifeq ($(UNAME_S),Darwin)
		@echo "Run 'sudo ./install-macos.sh' to install the macOS service"
	endif
endif

# Uninstall service (platform-specific)  
uninstall:
ifeq ($(OS),Windows_NT)
	@echo "Run uninstall.bat as Administrator to uninstall the Windows service"
else
	UNAME_S := $(shell uname -s)
	ifeq ($(UNAME_S),Linux)
		@echo "Run 'sudo ./uninstall.sh' to uninstall the Linux service"
	endif
	ifeq ($(UNAME_S),Darwin)
		@echo "Run 'sudo ./uninstall-macos.sh' to uninstall the macOS service"
	endif
endif

# Development helpers
run: build
	./$(BINARY_NAME)

run-debug: build
	./$(BINARY_NAME) -debug

run-test: build
	./$(BINARY_NAME) -test

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build for current platform"
	@echo "  build-all    - Build for all platforms"
	@echo "  deps         - Install Go dependencies"
	@echo "  test         - Run Go tests"
	@echo "  test-run     - Build and test with NFC reader"
	@echo "  clean        - Clean build artifacts"
	@echo "  install      - Show platform-specific install instructions"
	@echo "  uninstall    - Show platform-specific uninstall instructions"
	@echo "  run          - Build and run normally"
	@echo "  run-debug    - Build and run with debug output"
	@echo "  run-test     - Build and run in test mode"
	@echo "  help         - Show this help message"
