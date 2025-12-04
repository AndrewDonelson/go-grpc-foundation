package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMetricsInterceptor(t *testing.T) {
	interceptor := metricsInterceptor()

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	// Test successful handler
	successHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, successHandler)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp)

	// Test error handler
	errorHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Errorf(codes.Internal, "test error")
	}

	resp, err = interceptor(ctx, req, info, errorHandler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestMetricsInterceptor_Duration(t *testing.T) {
	interceptor := metricsInterceptor()

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/SlowMethod",
	}

	// Handler that takes some time
	slowHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		time.Sleep(10 * time.Millisecond)
		return "done", nil
	}

	start := time.Now()
	resp, err := interceptor(ctx, req, info, slowHandler)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, "done", resp)
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
}

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
		expected   string
	}{
		{
			name:       "standard format",
			fullMethod: "/hello.v1.HelloService/SayHello",
			expected:   "hello.v1.HelloService",
		},
		{
			name:       "no leading slash",
			fullMethod: "hello.v1.HelloService/SayHello",
			expected:   "hello.v1.HelloService",
		},
		{
			name:       "multiple slashes",
			fullMethod: "/service/v1/Service/Method",
			expected:   "service/v1/Service",
		},
		{
			name:       "no slash in method",
			fullMethod: "ServiceMethod",
			expected:   "ServiceMethod",
		},
		{
			name:       "empty string",
			fullMethod: "",
			expected:   "",
		},
		{
			name:       "only slash",
			fullMethod: "/",
			expected:   "",
		},
		{
			name:       "method at start",
			fullMethod: "/Method",
			expected:   "Method", // When there's no service prefix, return the method name
		},
		{
			name:       "nested service",
			fullMethod: "/com.example.service.v1.UserService/GetUser",
			expected:   "com.example.service.v1.UserService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractServiceName(tt.fullMethod)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMetricsInterceptor_ActiveRequests(t *testing.T) {
	interceptor := metricsInterceptor()

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	// Handler that blocks briefly
	blockingHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
		time.Sleep(5 * time.Millisecond)
		return "response", nil
	}

	// Run multiple concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := interceptor(ctx, req, info, blockingHandler)
			assert.NoError(t, err)
		}()
	}

	wg.Wait()
}

func TestMetricsInterceptor_ErrorCodes(t *testing.T) {
	interceptor := metricsInterceptor()

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	errorCodes := []codes.Code{
		codes.InvalidArgument,
		codes.NotFound,
		codes.PermissionDenied,
		codes.Unauthenticated,
		codes.ResourceExhausted,
		codes.Unimplemented,
		codes.Internal,
		codes.Unavailable,
		codes.DeadlineExceeded,
	}

	for _, code := range errorCodes {
		t.Run(code.String(), func(t *testing.T) {
			errorHandler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, status.Errorf(code, "test error")
			}

			resp, err := interceptor(ctx, req, info, errorHandler)
			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, code, status.Code(err))
		})
	}
}

