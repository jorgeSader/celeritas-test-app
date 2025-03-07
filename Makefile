BINARY_NAME=celeritasApp
BUILD_DIR=tmp

# Build the application locally
.PHONY: build
build: 
	@echo "Building ${BINARY_NAME}..."
	@mkdir -p ${BUILD_DIR}
	@go build -o ./${BUILD_DIR}/${BINARY_NAME} . || { echo "Build failed!"; exit 1; }
	@echo "${BINARY_NAME} built!"

# Run the application locally
.PHONY: run
run: build
	@echo "Starting ${BINARY_NAME}..."
	@./${BUILD_DIR}/${BINARY_NAME} &
	@echo "${BINARY_NAME} started!"

# Start the database containers using Podman Compose
.PHONY: compose-up
compose-up:
	@echo "Starting DB Containers..."
	@podman-compose up -d
	@echo "DB Containers Started!"

# Stop the database containers
.PHONY: compose-down
compose-down:
	@echo "Stopping DB Containers..."
	@podman-compose down
	@echo "DB Containers Stopped!"

# Tail the logs of the database containers
.PHONY: compose-logs
compose-logs:
	@echo "Showing Logs..."
	@podman-compose logs -f

# Run the application with databases
.PHONY: run-with-db
run-with-db: compose-up run

# Clean up
.PHONY: clean
clean:
	@echo "Cleaning..."
	@go clean
	@rm -f ${BUILD_DIR}/${BINARY_NAME}
	@echo "Cleaned!"

# Run all tests
.PHONY: test
test:
	@echo "Testing..."
	@go test ./...
	@echo "Done!"

.PHONY: start
start: run

.PHONY: stop
stop:
	@echo "Stopping ${BINARY_NAME}..."
	@-pkill -SIGTERM -f "./${BUILD_DIR}/${BINARY_NAME}" || true
	@echo "Stopped ${BINARY_NAME}!"

.PHONY: restart
restart: stop start

##################################
# Dev-Only Targets
##################################
.PHONY: stage-all
stage-all:
	@echo "Staging all files..."
	@git add .
	@echo "All files staged!"

.PHONY: diff
diff:
	@echo "Copying diff to clipboard..."
	@if [ "$$(uname)" = "Darwin" ]; then \
		git diff --staged | pbcopy; \
		echo "Diff copied to clipboard (macOS)"; \
	elif [ "$$(uname)" = "Linux" ]; then \
		if command -v xclip >/dev/null 2>&1; then \
			git diff --staged | xclip -selection clipboard; \
			echo "Diff copied to clipboard (Linux/xclip)"; \
		elif command -v wl-copy >/dev/null 2>&1; then \
			git diff --staged | wl-copy; \
			echo "Diff copied to clipboard (Linux/wl-copy)"; \
		else \
			echo "Error: Install xclip or wl-copy for clipboard support"; \
			exit 1; \
		fi; \
	elif [ "$$(uname -o 2>/dev/null)" = "Msys" ] || [ "$$(uname -o 2>/dev/null)" = "Cygwin" ]; then \
		git diff --staged | clip; \
		echo "Diff copied to clipboard (Windows)"; \
	else \
		echo "Error: Unsupported OS for clipboard copy"; \
		exit 1; \
	fi

.PHONY: diff-all
diff-all: stage-all diff
	@echo "Staged all modified files and copied diff to clipboard."