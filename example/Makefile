.PHONY: proto server client clean

# Generate protocol buffer code
proto:
	@echo "Generating protocol buffer code..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/demo/v1/demo.proto
	@echo "✓ Protocol buffer code generated"

# Run the server
server:
	@echo "Starting demo server..."
	@cd server && go run main.go

# Run the client
client:
	@echo "Starting demo client..."
	@cd client && go run main.go

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	@rm -rf pkg/pb/*
	@echo "✓ Cleaned"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@echo "✓ Dependencies installed"

# Full setup: install deps, generate proto, and run server
setup: deps proto
	@echo "✓ Setup complete! Run 'make server' to start the server"

# Help
help:
	@echo "Available targets:"
	@echo "  make proto    - Generate protocol buffer code"
	@echo "  make server   - Run the demo server"
	@echo "  make client    - Run the demo client"
	@echo "  make deps      - Install required dependencies"
	@echo "  make setup     - Full setup (deps + proto generation)"
	@echo "  make clean     - Clean generated files"
	@echo "  make help      - Show this help message"

