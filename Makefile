BUILD_DIR := ./bin
BUILD_FILE := $(BUILD_DIR)/app
all: build run

build:
	@echo "Building the application..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_FILE) ./cmd/app
	@echo "Build complete. Binary saved to $(BUILD_FILE)"

run: build
	@echo "Running the application..."
	@$(BUILD_FILE)

clean:
	@echo "Cleaning build files..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete."
	
.PHONY: build run clean all
