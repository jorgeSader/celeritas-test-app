# Makefile for managing the Celeritas test application, including app and container operations.

# Variables
BINARY_NAME = celeritasApp
COMPOSE_FILE = docker-compose.yml
DB_DATA_DIR = ./db-data

# --- App-Only Targets ---
.PHONY: start
start: build
	@echo "Starting Celeritas..."
	@./tmp/${BINARY_NAME} &
	@echo "Celeritas started!"

.PHONY: stop
stop:
	@echo "Stopping Celeritas..."
	@-pkill -SIGTERM -f "./tmp/${BINARY_NAME}"
	@echo "Stopped Celeritas!"

.PHONY: clean
clean:
	@echo "Cleaning app artifacts..."
	@go clean
	@rm -f tmp/${BINARY_NAME}
	@echo "App cleaned!"

.PHONY: restart
restart: stop start
	@echo "Restarted Celeritas app!"

# --- Container-Only Targets ---
.PHONY: container-start
container-start: container-setup
	@echo "Starting containers..."
	podman-compose -f $(COMPOSE_FILE) up -d

.PHONY: container-stop
container-stop:
	@echo "Stopping containers..."
	podman-compose -f $(COMPOSE_FILE) stop

.PHONY: container-clean
container-clean:
	@echo "Cleaning containers..."
	podman-compose -f $(COMPOSE_FILE) down
	podman ps -aq | xargs -r podman rm -f || true
	@echo "Containers cleaned!"

.PHONY: container-restart
container-restart: container-stop container-start
	@echo "Restarted containers!"

# --- Combined Targets (App + Containers) ---
.PHONY: start-all
start-all: container-start start
	@echo "Started app and containers!"

.PHONY: stop-all
stop-all: stop container-stop
	@echo "Stopped app and containers!"

.PHONY: clean-all
clean-all: clean container-clean
	@echo "Cleaned app and containers!"

.PHONY: restart-all
restart-all: stop-all start-all
	@echo "Restarted app and containers!"

# --- Utility Targets ---
.PHONY: build
build:
	@echo "Building Celeritas..."
	@go build -o tmp/${BINARY_NAME} .
	@echo "Celeritas Built!"

.PHONY: container-setup
container-setup:
	@echo "Creating volume directories..."
	mkdir -p $(DB_DATA_DIR)/postgres $(DB_DATA_DIR)/redis $(DB_DATA_DIR)/mariadb $(DB_DATA_DIR)/init-scripts
	@echo "Setting ownership for containers..."
	sudo chown -R $(USER):$(USER) $(DB_DATA_DIR)/postgres $(DB_DATA_DIR)/redis $(DB_DATA_DIR)/mariadb $(DB_DATA_DIR)/init-scripts
	sudo chmod -R 700 $(DB_DATA_DIR)/postgres  # Ensure Postgres can write
	@echo "Setting SELinux context..."
	sudo chcon -Rt container_file_t $(DB_DATA_DIR)/postgres $(DB_DATA_DIR)/redis $(DB_DATA_DIR)/mariadb $(DB_DATA_DIR)/init-scripts

# --- Full Wipe Target ---
.PHONY: full-clean
full-clean: clean-all
	@echo "Removing volume directories..."
	sudo rm -rf $(DB_DATA_DIR)
	@echo "Full cleanup complete!"

# --- Dev-Only Targets ---
.PHONY: stage-all
stage-all:
	@echo "Staging all files..."
	git add .
	@echo "All files staged!"

.PHONY: diff
diff:
	@echo "Copying diff to clipboard..."
	@# Detect OS and use appropriate clipboard tool
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