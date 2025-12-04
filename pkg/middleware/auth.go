package middleware

import (
	"context"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthInterceptor validates JWT tokens from Authorization header
// This is a basic implementation - customize based on your auth needs
func AuthInterceptor(logger *zap.Logger, validateToken func(token string) (bool, error)) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Skip auth for health checks
		if strings.HasPrefix(info.FullMethod, "/grpc.health") {
			return handler(ctx, req)
		}

		// Extract token from metadata
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			logger.Warn("no metadata found in context", zap.String("method", info.FullMethod))
			return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
		}

		// Get authorization header
		authHeaders := md.Get("authorization")
		if len(authHeaders) == 0 {
			logger.Warn("no authorization header", zap.String("method", info.FullMethod))
			return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
		}

		// Extract token (expecting "Bearer <token>")
		authHeader := authHeaders[0]
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			logger.Warn("invalid authorization format", zap.String("method", info.FullMethod))
			return nil, status.Errorf(codes.Unauthenticated, "invalid authorization format")
		}

		token := parts[1]

		// Validate token
		valid, err := validateToken(token)
		if err != nil {
			logger.Error("token validation error", zap.Error(err), zap.String("method", info.FullMethod))
			return nil, status.Errorf(codes.Internal, "token validation failed")
		}

		if !valid {
			logger.Warn("invalid token", zap.String("method", info.FullMethod))
			return nil, status.Errorf(codes.Unauthenticated, "invalid token")
		}

		// Token is valid, proceed with handler
		return handler(ctx, req)
	}
}

