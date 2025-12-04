#!/bin/bash
# Setup script for the gRPC communication types example

set -e

echo "🚀 Setting up gRPC Communication Types Example"
echo ""

# Check if protoc is installed
if ! command -v protoc &> /dev/null; then
    echo "❌ Error: protoc is not installed"
    echo "   Install it from: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

# Install Go plugins
echo "📦 Installing Go protobuf plugins..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate protocol buffer code
echo "🔨 Generating protocol buffer code..."
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/demo/v1/demo.proto

if [ $? -eq 0 ]; then
    echo "✅ Protocol buffer code generated successfully!"
    echo ""
    echo "📋 Next steps:"
    echo "   1. Start the server:  cd server && go run main.go"
    echo "   2. Run the client:    cd client && go run main.go"
    echo ""
    echo "   Or use the Makefile:"
    echo "   - make server  (start server)"
    echo "   - make client  (run client)"
else
    echo "❌ Failed to generate protocol buffer code"
    exit 1
fi

