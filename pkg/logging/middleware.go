package logging

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// UnaryLoggingInterceptor logs unary RPC calls
func UnaryLoggingInterceptor(logger *zap.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Get status
		st, _ := status.FromError(err)

		// Log based on result
		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("code", st.Code().String()),
		}

		if err != nil {
			logger.Error("rpc call failed", append(fields, zap.Error(err))...)
		} else {
			logger.Debug("rpc call completed", fields...)
		}

		return resp, err
	}
}

// StreamLoggingInterceptor logs streaming RPC calls
func StreamLoggingInterceptor(logger *zap.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()

		// Call handler
		err := handler(srv, ss)

		// Calculate duration
		duration := time.Since(start)

		// Get status
		st, _ := status.FromError(err)

		// Log
		fields := []zap.Field{
			zap.String("method", info.FullMethod),
			zap.Duration("duration", duration),
			zap.String("code", st.Code().String()),
			zap.Bool("is_client_stream", info.IsClientStream),
			zap.Bool("is_server_stream", info.IsServerStream),
		}

		if err != nil {
			logger.Error("stream call failed", append(fields, zap.Error(err))...)
		} else {
			logger.Debug("stream call completed", fields...)
		}

		return err
	}
}

