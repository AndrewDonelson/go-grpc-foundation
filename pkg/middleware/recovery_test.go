package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoveryInterceptor_NoPanic(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := RecoveryInterceptor(logger)

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

func TestRecoveryInterceptor_PanicString(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := RecoveryInterceptor(logger)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic")
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "internal server error")
}

func TestRecoveryInterceptor_PanicError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := RecoveryInterceptor(logger)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	testError := errors.New("test error")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic(testError)
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRecoveryInterceptor_PanicInt(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := RecoveryInterceptor(logger)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic(42)
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRecoveryInterceptor_PanicNil(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := RecoveryInterceptor(logger)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic(nil)
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Internal, status.Code(err))
}

func TestRecoveryInterceptor_HandlerError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := RecoveryInterceptor(logger)

	ctx := context.Background()
	req := "test-request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.v1.TestService/TestMethod",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, status.Errorf(codes.NotFound, "not found")
	}

	resp, err := interceptor(ctx, req, info, handler)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestStreamRecoveryInterceptor_NoPanic(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := StreamRecoveryInterceptor(logger)

	srv := struct{}{}
	ss := &mockServerStream{}
	info := &grpc.StreamServerInfo{
		FullMethod: "/test.v1.TestService/TestStream",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return nil
	}

	err := interceptor(srv, ss, info, handler)
	assert.NoError(t, err)
}

func TestStreamRecoveryInterceptor_Panic(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := StreamRecoveryInterceptor(logger)

	srv := struct{}{}
	ss := &mockServerStream{}
	info := &grpc.StreamServerInfo{
		FullMethod: "/test.v1.TestService/TestStream",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		panic("stream panic")
	}

	err := interceptor(srv, ss, info, handler)
	assert.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Contains(t, err.Error(), "internal server error")
}

func TestStreamRecoveryInterceptor_HandlerError(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := StreamRecoveryInterceptor(logger)

	srv := struct{}{}
	ss := &mockServerStream{}
	info := &grpc.StreamServerInfo{
		FullMethod: "/test.v1.TestService/TestStream",
	}

	handler := func(srv interface{}, stream grpc.ServerStream) error {
		return status.Errorf(codes.Unavailable, "service unavailable")
	}

	err := interceptor(srv, ss, info, handler)
	assert.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

// mockServerStream for testing
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

