# gRPC Communication Types - Complete Example

This example demonstrates all **4 core gRPC communication patterns** plus common patterns built on them.

## 🚀 Quick Start

### Option 1: Using the Setup Script (Recommended)

```bash
cd example
./setup.sh
```

This will:
- Install required Go plugins
- Generate protocol buffer code
- Show you next steps

### Option 2: Manual Setup

#### 1. Install Dependencies

```bash
# Install protoc (if not already installed)
# macOS: brew install protobuf
# Linux: apt-get install protobuf-compiler
# Windows: Download from https://github.com/protocolbuffers/protobuf/releases

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

#### 2. Generate Protocol Buffer Code

```bash
cd example
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/demo/v1/demo.proto
```

#### 3. Start the Server

```bash
cd example/server
go run main.go
```

The server will start on `localhost:50051` with metrics on `localhost:9090`.

#### 4. Run the Client

In a new terminal:

```bash
cd example/client
go run main.go
```

The client will demonstrate all 4 communication types.

### Option 3: Using Makefile

```bash
cd example
make setup    # Install deps and generate proto code
make server   # Start the server (in one terminal)
make client   # Run the client (in another terminal)
```

## 📋 Communication Types Demonstrated

### 1. Unary RPC (Request-Response)
- **Pattern**: Client sends one request → Server sends one response
- **Examples**:
  - `GetUser` - Simple data retrieval
  - `LogEvent` - Fire-and-forget pattern

### 2. Server Streaming RPC
- **Pattern**: Client sends one request → Server sends stream of responses
- **Examples**:
  - `ListUsers` - Paginated results
  - `SubscribeToEvents` - Real-time event streaming

### 3. Client Streaming RPC
- **Pattern**: Client sends stream of requests → Server sends one response
- **Examples**:
  - `UploadData` - File upload simulation
  - `ProcessBatch` - Batch processing

### 4. Bidirectional Streaming RPC
- **Pattern**: Both client and server send streams independently
- **Examples**:
  - `Chat` - Full bidirectional communication
  - `Interactive` - Ping-pong pattern

## 📁 Project Structure

```
example/
├── proto/
│   └── demo/
│       └── v1/
│           └── demo.proto          # Protocol buffer definitions
├── pkg/
│   └── pb/                          # Generated protobuf code (after protoc)
├── server/
│   └── main.go                      # Server implementation
├── client/
│   └── main.go                      # Client implementation
├── config/
│   └── demo.yaml                    # Configuration file
└── README.md                        # This file
```

## 🔧 Configuration

Edit `config/demo.yaml` to customize:

```yaml
environment: development

server:
  port: 50051
  metrics_port: 9090
```

## 📊 Testing with grpcurl

### List Services
```bash
grpcurl -plaintext localhost:50051 list
```

### List Methods
```bash
grpcurl -plaintext localhost:50051 list demo.v1.DemoService
```

### Call Unary RPC
```bash
grpcurl -plaintext -d '{"user_id": "user-1"}' \
  localhost:50051 demo.v1.DemoService/GetUser
```

### Call Server Streaming
```bash
grpcurl -plaintext -d '{"page_size": 5}' \
  localhost:50051 demo.v1.DemoService/ListUsers
```

## 🎯 Use Cases by Pattern

| Pattern | Best For |
|---------|----------|
| **Unary** | REST-like APIs, CRUD operations, authentication |
| **Server Streaming** | Paginated results, real-time updates, event notifications |
| **Client Streaming** | File uploads, batch operations, data ingestion |
| **Bidirectional** | Chat, gaming, real-time collaboration, WebSocket replacement |

## 📝 Code Documentation

All methods include comprehensive in-code documentation explaining:
- The communication pattern
- Use cases
- Characteristics
- Implementation details

## 🔍 Health Checks

```bash
# gRPC health check
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check

# HTTP health check
curl http://localhost:9090/health
```

## 📈 Metrics

View Prometheus metrics:

```bash
curl http://localhost:9090/metrics
```

## 🧪 Fake Data

This example uses **in-memory fake data** - no database or Redis required:
- 5 sample users
- Simulated event broadcasting
- In-memory chat rooms
- Random batch processing results

## 📚 Learn More

See the main project README for:
- Framework features
- Configuration options
- Deployment guides
- Best practices

## 🎓 Understanding the Patterns

### Unary RPC
Simplest pattern, similar to HTTP REST. Client sends one request and waits for one response.

### Server Streaming
Server keeps connection open and sends multiple messages. Client receives them progressively.

### Client Streaming
Client sends multiple messages (like file chunks), server processes them all and sends one summary response.

### Bidirectional Streaming
Most flexible pattern. Both sides can send messages independently at any time, enabling full-duplex communication.

---

**Built with [go-grpc-foundation](https://github.com/AndrewDonelson/go-grpc-foundation)** 🚀

