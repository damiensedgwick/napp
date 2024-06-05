BINARY_NAME=%s

all: clean format build run

build:
	@echo "Building..."
	@go build -o bin/$(BINARY_NAME) cmd/main.go

clean:
	@echo "Cleaning..."
	@go clean
	@if [ -e bin/$(BINARY_NAME) ]; then rm bin/$(BINARY_NAME); fi

format:
	@echo "Formatting..."
	@go fmt ./...

run:
	@echo "Running..."
	@./bin/$(BINARY_NAME)

test:
	@echo "Testing..."
	@go test ./...
