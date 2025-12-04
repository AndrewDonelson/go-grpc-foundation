package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor_HealthCheck(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validateToken := func(token string) (bool, error) {
		return false, errors.New("should not be called")
	}

	interceptor := AuthInterceptor(logger, validateToken)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.NoError(t, err)
	assert.Equal(t, "success", resp)
}

func TestAuthInterceptor_ValidToken(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validateToken := func(token string) (bool, error) {
		return token == "valid-token", nil
	}

	interceptor := AuthInterceptor(logger, validateToken)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"authorization": "Bearer valid-token",
	}))
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

func TestAuthInterceptor_InvalidToken(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validateToken := func(token string) (bool, error) {
		return token == "valid-token", nil
	}

	interceptor := AuthInterceptor(logger, validateToken)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"authorization": "Bearer invalid-token",
	}))
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_NoMetadata(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validateToken := func(token string) (bool, error) {
		return true, nil
	}

	interceptor := AuthInterceptor(logger, validateToken)

	ctx := context.Background() // No metadata
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.Contains(t, err.Error(), "missing metadata")
}

func TestAuthInterceptor_NoAuthorizationHeader(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validateToken := func(token string) (bool, error) {
		return true, nil
	}

	interceptor := AuthInterceptor(logger, validateToken)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{}))
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.Contains(t, err.Error(), "missing authorization header")
}

func TestAuthInterceptor_InvalidFormat_NoBearer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validateToken := func(token string) (bool, error) {
		return true, nil
	}

	interceptor := AuthInterceptor(logger, validateToken)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"authorization": "token-without-bearer",
	}))
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
	assert.Contains(t, err.Error(), "invalid authorization format")
}

func TestAuthInterceptor_InvalidFormat_TooManyParts(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validateToken := func(token string) (bool, error) {
		return true, nil
	}

	interceptor := AuthInterceptor(logger, validateToken)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"authorization": "Bearer token extra",
	}))
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_InvalidFormat_EmptyBearer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validateToken := func(token string) (bool, error) {
		return true, nil
	}

	interceptor := AuthInterceptor(logger, validateToken)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"authorization": "bearer", // No token after bearer
	}))
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))
}

func TestAuthInterceptor_TokenValidationError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validateToken := func(token string) (bool, error) {
		return false, errors.New("validation error")
	}

	interceptor := AuthInterceptor(logger, validateToken)

	ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
		"authorization": "Bearer test-token",
	}))
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "success", nil
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "token validation failed")
}

func TestAuthInterceptor_CaseInsensitiveBearer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	validateToken := func(token string) (bool, error) {
		return token == "valid-token", nil
	}

	interceptor := AuthInterceptor(logger, validateToken)

	tests := []struct {
		name    string
		auth    string
		success bool
	}{
		{
			name:    "lowercase bearer",
			auth:    "bearer valid-token",
			success: true,
		},
		{
			name:    "uppercase bearer",
			auth:    "BEARER valid-token",
			success: true,
		},
		{
			name:    "mixed case bearer",
			auth:    "BeArEr valid-token",
			success: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := metadata.NewIncomingContext(context.Background(), metadata.New(map[string]string{
				"authorization": tt.auth,
			}))
			req := "test-request"
			info := &grpc.UnaryServerInfo{
				FullMethod: "/test.v1.TestService/TestMethod",
			}

			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return "success", nil
			}

			resp, err := interceptor(ctx, req, info, handler)
			if tt.success {
				assert.NoError(t, err)
				assert.Equal(t, "success", resp)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

