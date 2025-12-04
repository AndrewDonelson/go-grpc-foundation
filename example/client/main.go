package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/AndrewDonelson/go-grpc-foundation/example/pkg/pb"
)

const (
	serverAddress = "localhost:50051"
)

func main() {
	// Connect to server
	conn, err := grpc.Dial(serverAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewDemoServiceClient(conn)

	fmt.Println(strings.Repeat("=", 82))
	fmt.Println("gRPC Communication Types Demo Client")
	fmt.Println(strings.Repeat("=", 82))
	fmt.Println()

	// Run all demos
	if err := demoUnary(client); err != nil {
		log.Printf("Unary demo error: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 82) + "\n")

	if err := demoServerStreaming(client); err != nil {
		log.Printf("Server Streaming demo error: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 82) + "\n")

	if err := demoClientStreaming(client); err != nil {
		log.Printf("Client Streaming demo error: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 82) + "\n")

	if err := demoBidirectionalStreaming(client); err != nil {
		log.Printf("Bidirectional Streaming demo error: %v", err)
	}

	fmt.Println("\n" + strings.Repeat("=", 82) + "\n")
	fmt.Println("All demos completed!")
}

// ============================================================================
// 1. UNARY RPC DEMO
// ============================================================================

// demoUnary demonstrates Unary RPC pattern
// Pattern: Client sends one request → Server sends one response
// This is the simplest pattern, similar to HTTP REST API calls
func demoUnary(client pb.DemoServiceClient) error {
	fmt.Println("1. UNARY RPC DEMONSTRATION")
	fmt.Println("   Pattern: Client → Request → Server → Response")
	fmt.Println("   Use Case: Simple data retrieval, REST-like APIs")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Example 1: GetUser - Simple request/response
	fmt.Println("   Example 1: GetUser")
	fmt.Println("   ──────────────────────────────────────────────────────────")
	userResp, err := client.GetUser(ctx, &pb.GetUserRequest{
		UserId: "user-1",
	})
	if err != nil {
		return fmt.Errorf("GetUser failed: %v", err)
	}
	fmt.Printf("   ✓ Received user: %s (%s)\n", userResp.User.Name, userResp.User.Email)
	fmt.Printf("   ✓ Timestamp: %d\n", userResp.Timestamp)
	fmt.Println()

	// Example 2: LogEvent - Fire-and-Forget pattern
	fmt.Println("   Example 2: LogEvent (Fire-and-Forget)")
	fmt.Println("   ──────────────────────────────────────────────────────────")
	_, err = client.LogEvent(ctx, &pb.LogEventRequest{
		EventType: "user.action",
		Message:   "User performed an action",
		Metadata: map[string]string{
			"user_id": "user-1",
			"action":  "view_profile",
		},
	})
	if err != nil {
		return fmt.Errorf("LogEvent failed: %v", err)
	}
	fmt.Println("   ✓ Event logged (empty response received)")
	fmt.Println()

	return nil
}

// ============================================================================
// 2. SERVER STREAMING RPC DEMO
// ============================================================================

// demoServerStreaming demonstrates Server Streaming RPC pattern
// Pattern: Client sends one request → Server sends stream of responses
// The server keeps the connection open and sends multiple messages
func demoServerStreaming(client pb.DemoServiceClient) error {
	fmt.Println("2. SERVER STREAMING RPC DEMONSTRATION")
	fmt.Println("   Pattern: Client → Request → Server → Stream of Responses")
	fmt.Println("   Use Case: Paginated results, real-time updates, event streaming")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Example 1: ListUsers - Server streams multiple users
	fmt.Println("   Example 1: ListUsers")
	fmt.Println("   ──────────────────────────────────────────────────────────")
	stream, err := client.ListUsers(ctx, &pb.ListUsersRequest{
		PageSize: 5,
		Filter:   "active", // Filter for active users only
	})
	if err != nil {
		return fmt.Errorf("ListUsers failed: %v", err)
	}

	fmt.Println("   Receiving users from server stream...")
	userCount := 0
	for {
		user, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("stream receive failed: %v", err)
		}
		userCount++
		fmt.Printf("   [%d] ✓ %s (%s) - Status: %s\n",
			userCount, user.Name, user.Email, user.Status.String())
	}
	fmt.Printf("   ✓ Received %d users from stream\n", userCount)
	fmt.Println()

	// Example 2: SubscribeToEvents - Event streaming
	fmt.Println("   Example 2: SubscribeToEvents (Event Streaming)")
	fmt.Println("   ──────────────────────────────────────────────────────────")
	eventStream, err := client.SubscribeToEvents(ctx, &pb.SubscribeRequest{
		EventTypes: []string{"user.created", "system.alert"}, // Subscribe to specific events
	})
	if err != nil {
		return fmt.Errorf("SubscribeToEvents failed: %v", err)
	}

	fmt.Println("   Subscribed to events. Receiving events for 10 seconds...")
	eventCount := 0
	eventCtx, eventCancel := context.WithTimeout(ctx, 10*time.Second)
	defer eventCancel()

	for {
		select {
		case <-eventCtx.Done():
			fmt.Printf("   ✓ Received %d events\n", eventCount)
			return nil
		default:
			event, err := eventStream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return fmt.Errorf("event stream receive failed: %v", err)
			}
			eventCount++
			fmt.Printf("   [%d] ✓ Event: %s - %s (ID: %s)\n",
				eventCount, event.EventType, event.Message, event.EventId)
		}
	}
}

// ============================================================================
// 3. CLIENT STREAMING RPC DEMO
// ============================================================================

// demoClientStreaming demonstrates Client Streaming RPC pattern
// Pattern: Client sends stream of requests → Server sends one response
// The client sends multiple messages and the server processes them all
func demoClientStreaming(client pb.DemoServiceClient) error {
	fmt.Println("3. CLIENT STREAMING RPC DEMONSTRATION")
	fmt.Println("   Pattern: Client → Stream of Requests → Server → Response")
	fmt.Println("   Use Case: File uploads, batch operations, data ingestion")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Example 1: UploadData - Client streams data chunks
	fmt.Println("   Example 1: UploadData")
	fmt.Println("   ──────────────────────────────────────────────────────────")
	uploadStream, err := client.UploadData(ctx)
	if err != nil {
		return fmt.Errorf("UploadData failed: %v", err)
	}

	// Simulate uploading 5 chunks
	chunks := []string{
		"Chunk 1 data: Lorem ipsum dolor sit amet",
		"Chunk 2 data: consectetur adipiscing elit",
		"Chunk 3 data: sed do eiusmod tempor",
		"Chunk 4 data: incididunt ut labore",
		"Chunk 5 data: et dolore magna aliqua",
	}

	fmt.Printf("   Sending %d chunks to server...\n", len(chunks))
	for i, chunk := range chunks {
		err := uploadStream.Send(&pb.DataChunk{
			ChunkNumber: int32(i + 1),
			Data:        []byte(chunk),
			IsLast:      i == len(chunks)-1,
		})
		if err != nil {
			return fmt.Errorf("failed to send chunk: %v", err)
		}
		fmt.Printf("   [%d] ✓ Sent chunk %d (%d bytes)\n", i+1, i+1, len(chunk))
		time.Sleep(200 * time.Millisecond) // Simulate network delay
	}

	// Close and receive response
	response, err := uploadStream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("failed to receive response: %v", err)
	}
	fmt.Printf("   ✓ Upload complete: %s\n", response.Message)
	fmt.Printf("   ✓ Total bytes: %d, Chunks: %d\n", response.TotalBytes, response.ChunksReceived)
	fmt.Println()

	// Example 2: ProcessBatch - Batch processing
	fmt.Println("   Example 2: ProcessBatch (Batch Processing)")
	fmt.Println("   ──────────────────────────────────────────────────────────")
	batchStream, err := client.ProcessBatch(ctx)
	if err != nil {
		return fmt.Errorf("ProcessBatch failed: %v", err)
	}

	// Send 10 batch items
	batchSize := 10
	fmt.Printf("   Sending %d batch items to server...\n", batchSize)
	for i := 0; i < batchSize; i++ {
		item := &pb.BatchItem{
			ItemId: fmt.Sprintf("item-%d", i+1),
			Data:   fmt.Sprintf("Batch item data %d", i+1),
			Metadata: map[string]string{
				"index": fmt.Sprintf("%d", i+1),
				"type":  "demo",
			},
		}
		if err := batchStream.Send(item); err != nil {
			return fmt.Errorf("failed to send batch item: %v", err)
		}
		fmt.Printf("   [%d] ✓ Sent batch item: %s\n", i+1, item.ItemId)
		time.Sleep(100 * time.Millisecond)
	}

	// Close and receive batch result
	batchResult, err := batchStream.CloseAndRecv()
	if err != nil {
		return fmt.Errorf("failed to receive batch result: %v", err)
	}
	fmt.Printf("   ✓ Batch processing complete\n")
	fmt.Printf("   ✓ Items processed: %d\n", batchResult.ItemsProcessed)
	fmt.Printf("   ✓ Items succeeded: %d\n", batchResult.ItemsSucceeded)
	fmt.Printf("   ✓ Items failed: %d\n", batchResult.ItemsFailed)
	if len(batchResult.Errors) > 0 {
		fmt.Printf("   ⚠ Errors: %v\n", batchResult.Errors)
	}
	fmt.Println()

	return nil
}

// ============================================================================
// 4. BIDIRECTIONAL STREAMING RPC DEMO
// ============================================================================

// demoBidirectionalStreaming demonstrates Bidirectional Streaming RPC pattern
// Pattern: Both client and server send streams independently
// This is the most flexible pattern, enabling full-duplex communication
func demoBidirectionalStreaming(client pb.DemoServiceClient) error {
	fmt.Println("4. BIDIRECTIONAL STREAMING RPC DEMONSTRATION")
	fmt.Println("   Pattern: Client ↔ Stream ↔ Server (both send independently)")
	fmt.Println("   Use Case: Chat, gaming, real-time collaboration, WebSocket replacement")
	fmt.Println()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Example 1: Chat - Full bidirectional communication
	fmt.Println("   Example 1: Chat (Bidirectional Streaming)")
	fmt.Println("   ──────────────────────────────────────────────────────────")
	chatStream, err := client.Chat(ctx)
	if err != nil {
		return fmt.Errorf("Chat failed: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine to send messages
	go func() {
		defer wg.Done()
		messages := []string{
			"Hello, server!",
			"How are you?",
			"This is a bidirectional stream demo",
			"Both sides can send messages independently",
		}

		for i, text := range messages {
			msg := &pb.ChatMessage{
				MessageId: fmt.Sprintf("msg-%d", i+1),
				UserId:    "client-user",
				Text:      text,
				Timestamp: time.Now().Unix(),
				Type:      pb.MessageType_MESSAGE_TYPE_USER,
			}

			if err := chatStream.Send(msg); err != nil {
				log.Printf("Failed to send chat message: %v", err)
				return
			}
			fmt.Printf("   → Sent: %s\n", text)
			time.Sleep(1 * time.Second)
		}

		// Close send side
		if err := chatStream.CloseSend(); err != nil {
			log.Printf("Failed to close send: %v", err)
		}
	}()

	// Goroutine to receive messages
	go func() {
		defer wg.Done()
		messageCount := 0
		for {
			msg, err := chatStream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Failed to receive chat message: %v", err)
				return
			}
			messageCount++
			fmt.Printf("   ← Received: %s (from %s)\n", msg.Text, msg.UserId)
		}
		fmt.Printf("   ✓ Received %d messages from server\n", messageCount)
	}()

	wg.Wait()
	fmt.Println()

	// Example 2: Interactive - Ping-Pong pattern
	fmt.Println("   Example 2: Interactive (Ping-Pong Pattern)")
	fmt.Println("   ──────────────────────────────────────────────────────────")
	interactiveStream, err := client.Interactive(ctx)
	if err != nil {
		return fmt.Errorf("Interactive failed: %v", err)
	}

	var interactiveWg sync.WaitGroup
	interactiveWg.Add(2)

	// Goroutine to send requests
	go func() {
		defer interactiveWg.Done()
		commands := []struct {
			cmd       string
			params    map[string]string
			requestID string
		}{
			{"ping", nil, "req-1"},
			{"echo", map[string]string{"text": "Hello World"}, "req-2"},
			{"time", nil, "req-3"},
			{"users", nil, "req-4"},
			{"unknown", nil, "req-5"}, // This should fail
		}

		for _, cmd := range commands {
			req := &pb.Request{
				RequestId:  cmd.requestID,
				Command:    cmd.cmd,
				Parameters: cmd.params,
			}

			if err := interactiveStream.Send(req); err != nil {
				log.Printf("Failed to send request: %v", err)
				return
			}
			fmt.Printf("   → Sent command: %s (ID: %s)\n", cmd.cmd, cmd.requestID)
			time.Sleep(500 * time.Millisecond)
		}

		if err := interactiveStream.CloseSend(); err != nil {
			log.Printf("Failed to close send: %v", err)
		}
	}()

	// Goroutine to receive responses
	go func() {
		defer interactiveWg.Done()
		responseCount := 0
		for {
			resp, err := interactiveStream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("Failed to receive response: %v", err)
				return
			}
			responseCount++
			if resp.Success {
				fmt.Printf("   ← Response [%s]: %s\n", resp.RequestId, resp.Result)
			} else {
				fmt.Printf("   ← Error [%s]: %s\n", resp.RequestId, resp.ErrorMessage)
			}
		}
		fmt.Printf("   ✓ Received %d responses\n", responseCount)
	}()

	interactiveWg.Wait()
	fmt.Println()

	return nil
}
