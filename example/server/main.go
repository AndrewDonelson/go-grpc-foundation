package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/AndrewDonelson/go-grpc-foundation/pkg/config"
	"github.com/AndrewDonelson/go-grpc-foundation/pkg/server"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/AndrewDonelson/go-grpc-foundation/example/pkg/pb"
)

// DemoService implements all 4 gRPC communication patterns
// This service demonstrates:
// 1. Unary RPC - Simple request/response
// 2. Server Streaming - Server sends multiple responses
// 3. Client Streaming - Client sends multiple requests
// 4. Bidirectional Streaming - Both send streams independently
type DemoService struct {
	pb.UnimplementedDemoServiceServer

	logger *zap.Logger
	// In-memory fake data stores (no real database)
	users      map[string]*pb.User
	usersMutex sync.RWMutex
	events     chan *pb.Event
	chatRooms  map[string][]*pb.ChatMessage
	chatMutex  sync.RWMutex
}

// NewDemoService creates a new demo service with fake data
func NewDemoService(logger *zap.Logger) *DemoService {
	service := &DemoService{
		logger:    logger,
		users:     make(map[string]*pb.User),
		events:    make(chan *pb.Event, 100),
		chatRooms: make(map[string][]*pb.ChatMessage),
	}

	// Initialize with some fake users
	service.initFakeData()

	// Start event broadcaster
	go service.broadcastEvents()

	return service
}

// initFakeData populates the service with sample data
func (s *DemoService) initFakeData() {
	fakeUsers := []*pb.User{
		{
			UserId:    "user-1",
			Name:      "Alice Johnson",
			Email:     "alice@example.com",
			CreatedAt: time.Now().Add(-30 * 24 * time.Hour).Unix(),
			Status:    pb.UserStatus_USER_STATUS_ACTIVE,
		},
		{
			UserId:    "user-2",
			Name:      "Bob Smith",
			Email:     "bob@example.com",
			CreatedAt: time.Now().Add(-60 * 24 * time.Hour).Unix(),
			Status:    pb.UserStatus_USER_STATUS_ACTIVE,
		},
		{
			UserId:    "user-3",
			Name:      "Charlie Brown",
			Email:     "charlie@example.com",
			CreatedAt: time.Now().Add(-90 * 24 * time.Hour).Unix(),
			Status:    pb.UserStatus_USER_STATUS_INACTIVE,
		},
		{
			UserId:    "user-4",
			Name:      "Diana Prince",
			Email:     "diana@example.com",
			CreatedAt: time.Now().Add(-15 * 24 * time.Hour).Unix(),
			Status:    pb.UserStatus_USER_STATUS_ACTIVE,
		},
		{
			UserId:    "user-5",
			Name:      "Eve Wilson",
			Email:     "eve@example.com",
			CreatedAt: time.Now().Add(-45 * 24 * time.Hour).Unix(),
			Status:    pb.UserStatus_USER_STATUS_SUSPENDED,
		},
	}

	s.usersMutex.Lock()
	for _, user := range fakeUsers {
		s.users[user.UserId] = user
	}
	s.usersMutex.Unlock()
}

// broadcastEvents simulates event broadcasting (for demonstration)
func (s *DemoService) broadcastEvents() {
	eventTypes := []string{"user.created", "user.updated", "system.alert", "notification"}
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			event := &pb.Event{
				EventId:   fmt.Sprintf("event-%d", time.Now().Unix()),
				EventType: eventTypes[rand.Intn(len(eventTypes))],
				Message:   fmt.Sprintf("Sample event: %s", eventTypes[rand.Intn(len(eventTypes))]),
				Timestamp: time.Now().Unix(),
				Data: map[string]string{
					"source": "demo-service",
					"random": fmt.Sprintf("%d", rand.Intn(1000)),
				},
			}
			select {
			case s.events <- event:
			default:
				// Channel full, skip
			}
		}
	}
}

// ============================================================================
// 1. UNARY RPC IMPLEMENTATIONS
// ============================================================================

// GetUser implements Unary RPC pattern
// Pattern: Client sends one request → Server sends one response
// Use Case: Simple data retrieval, REST-like API calls
//
// This is the simplest gRPC pattern, similar to HTTP GET requests.
// The client sends a single request and waits for a single response.
func (s *DemoService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	s.logger.Info("GetUser called",
		zap.String("user_id", req.UserId),
	)

	s.usersMutex.RLock()
	user, exists := s.users[req.UserId]
	s.usersMutex.RUnlock()

	if !exists {
		return nil, status.Errorf(codes.NotFound, "user with id %s not found", req.UserId)
	}

	return &pb.GetUserResponse{
		User:      user,
		Timestamp: time.Now().Unix(),
	}, nil
}

// LogEvent implements Fire-and-Forget pattern (built on Unary)
// Pattern: Client sends one request → Server sends empty response
// Use Case: Logging, metrics, non-blocking notifications
//
// The client doesn't wait for meaningful data in the response.
// This is useful for operations where the client doesn't need to know the result.
func (s *DemoService) LogEvent(ctx context.Context, req *pb.LogEventRequest) (*emptypb.Empty, error) {
	s.logger.Info("LogEvent called",
		zap.String("event_type", req.EventType),
		zap.String("message", req.Message),
		zap.Any("metadata", req.Metadata),
	)

	// In a real application, you would log this to your logging system
	// For demo purposes, we just log it via zap

	return &emptypb.Empty{}, nil
}

// ============================================================================
// 2. SERVER STREAMING RPC IMPLEMENTATIONS
// ============================================================================

// ListUsers implements Server Streaming RPC pattern
// Pattern: Client sends one request → Server sends stream of responses
// Use Case: Paginated results, large datasets, progressive data delivery
//
// The server keeps the connection open and sends multiple messages.
// The client receives them as a stream, processing each as it arrives.
// This is efficient for large datasets that don't fit in a single response.
func (s *DemoService) ListUsers(req *pb.ListUsersRequest, stream pb.DemoService_ListUsersServer) error {
	s.logger.Info("ListUsers called",
		zap.Int32("page_size", req.PageSize),
		zap.String("filter", req.Filter),
	)

	s.usersMutex.RLock()
	users := make([]*pb.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	s.usersMutex.RUnlock()

	// Apply filter if provided
	if req.Filter != "" {
		filtered := make([]*pb.User, 0)
		for _, user := range users {
			if req.Filter == "active" && user.Status == pb.UserStatus_USER_STATUS_ACTIVE {
				filtered = append(filtered, user)
			} else if req.Filter == "inactive" && user.Status == pb.UserStatus_USER_STATUS_INACTIVE {
				filtered = append(filtered, user)
			}
		}
		users = filtered
	}

	// Stream users one by one
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10 // Default
	}

	for i, user := range users {
		if i >= pageSize {
			break
		}

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)

		if err := stream.Send(user); err != nil {
			s.logger.Error("failed to send user", zap.Error(err))
			return err
		}

		s.logger.Debug("sent user",
			zap.String("user_id", user.UserId),
			zap.Int("index", i),
		)
	}

	return nil
}

// SubscribeToEvents implements Event Streaming pattern (built on Server Streaming)
// Pattern: Client sends one request → Server sends continuous stream of events
// Use Case: Real-time notifications, pub/sub, live updates, WebSocket replacement
//
// The server maintains a long-lived connection and pushes events as they occur.
// The client receives events in real-time without polling.
// This is ideal for event-driven architectures and real-time applications.
func (s *DemoService) SubscribeToEvents(req *pb.SubscribeRequest, stream pb.DemoService_SubscribeToEventsServer) error {
	s.logger.Info("SubscribeToEvents called",
		zap.Strings("event_types", req.EventTypes),
	)

	// Create a channel to receive events
	eventChan := make(chan *pb.Event, 10)
	defer close(eventChan)

	// Start goroutine to forward events
	go func() {
		for event := range s.events {
			// Filter by event types if specified
			if len(req.EventTypes) > 0 {
				found := false
				for _, eventType := range req.EventTypes {
					if event.EventType == eventType {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}

			select {
			case eventChan <- event:
			case <-stream.Context().Done():
				return
			}
		}
	}()

	// Stream events to client
	for {
		select {
		case event := <-eventChan:
			if err := stream.Send(event); err != nil {
				s.logger.Error("failed to send event", zap.Error(err))
				return err
			}
			s.logger.Debug("sent event",
				zap.String("event_id", event.EventId),
				zap.String("event_type", event.EventType),
			)

		case <-stream.Context().Done():
			s.logger.Info("client disconnected from event stream")
			return nil
		}
	}
}

// ============================================================================
// 3. CLIENT STREAMING RPC IMPLEMENTATIONS
// ============================================================================

// UploadData implements Client Streaming RPC pattern
// Pattern: Client sends stream of requests → Server sends one response
// Use Case: File uploads, batch data ingestion, progressive data collection
//
// The client sends multiple messages (like file chunks) and the server
// processes them all before sending a single response with the result.
// This is efficient for uploading large files or sending batch data.
func (s *DemoService) UploadData(stream pb.DemoService_UploadDataServer) error {
	s.logger.Info("UploadData called")

	var (
		totalBytes     int64
		chunksReceived int32
		chunkNumbers   []int32
	)

	// Receive all chunks from client
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			// Client finished sending
			break
		}
		if err != nil {
			s.logger.Error("failed to receive chunk", zap.Error(err))
			return err
		}

		chunksReceived++
		totalBytes += int64(len(chunk.Data))
		chunkNumbers = append(chunkNumbers, chunk.ChunkNumber)

		s.logger.Debug("received chunk",
			zap.Int32("chunk_number", chunk.ChunkNumber),
			zap.Int("data_size", len(chunk.Data)),
			zap.Bool("is_last", chunk.IsLast),
		)

		// Check if this is the last chunk
		if chunk.IsLast {
			break
		}
	}

	// Send single response with summary
	response := &pb.UploadResponse{
		Success:        true,
		TotalBytes:     totalBytes,
		ChunksReceived: chunksReceived,
		Message:        fmt.Sprintf("Successfully received %d chunks (%d bytes)", chunksReceived, totalBytes),
	}

	s.logger.Info("upload complete",
		zap.Int32("chunks_received", chunksReceived),
		zap.Int64("total_bytes", totalBytes),
	)

	return stream.SendAndClose(response)
}

// ProcessBatch implements Batch Processing pattern (built on Client Streaming)
// Pattern: Client sends stream of items → Server processes all → Returns summary
// Use Case: Bulk operations, batch imports, data aggregation, analytics
//
// The client sends multiple items in a stream, and the server processes
// them all before returning a single result with statistics.
// This is ideal for batch operations where you need to process many items.
func (s *DemoService) ProcessBatch(stream pb.DemoService_ProcessBatchServer) error {
	s.logger.Info("ProcessBatch called")

	var (
		itemsProcessed int32
		itemsSucceeded int32
		itemsFailed    int32
		errors         []string
	)

	// Receive and process all batch items
	for {
		item, err := stream.Recv()
		if err == io.EOF {
			// Client finished sending
			break
		}
		if err != nil {
			s.logger.Error("failed to receive batch item", zap.Error(err))
			return err
		}

		itemsProcessed++

		// Simulate processing (with random success/failure for demo)
		success := rand.Float32() > 0.2 // 80% success rate
		if success {
			itemsSucceeded++
			s.logger.Debug("processed batch item",
				zap.String("item_id", item.ItemId),
				zap.Bool("success", true),
			)
		} else {
			itemsFailed++
			errorMsg := fmt.Sprintf("failed to process item %s", item.ItemId)
			errors = append(errors, errorMsg)
			s.logger.Debug("failed to process batch item",
				zap.String("item_id", item.ItemId),
			)
		}

		// Simulate processing time
		time.Sleep(50 * time.Millisecond)
	}

	// Send single response with batch results
	response := &pb.BatchResult{
		Success:        itemsFailed == 0,
		ItemsProcessed: itemsProcessed,
		ItemsSucceeded: itemsSucceeded,
		ItemsFailed:    itemsFailed,
		Errors:         errors,
	}

	s.logger.Info("batch processing complete",
		zap.Int32("items_processed", itemsProcessed),
		zap.Int32("items_succeeded", itemsSucceeded),
		zap.Int32("items_failed", itemsFailed),
	)

	return stream.SendAndClose(response)
}

// ============================================================================
// 4. BIDIRECTIONAL STREAMING RPC IMPLEMENTATIONS
// ============================================================================

// Chat implements Bidirectional Streaming RPC pattern
// Pattern: Both client and server send streams independently
// Use Case: Chat applications, real-time collaboration, gaming, WebSocket replacement
//
// This is the most flexible pattern. Both sides can send messages at any time.
// The streams are independent - you can read and write concurrently.
// This enables full-duplex communication similar to WebSockets.
func (s *DemoService) Chat(stream pb.DemoService_ChatServer) error {
	s.logger.Info("Chat called")

	// Use a channel to handle incoming messages
	messageChan := make(chan *pb.ChatMessage, 10)
	defer close(messageChan)

	// Goroutine to receive messages from client
	go func() {
		for {
			msg, err := stream.Recv()
			if err == io.EOF {
				close(messageChan)
				return
			}
			if err != nil {
				s.logger.Error("failed to receive chat message", zap.Error(err))
				close(messageChan)
				return
			}

			s.logger.Debug("received chat message",
				zap.String("message_id", msg.MessageId),
				zap.String("user_id", msg.UserId),
				zap.String("text", msg.Text),
			)

			// Store message in chat room (fake persistence)
			s.chatMutex.Lock()
			roomID := "default"
			s.chatRooms[roomID] = append(s.chatRooms[roomID], msg)
			if len(s.chatRooms[roomID]) > 100 {
				// Keep only last 100 messages
				s.chatRooms[roomID] = s.chatRooms[roomID][len(s.chatRooms[roomID])-100:]
			}
			s.chatMutex.Unlock()

			messageChan <- msg
		}
	}()

	// Main loop: receive messages and send responses
	for {
		select {
		case msg, ok := <-messageChan:
			if !ok {
				// Channel closed, client disconnected
				s.logger.Info("client disconnected from chat")
				return nil
			}

			// Echo the message back with a prefix
			response := &pb.ChatMessage{
				MessageId: fmt.Sprintf("echo-%d", time.Now().UnixNano()),
				UserId:    "server",
				Text:      fmt.Sprintf("Echo: %s", msg.Text),
				Timestamp: time.Now().Unix(),
				Type:      pb.MessageType_MESSAGE_TYPE_ECHO,
			}

			if err := stream.Send(response); err != nil {
				s.logger.Error("failed to send chat response", zap.Error(err))
				return err
			}

			s.logger.Debug("sent chat response",
				zap.String("message_id", response.MessageId),
			)

		case <-stream.Context().Done():
			s.logger.Info("chat stream context cancelled")
			return nil
		}
	}
}

// Interactive implements Ping-Pong pattern (built on Bidirectional Streaming)
// Pattern: Client and server alternate messages in a request-response chain
// Use Case: Turn-based games, interactive shells, conversational AI, command processing
//
// This pattern uses bidirectional streaming but with a structured flow where
// each client message expects a corresponding server response.
// It's like a conversation where each message gets a reply.
func (s *DemoService) Interactive(stream pb.DemoService_InteractiveServer) error {
	s.logger.Info("Interactive called")

	// Receive requests and send responses in a ping-pong fashion
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// Client finished sending
			return nil
		}
		if err != nil {
			s.logger.Error("failed to receive request", zap.Error(err))
			return err
		}

		s.logger.Debug("received interactive request",
			zap.String("request_id", req.RequestId),
			zap.String("command", req.Command),
		)

		// Process command and generate response
		var result string
		var success bool
		var errorMsg string

		switch req.Command {
		case "ping":
			result = "pong"
			success = true
		case "echo":
			if text, ok := req.Parameters["text"]; ok {
				result = text
				success = true
			} else {
				errorMsg = "missing 'text' parameter"
				success = false
			}
		case "time":
			result = time.Now().Format(time.RFC3339)
			success = true
		case "users":
			s.usersMutex.RLock()
			result = fmt.Sprintf("Total users: %d", len(s.users))
			s.usersMutex.RUnlock()
			success = true
		default:
			errorMsg = fmt.Sprintf("unknown command: %s", req.Command)
			success = false
		}

		// Send response
		response := &pb.Response{
			RequestId:    req.RequestId,
			Result:       result,
			Success:      success,
			ErrorMessage: errorMsg,
		}

		if err := stream.Send(response); err != nil {
			s.logger.Error("failed to send response", zap.Error(err))
			return err
		}

		s.logger.Debug("sent interactive response",
			zap.String("request_id", req.RequestId),
			zap.Bool("success", success),
		)
	}
}

func main() {
	// Load configuration (optional - will use defaults if not found)
	if err := config.Load("./config", "demo"); err != nil {
		log.Printf("Warning: failed to load config: %v (using defaults)", err)
	}

	// Create server configuration
	cfg := server.DefaultConfig("demo-service")
	cfg.Port = config.GetInt("server.port")
	if cfg.Port == 0 {
		cfg.Port = 50051 // Default port
	}
	cfg.MetricsPort = config.GetInt("server.metrics_port")
	if cfg.MetricsPort == 0 {
		cfg.MetricsPort = 9090 // Default metrics port
	}
	cfg.Environment = config.GetString("environment")
	if cfg.Environment == "" {
		cfg.Environment = "development"
	}
	cfg.EnableReflection = true // Enable for demo purposes
	cfg.EnableMetrics = true    // Enable metrics
	cfg.ShutdownTimeout = 10 * time.Second

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	logger := srv.GetLogger()
	logger.Info("starting demo service",
		zap.Int("port", cfg.Port),
		zap.Int("metrics_port", cfg.MetricsPort),
	)

	// Create and register demo service
	demoSvc := NewDemoService(logger)
	srv.RegisterService(&pb.DemoService_ServiceDesc, demoSvc)

	logger.Info("demo service registered",
		zap.String("service", "demo.v1.DemoService"),
		zap.Int("methods", 8), // 8 RPC methods
	)

	// Start server (blocks until shutdown signal)
	if err := srv.Start(); err != nil {
		logger.Fatal("server failed", zap.Error(err))
	}
}
