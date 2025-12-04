# Example: Simple Hello Service

Complete example demonstrating how to use go-grpc-foundation to build a gRPC microservice.

## Project Structure

```
simple-service/
├── proto/
│   └── hello/
│       └── v1/
│           └── hello.proto
├── pkg/
│   └── pb/
│       └── (generated protobuf files)
├── cmd/
│   └── server/
│       └── main.go
├── config/
│   └── hello.yaml
├── go.mod
└── go.sum
```

---

## 1. Protocol Buffer Definition

**File:** `proto/hello/v1/hello.proto`

```protobuf
syntax = "proto3";

package hello.v1;

option go_package = "github.com/yourorg/simple-service/pkg/pb;pb";

service HelloService {
  rpc SayHello(HelloRequest) returns (HelloResponse) {}
  rpc SayGoodbye(GoodbyeRequest) returns (GoodbyeResponse) {}
}

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string message = 1;
  int64 timestamp = 2;
}

message GoodbyeRequest {
  string name = 1;
}

message GoodbyeResponse {
  string message = 1;
}
```

---

## 2. Generate Protobuf Code

```bash
# Install protoc-gen-go and protoc-gen-go-grpc
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/hello/v1/hello.proto
```

---

## 3. Service Implementation

**File:** `cmd/server/main.go`

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/AndrewDonelson/go-grpc-foundation/pkg/config"
	"github.com/AndrewDonelson/go-grpc-foundation/pkg/database"
	"github.com/AndrewDonelson/go-grpc-foundation/pkg/server"
	
	pb "github.com/yourorg/simple-service/pkg/pb"
)

// HelloService implements the HelloService gRPC interface
type HelloService struct {
	pb.UnimplementedHelloServiceServer
	db    *pgxpool.Pool
	redis *redis.Client
}

// NewHelloService creates a new hello service
func NewHelloService(db *pgxpool.Pool, redis *redis.Client) *HelloService {
	return &HelloService{
		db:    db,
		redis: redis,
	}
}

// SayHello implements the SayHello RPC method
func (s *HelloService) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	// Business logic here
	message := "Hello, " + req.Name + "!"
	
	// Example: Store in Redis cache
	if s.redis != nil {
		key := "last_greeting:" + req.Name
		s.redis.Set(ctx, key, message, 5*time.Minute)
	}
	
	return &pb.HelloResponse{
		Message:   message,
		Timestamp: time.Now().Unix(),
	}, nil
}

// SayGoodbye implements the SayGoodbye RPC method
func (s *HelloService) SayGoodbye(ctx context.Context, req *pb.GoodbyeRequest) (*pb.GoodbyeResponse, error) {
	return &pb.GoodbyeResponse{
		Message: "Goodbye, " + req.Name + "!",
	}, nil
}

func main() {
	// Load configuration
	if err := config.Load("./config", "hello"); err != nil {
		log.Printf("Warning: failed to load config: %v", err)
	}
	
	// Create server with configuration
	cfg := server.DefaultConfig("hello-service")
	cfg.Port = config.GetInt("server.port")
	cfg.MetricsPort = config.GetInt("server.metrics_port")
	cfg.Environment = config.GetString("environment")
	
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	
	logger := srv.GetLogger()
	logger.Info("starting hello service")
	
	// Setup PostgreSQL (optional)
	var db *pgxpool.Pool
	if config.GetString("database.host") != "" {
		pgConfig := &database.PostgresConfig{
			Host:           config.GetString("database.host"),
			Port:           config.GetInt("database.port"),
			Database:       config.GetString("database.database"),
			User:           config.GetString("database.user"),
			Password:       config.GetString("database.password"),
			MaxConnections: 25,
			MinConnections: 5,
		}
		
		db, err = database.NewPostgresPool(context.Background(), pgConfig, logger)
		if err != nil {
			log.Fatal(err)
		}
		
		// Register cleanup
		srv.AddShutdownFunc(func(ctx context.Context) error {
			logger.Info("closing database connection")
			db.Close()
			return nil
		})
	}
	
	// Setup Redis (optional)
	var redisClient *redis.Client
	if config.GetString("redis.host") != "" {
		redisConfig := &database.RedisConfig{
			Host: config.GetString("redis.host"),
			Port: config.GetInt("redis.port"),
			DB:   config.GetInt("redis.db"),
		}
		
		redisClient, err = database.NewRedisClient(context.Background(), redisConfig, logger)
		if err != nil {
			log.Fatal(err)
		}
		
		// Register cleanup
		srv.AddShutdownFunc(func(ctx context.Context) error {
			logger.Info("closing redis connection")
			return redisClient.Close()
		})
	}
	
	// Create and register service
	helloSvc := NewHelloService(db, redisClient)
	srv.RegisterService(&pb.HelloService_ServiceDesc, helloSvc)
	
	// Start server (blocks until shutdown signal)
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
```

---

## 4. Configuration File

**File:** `config/hello.yaml`

```yaml
environment: development  # or "production"

server:
  port: 50051
  metrics_port: 9090

database:
  host: localhost
  port: 5432
  database: hello_db
  user: postgres
  password: ${DB_PASSWORD}

redis:
  host: localhost
  port: 6379
  db: 0

logging:
  level: debug
```

---

## 5. Go Module Setup

**File:** `go.mod`

```go
module github.com/yourorg/simple-service

go 1.21

require (
	github.com/AndrewDonelson/go-grpc-foundation v0.1.0
	google.golang.org/grpc v1.59.0
	google.golang.org/protobuf v1.31.0
)
```

```bash
go mod tidy
```

---

## 6. Running the Service

### Development Mode

```bash
# With verbose logging
ENVIRONMENT=development go run cmd/server/main.go
```

**Output:**
```
2024-12-04T12:00:00.000Z	INFO	server/server.go:45	initializing server	{"service": "hello-service", "environment": "development", "host": "0.0.0.0", "port": 50051}
2024-12-04T12:00:00.050Z	INFO	database/postgres.go:67	postgres connection pool created	{"host": "localhost", "port": 5432, "database": "hello_db"}
2024-12-04T12:00:00.100Z	INFO	database/redis.go:45	redis client created	{"host": "localhost", "port": 6379}
2024-12-04T12:00:00.150Z	INFO	server/server.go:125	server listening	{"address": "0.0.0.0:50051"}
2024-12-04T12:00:00.151Z	INFO	cmd/server/main.go:85	starting hello service
```

### Production Mode

```bash
# With errors/warnings only
ENVIRONMENT=production DB_PASSWORD=secret go run cmd/server/main.go
```

---

## 7. Testing the Service

### Using grpcurl

```bash
# Install grpcurl
go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List services
grpcurl -plaintext localhost:50051 list

# List methods
grpcurl -plaintext localhost:50051 list hello.v1.HelloService

# Call SayHello
grpcurl -plaintext -d '{"name": "World"}' \
  localhost:50051 hello.v1.HelloService/SayHello

# Response:
{
  "message": "Hello, World!",
  "timestamp": "1701696000"
}
```

### Using Go Client

```go
package main

import (
	"context"
	"log"
	"time"

	pb "github.com/yourorg/simple-service/pkg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to server
	conn, err := grpc.Dial("localhost:50051",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	
	// Create client
	client := pb.NewHelloServiceClient(conn)
	
	// Call SayHello
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	
	resp, err := client.SayHello(ctx, &pb.HelloRequest{
		Name: "World",
	})
	if err != nil {
		log.Fatal(err)
	}
	
	log.Printf("Response: %s (timestamp: %d)", resp.Message, resp.Timestamp)
}
```

---

## 8. Health Check

```bash
# gRPC health check
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check

# Response:
{
  "status": "SERVING"
}

# HTTP health check
curl http://localhost:9090/health

# Response:
OK
```

---

## 9. Metrics

```bash
# View Prometheus metrics
curl http://localhost:9090/metrics

# Sample output:
# HELP grpc_requests_total Total number of gRPC requests
# TYPE grpc_requests_total counter
grpc_requests_total{code="OK",method="/hello.v1.HelloService/SayHello",service="/hello.v1.HelloService/SayHello"} 42

# HELP grpc_request_duration_seconds gRPC request duration in seconds
# TYPE grpc_request_duration_seconds histogram
grpc_request_duration_seconds_bucket{method="/hello.v1.HelloService/SayHello",service="/hello.v1.HelloService/SayHello",le="0.005"} 38
grpc_request_duration_seconds_bucket{method="/hello.v1.HelloService/SayHello",service="/hello.v1.HelloService/SayHello",le="0.01"} 42
```

---

## 10. Docker Deployment

**File:** `Dockerfile`

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN go build -o /hello-service ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates

COPY --from=builder /hello-service /hello-service
COPY --from=builder /app/config /config

EXPOSE 50051 9090

CMD ["/hello-service"]
```

**Build and run:**

```bash
# Build image
docker build -t hello-service:latest .

# Run container
docker run -p 50051:50051 -p 9090:9090 \
  -e ENVIRONMENT=production \
  -e DB_PASSWORD=secret \
  hello-service:latest
```

---

## 11. Kubernetes Deployment

**File:** `k8s/deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: hello-service
  template:
    metadata:
      labels:
        app: hello-service
    spec:
      containers:
      - name: hello-service
        image: hello-service:latest
        ports:
        - containerPort: 50051
          name: grpc
        - containerPort: 9090
          name: metrics
        env:
        - name: ENVIRONMENT
          value: "production"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        livenessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 9090
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: hello-service
spec:
  selector:
    app: hello-service
  ports:
  - port: 50051
    targetPort: 50051
    name: grpc
  - port: 9090
    targetPort: 9090
    name: metrics
```

**Deploy:**

```bash
kubectl apply -f k8s/deployment.yaml
```

---

## Summary

This example demonstrates:

✅ **Minimal boilerplate** - ~100 lines of code  
✅ **Production-ready** - Health checks, metrics, logging  
✅ **Database integration** - PostgreSQL and Redis  
✅ **Configuration management** - YAML + environment variables  
✅ **Graceful shutdown** - Proper cleanup  
✅ **Docker & Kubernetes** - Ready for deployment  

All powered by **go-grpc-foundation**! 🚀
