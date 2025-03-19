BINARY_NAME=devifyApp
BUILD_DIR=tmp

.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  build             Build the application locally"
	@echo "  run               Run the application locally (builds first)"
	@echo "  compose-up        Start the database containers using Podman Compose"
	@echo "  compose-down      Stop the database containers"
	@echo "  compose-logs      Tail the logs of the database containers"
	@echo "  run-with-db       Run the application with databases (starts DB and app)"
	@echo "  clean             Clean up build artifacts"
	@echo "  test              Run all tests"
	@echo "  coverage          Display test coverage"
	@echo "  cover             Open coverage report in browser"
	@echo "  start             Alias for 'run'"
	@echo "  stop              Stop the running application"
	@echo "  restart           Restart the application (stop then start)"
	@echo "  stage-all         Stage all files for git commit"
	@echo "  unstage-all       Unstage all files from git"
	@echo "  diff-to-clipboard Copy staged git diff to clipboard"
	@echo "  diff              Stage all, copy diff to clipboard, then unstage"
	@echo "  diff-file         Stage all, save diff to diff_output.txt, then unstage"
	@echo ""
	@echo "Note: Output file for 'diff-file' is 'diff_output.txt'."

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

# Display test coverage
.PHONY: coverage
coverage:
	@echo "Generating test coverage..."
	@go test -cover ./...
	@echo "Coverage displayed!"

# Open coverage report in browser
.PHONY: cover
cover:
	@echo "Generating and opening coverage report..."
	@go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
	@echo "Coverage report opened!"

# Alias for 'run'
.PHONY: start
start: run

# Stop the running application
.PHONY: stop
stop:
	@echo "Stopping ${BINARY_NAME}..."
	@-pkill -SIGTERM -f "./${BUILD_DIR}/${BINARY_NAME}" || true
	@echo "Stopped ${BINARY_NAME}!"

#Restart the application (stop then start)
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

.PHONY: unstage-all
unstage-all:
	@echo "Unstaging all files..."
	@git restore --staged .
	@echo "All files unstaged!"

.PHONY: diff-to-clipboard
diff-to-clipboard:
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

.PHONY: diff
diff: stage-all diff-to-clipboard  unstage-all
	@echo "DIff content on clipboard and ready to paste."

.PHONY: diff-file
diff-file: stage-all
	@echo "Saving diff to diff_output.txt..."
	@git diff --staged > diff_output.txt
	@echo "Diff saved to diff_output.txt."
	@$(MAKE) unstage-all
