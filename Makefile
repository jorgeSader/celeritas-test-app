# Variables
BINARY_NAME = celeritasApp
COMPOSE_FILE = docker-compose.yml
DB_DATA_DIR = ./db-data

# Default target
.PHONY: all
all: container-up start

# --- Application Targets ---
.PHONY: update
update:
	@echo "Updating Vendors..."
	@go get github.com/jorgeSader/celeritas
	@echo "Vendors Updated!"

.PHONY: build
build: update
	@echo "Building Celeritas..."
	@go build -o tmp/${BINARY_NAME} .
	@echo "Celeritas Built!"

.PHONY: run
run: build
	@echo "Starting Celeritas..."
	@./tmp/${BINARY_NAME} &
	@echo "Celeritas started!"

.PHONY: clean
clean:
	@echo "Cleaning..."
	@go clean
	@rm -f tmp/${BINARY_NAME}
	@echo "Cleaned!"

.PHONY: test
test:
	@echo "Testing..."
	@go test ./...
	@echo "Done!"

.PHONY: start
start: container-up run

.PHONY: stop
stop:
	@echo "Stopping Celeritas..."
	@-pkill -SIGTERM -f "./tmp/${BINARY_NAME}"
	@echo "Stopped Celeritas!"

.PHONY: restart
restart: stop start
	@echo "Restarted Celeritas!"

# --- Container Targets ---
.PHONY: container-setup
container-setup:
	@echo "Creating volume directories..."
	mkdir -p $(DB_DATA_DIR)/postgres $(DB_DATA_DIR)/redis $(DB_DATA_DIR)/mariadb $(DB_DATA_DIR)/init-scripts
	@echo "Setting ownership to $(USER)..."
	sudo chown -R $(USER):$(USER) $(DB_DATA_DIR)/postgres $(DB_DATA_DIR)/redis $(DB_DATA_DIR)/mariadb $(DB_DATA_DIR)/init-scripts
	@echo "Setting SELinux context..."
	sudo chcon -Rt container_file_t $(DB_DATA_DIR)/postgres $(DB_DATA_DIR)/redis $(DB_DATA_DIR)/mariadb $(DB_DATA_DIR)/init-scripts

.PHONY: container-up
container-up: container-setup
	@echo "Starting containers..."
	podman-compose -f $(COMPOSE_FILE) up -d

.PHONY: container-down
container-down:
	@echo "Stopping and removing containers..."
	podman-compose -f $(COMPOSE_FILE) down
	podman rm -f $$(podman ps -aq) || true

.PHONY: container-ps
container-ps:
	@echo "Listing running containers..."
	podman ps

.PHONY: container-logs
container-logs:
	@echo "Showing container logs..."
	podman-compose -f $(COMPOSE_FILE) logs

.PHONY: container-clean
container-clean: container-down
	@echo "Removing volume directories..."
	sudo rm -rf $(DB_DATA_DIR)

.PHONY: full-clean
full-clean: clean container-clean
	@echo "Full cleanup complete!"



# For Dev Only
.PHONY: stage-all
stage-all:
	@echo "Staging all files..."
	git add .
	@echo "All file Staged"

.PHONY: diff
diff:
	@echo "Creating diff file..."
	git diff --staged > changes.diff
	@echo "Diff file Created"

.PHONY: diff-all
diff-all: stage-all diff
	@echo "Staged all modified files and created a 'changes.diff' file."