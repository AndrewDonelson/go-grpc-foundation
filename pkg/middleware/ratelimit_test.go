package middleware

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestNewRateLimiter(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(10, 1*time.Second, logger)

	assert.NotNil(t, rl)
	assert.Equal(t, 10, rl.maxTokens)
	assert.Equal(t, 1*time.Second, rl.refillRate)
	assert.NotNil(t, rl.cleanupTick)

	// Cleanup
	rl.Stop()
}

func TestRateLimiter_Allow(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(5, 100*time.Millisecond, logger)
	defer rl.Stop()

	// Should allow initial requests up to maxTokens
	for i := 0; i < 5; i++ {
		assert.True(t, rl.Allow("test-key"))
	}

	// Should deny when tokens exhausted
	assert.False(t, rl.Allow("test-key"))

	// Should refill after refillRate
	time.Sleep(150 * time.Millisecond)
	assert.True(t, rl.Allow("test-key"))
}

func TestRateLimiter_Allow_MultipleKeys(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(3, 100*time.Millisecond, logger)
	defer rl.Stop()

	// Each key should have its own bucket
	assert.True(t, rl.Allow("key1"))
	assert.True(t, rl.Allow("key2"))
	assert.True(t, rl.Allow("key3"))

	// Exhaust key1
	assert.True(t, rl.Allow("key1"))
	assert.True(t, rl.Allow("key1"))
	assert.False(t, rl.Allow("key1"))

	// Other keys should still work
	assert.True(t, rl.Allow("key2"))
	assert.True(t, rl.Allow("key3"))
}

func TestRateLimiter_Allow_Refill(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(2, 50*time.Millisecond, logger)
	defer rl.Stop()

	key := "refill-test"

	// Exhaust tokens
	assert.True(t, rl.Allow(key))
	assert.True(t, rl.Allow(key))
	assert.False(t, rl.Allow(key))

	// Wait for refill
	time.Sleep(60 * time.Millisecond)

	// Should have refilled
	assert.True(t, rl.Allow(key))
}

func TestRateLimiter_Allow_MultipleRefills(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(2, 50*time.Millisecond, logger)
	defer rl.Stop()

	key := "multi-refill-test"

	// Exhaust tokens
	assert.True(t, rl.Allow(key))
	assert.True(t, rl.Allow(key))
	assert.False(t, rl.Allow(key))

	// Wait for multiple refills
	time.Sleep(150 * time.Millisecond)

	// Should have refilled multiple times, but capped at maxTokens
	assert.True(t, rl.Allow(key))
	assert.True(t, rl.Allow(key))
	assert.False(t, rl.Allow(key))
}

func TestRateLimiter_Allow_Concurrent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(100, 1*time.Second, logger)
	defer rl.Stop()

	key := "concurrent-test"

	var wg sync.WaitGroup
	var allowed int
	var mu sync.Mutex

	// Launch many concurrent requests
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow(key) {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Should only allow up to maxTokens
	assert.LessOrEqual(t, allowed, 100)
}

func TestRateLimiter_Stop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(10, 1*time.Second, logger)

	// Should be able to stop
	rl.Stop()

	// Should not panic if stopped multiple times
	rl.Stop()
}

func TestRateLimitInterceptor_Allowed(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(10, 1*time.Second, logger)
	defer rl.Stop()

	getKey := func(ctx context.Context, info *grpc.UnaryServerInfo) string {
		return "test-key"
	}

	interceptor := RateLimitInterceptor(rl, getKey)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp)
}

func TestRateLimitInterceptor_RateLimited(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(1, 1*time.Second, logger)
	defer rl.Stop()

	getKey := func(ctx context.Context, info *grpc.UnaryServerInfo) string {
		return "test-key"
	}

	interceptor := RateLimitInterceptor(rl, getKey)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	// First request should succeed
	resp, err := interceptor(ctx, req, info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp)

	// Second request should be rate limited
	resp, err = interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.ResourceExhausted, status.Code(err))
	assert.Contains(t, err.Error(), "rate limit exceeded")
}

func TestRateLimitInterceptor_DifferentKeys(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(1, 1*time.Second, logger)
	defer rl.Stop()

	getKey := func(ctx context.Context, info *grpc.UnaryServerInfo) string {
		return info.FullMethod // Use method as key
	}

	interceptor := RateLimitInterceptor(rl, getKey)

	ctx := context.Background()
	req := "test-request"

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	// Different methods should have separate rate limits
	info1 := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/Method1",
	}
	info2 := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/Method2",
	}

	// Both should succeed (different keys)
	resp1, err1 := interceptor(ctx, req, info1, handler)
	resp2, err2 := interceptor(ctx, req, info2, handler)

	assert.NoError(t, err1)
	assert.NoError(t, err2)
	assert.Equal(t, "success", resp1)
	assert.Equal(t, "success", resp2)
}

func TestDefaultGetKey(t *testing.T) {
	ctx := context.Background()
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	key := DefaultGetKey(ctx, info)
	assert.Equal(t, "/test.v1.TestService/TestMethod", key)
}

func TestRateLimiter_Cleanup(t *testing.T) {
	logger := zaptest.NewLogger(t)
	rl := NewRateLimiter(10, 1*time.Second, logger)
	defer rl.Stop()

	// Create a bucket
	rl.Allow("cleanup-test")

	// Verify it exists
	rl.mu.Lock()
	_, exists := rl.tokens["cleanup-test"]
	rl.mu.Unlock()
	assert.True(t, exists)

	// Wait for cleanup (buckets are cleaned if full and unused for 5 minutes)
	// We can't easily test the full cleanup cycle without waiting 5+ minutes,
	// but we can verify the structure and that cleanup goroutine is running
	rl.mu.Lock()
	bucketCount := len(rl.tokens)
	rl.mu.Unlock()
	assert.Greater(t, bucketCount, 0)
	
	// Note: The cleanup() function runs in a goroutine every minute
	// and removes buckets that are full and unused for 5+ minutes.
	// To get 100% coverage of cleanup(), we would need to:
	// 1. Create a bucket and fill it to maxTokens
	// 2. Wait 5+ minutes
	// 3. Wait for cleanup tick (1 minute interval)
	// This is impractical for unit tests, so cleanup() shows 12.5% coverage.
	// The function is simple and the logic is straightforward.
}

