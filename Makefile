BINARY_NAME=celeritasApp

update:
		@echo "Updating Vendors..."
		@go get github.com/jorgeSader/celeritas
		@echo "Vendors Updated!"

build: update
		@echo "Building Celeritas..."
		@go build -o tmp/${BINARY_NAME} .
		@echo "Celeritas Built!"

run: build
		@echo "Starting Celeritas..."
		@./tmp/${BINARY_NAME} &
		@echo "Celeritas started!"

clean: 
		@echo "Cleaning..."
		@go clean
		@rm tmp/${BINARY_NAME}
		@echo "Cleaned!"

test: 
		@echo "Testing..."
		@go test ./...
		@echo "Done!"

start: run

stop: 
		@echo "Stopping Celeritas..."
		@-pkill -SIGTERM -f "./tmp/${BINARY_NAME}"
		@echo "Stopped Celeritas!"

restart: stop start
		@echo "Restarted Celeritas!"