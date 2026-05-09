.PHONY: build run test fmt vet clean

BINARY := dungeon

build:
	@echo "Building $(BINARY)..."
	go build -o $(BINARY) .

run: build
	@echo "Preparing config and running..."
	@cp -f data/config.json config.json
	@./$(BINARY) < data/events
	@rm -f config.json

test:
	@echo "Running tests..."
	go test ./...

fmt:
	@echo "Formatting sources..."
	go fmt ./...

vet:
	@echo "Running go vet..."
	go vet ./...

clean:
	@echo "Cleaning..."
	-rm -f $(BINARY)
