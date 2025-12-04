package middleware

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RateLimiter implements a simple token bucket rate limiter
type RateLimiter struct {
	mu          sync.Mutex
	tokens      map[string]*tokenBucket
	maxTokens   int
	refillRate  time.Duration
	cleanupTick *time.Ticker
	logger      *zap.Logger
}

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(maxTokens int, refillRate time.Duration, logger *zap.Logger) *RateLimiter {
	rl := &RateLimiter{
		tokens:      make(map[string]*tokenBucket),
		maxTokens:   maxTokens,
		refillRate:  refillRate,
		cleanupTick: time.NewTicker(1 * time.Minute),
		logger:      logger,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request is allowed for the given key
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	bucket, exists := rl.tokens[key]
	if !exists {
		bucket = &tokenBucket{
			tokens:     rl.maxTokens,
			lastRefill: time.Now(),
		}
		rl.tokens[key] = bucket
	}
	rl.mu.Unlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens if needed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	if elapsed >= rl.refillRate {
		refills := int(elapsed / rl.refillRate)
		bucket.tokens = bucket.tokens + refills
		if bucket.tokens > rl.maxTokens {
			bucket.tokens = rl.maxTokens
		}
		bucket.lastRefill = now
	}

	// Check if we have tokens
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// cleanup removes old buckets periodically
func (rl *RateLimiter) cleanup() {
	for range rl.cleanupTick.C {
		rl.mu.Lock()
		for key, bucket := range rl.tokens {
			bucket.mu.Lock()
			// Remove if bucket is full and hasn't been used in 5 minutes
			if bucket.tokens == rl.maxTokens && time.Since(bucket.lastRefill) > 5*time.Minute {
				delete(rl.tokens, key)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// Stop stops the rate limiter cleanup
func (rl *RateLimiter) Stop() {
	rl.cleanupTick.Stop()
}

// RateLimitInterceptor provides rate limiting for gRPC requests
func RateLimitInterceptor(limiter *RateLimiter, getKey func(context.Context, *grpc.UnaryServerInfo) string) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		key := getKey(ctx, info)
		if !limiter.Allow(key) {
			return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

// DefaultGetKey extracts a rate limit key from context (e.g., IP address or user ID)
func DefaultGetKey(ctx context.Context, info *grpc.UnaryServerInfo) string {
	// Default: use method name as key
	// In production, you'd extract IP or user ID from metadata
	return info.FullMethod
}

