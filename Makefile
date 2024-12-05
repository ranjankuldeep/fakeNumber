BUILD_DIR := ./bin
BUILD_FILE_AMD64 := $(BUILD_DIR)/app_amd64
BUILD_FILE_ARM64 := $(BUILD_DIR)/app_arm64
BUILD_FILE_MAC := $(BUILD_DIR)/app_mac

ARCH := $(shell uname -m)
OS := $(shell uname -s)

GOARCH := $(if $(filter $(ARCH),x86_64),amd64,$(if $(filter $(ARCH),arm64 aarch64),arm64,unknown))
GOOS := $(if $(filter $(OS),Darwin),darwin,$(if $(filter $(OS),Linux),linux,unknown))

# Default build target
all: build

build:
	@mkdir -p $(BUILD_DIR)
	@if [ "$(GOARCH)" = "unknown" ] || [ "$(GOOS)" = "unknown" ]; then \
		echo "Unsupported platform: $(OS)/$(ARCH)"; \
		exit 1; \
	fi
	@echo "Building binaries..."
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_FILE_AMD64) -buildvcs=false ./cmd/app
	@echo "Built Linux amd64 binary at $(BUILD_FILE_AMD64)"
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_FILE_ARM64) -buildvcs=false ./cmd/app
	@echo "Built Linux arm64 binary at $(BUILD_FILE_ARM64)"
	GOOS=darwin GOARCH=$(GOARCH) go build -o $(BUILD_FILE_MAC) -buildvcs=false ./cmd/app
	@echo "Built macOS binary at $(BUILD_FILE_MAC)"

run:
	@if [ "$(GOOS)" = "linux" ]; then \
		if [ "$(GOARCH)" = "amd64" ]; then \
			echo "Running Linux amd64 binary..."; \
			$(BUILD_FILE_AMD64); \
		elif [ "$(GOARCH)" = "arm64" ]; then \
			echo "Running Linux arm64 binary..."; \
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
