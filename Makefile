BUILD_DIR := ./bin
BUILD_FILE_AMD64 := $(BUILD_DIR)/app_amd64
BUILD_FILE_ARM64 := $(BUILD_DIR)/app_arm64
BUILD_FILE_MAC := $(BUILD_DIR)/app_mac

ARCH := $(shell uname -m)
OS := $(shell uname -s)

GOARCH := $(if $(filter $(ARCH),x86_64),amd64,$(if $(filter $(ARCH),arm64 aarch64),arm64,unknown))
GOOS := $(if $(filter $(OS),Darwin),darwin,$(if $(filter $(OS),Linux),linux,unknown))

all: build

build:
	@mkdir -p $(BUILD_DIR)
	@if [ "$(GOARCH)" = "unknown" ] || [ "$(GOOS)" = "unknown" ]; then \
		echo "Unsupported platform: $(OS)/$(ARCH)"; \
		exit 1; \
	fi
	@echo "Building the application for amd64..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_FILE_AMD64) ./cmd/app
	@echo "Build complete for amd64. Binary saved to $(BUILD_FILE_AMD64)"
	@echo "Building the application for arm64..."
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_FILE_ARM64) ./cmd/app
	@echo "Build complete for arm64. Binary saved to $(BUILD_FILE_ARM64)"
	@echo "Building the application for macOS $(GOARCH)..."
	GOOS=darwin GOARCH=$(GOARCH) go build -o $(BUILD_FILE_MAC) ./cmd/app
	@echo "Build complete for macOS. Binary saved to $(BUILD_FILE_MAC)"

run:
	@if [ "$(GOOS)" = "linux" ]; then \
		if [ "$(GOARCH)" = "amd64" ]; then \
			echo "Running amd64 binary..."; \
			$(BUILD_FILE_AMD64); \
		elif [ "$(GOARCH)" = "arm64" ]; then \
			echo "Running arm64 binary..."; \
			$(BUILD_FILE_ARM64); \
		fi; \
	elif [ "$(GOOS)" = "darwin" ]; then \
		echo "Running macOS binary..."; \
		$(BUILD_FILE_MAC); \
	else \
		echo "Unsupported platform: $(OS)/$(ARCH)"; \
		exit 1; \
	fi

clean:
	@echo "Cleaning build files..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete."

.PHONY: build run clean all
