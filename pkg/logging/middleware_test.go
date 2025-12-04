package logging

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestUnaryLoggingInterceptor_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := UnaryLoggingInterceptor(logger)

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

func TestUnaryLoggingInterceptor_Error(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := UnaryLoggingInterceptor(logger)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	testError := status.Errorf(codes.Internal, "test error")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, testError
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestUnaryLoggingInterceptor_Duration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := UnaryLoggingInterceptor(logger)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/SlowMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		time.Sleep(10 * time.Millisecond)
		return "done", nil
	}

	start := time.Now()
	resp, err := interceptor(ctx, req, info, handler)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, "done", resp)
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
}

func TestUnaryLoggingInterceptor_ErrorCodes(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := UnaryLoggingInterceptor(logger)

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
			handler := func(ctx context.Context, req interface{}) (interface{}, error) {
				return nil, status.Errorf(code, "test error")
			}

			resp, err := interceptor(ctx, req, info, handler)
			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, code, status.Code(err))
		})
	}
}

func TestUnaryLoggingInterceptor_NonStatusError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := UnaryLoggingInterceptor(logger)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, errors.New("non-status error")
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestStreamLoggingInterceptor_Success(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := StreamLoggingInterceptor(logger)

	srv := struct{}{}
	ss := &mockServerStream{}
	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.v1.TestService/TestStream",
		IsClientStream: false,
		IsServerStream: true,
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := interceptor(srv, ss, info, handler)
	assert.NoError(t, err)
}

func TestStreamLoggingInterceptor_Error(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := StreamLoggingInterceptor(logger)

	srv := struct{}{}
	ss := &mockServerStream{}
	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.v1.TestService/TestStream",
		IsClientStream: true,
		IsServerStream: true,
	}

	testError := status.Errorf(codes.Internal, "stream error")
	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return testError
	}

	err := interceptor(srv, ss, info, handler)
	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestStreamLoggingInterceptor_StreamTypes(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := StreamLoggingInterceptor(logger)

	srv := struct{}{}
	ss := &mockServerStream{}

	tests := []struct {
		name           string
		isClientStream bool
		isServerStream bool
	}{
		{
			name:           "unary stream",
			isClientStream: false,
			isServerStream: false,
		},
		{
			name:           "client stream",
			isClientStream: true,
			isServerStream: false,
		},
		{
			name:           "server stream",
			isClientStream: false,
			isServerStream: true,
		},
		{
			name:           "bidirectional stream",
			isClientStream: true,
			isServerStream: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &grpc.StreamServerInfo{
				FullMethod:     "/test.v1.TestService/TestStream",
				IsClientStream: tt.isClientStream,
				IsServerStream: tt.isServerStream,
			}

			handler := func(srv interface{}, stream grpc.ServerStream) error {
				return nil
			}

			err := interceptor(srv, ss, info, handler)
			assert.NoError(t, err)
		})
	}
}

func TestStreamLoggingInterceptor_Duration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := StreamLoggingInterceptor(logger)

	srv := struct{}{}
	ss := &mockServerStream{}
	info := &grpc.StreamServerInfo{
		FullMethod:     "/test.v1.TestService/SlowStream",
		IsClientStream: false,
		IsServerStream: true,
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}

	start := time.Now()
	err := interceptor(srv, ss, info, handler)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 10*time.Millisecond)
}

// mockServerStream is a minimal implementation of grpc.ServerStream for testing
type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	if m.ctx == nil {
		return context.Background()
	}
	return m.ctx
}

func (m *mockServerStream) SendMsg(msg interface{}) error {
	return nil
}

func (m *mockServerStream) RecvMsg(msg interface{}) error {
	return nil
}

