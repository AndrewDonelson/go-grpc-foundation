package server

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	grpcRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "grpc_requests_total",
			Help: "Total number of gRPC requests",
		},
		[]string{"service", "method", "code"},
	)

	grpcRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "grpc_request_duration_seconds",
			Help:    "gRPC request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
		},
		[]string{"service", "method"},
	)

	grpcActiveRequests = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "grpc_active_requests",
			Help: "Number of active gRPC requests",
		},
		[]string{"service", "method"},
	)
)

// metricsInterceptor tracks request metrics
func metricsInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()

		// Extract service name from full method (e.g., "/service.v1.Service/Method" -> "service.v1.Service")
		serviceName := extractServiceName(info.FullMethod)

		// Track active requests
		grpcActiveRequests.WithLabelValues(serviceName, info.FullMethod).Inc()
		defer grpcActiveRequests.WithLabelValues(serviceName, info.FullMethod).Dec()

		// Call handler
		resp, err := handler(ctx, req)

		// Record metrics
		duration := time.Since(start).Seconds()
		st, _ := status.FromError(err)
		code := st.Code().String()

		grpcRequestsTotal.WithLabelValues(serviceName, info.FullMethod, code).Inc()
		grpcRequestDuration.WithLabelValues(serviceName, info.FullMethod).Observe(duration)

		return resp, err
	}
}

// extractServiceName extracts the service name from a full method path
// Example: "/hello.v1.HelloService/SayHello" -> "hello.v1.HelloService"
func extractServiceName(fullMethod string) string {
	// Remove leading "/"
	if len(fullMethod) > 0 && fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}

	// Find the last "/" to separate service from method
	lastSlash := -1
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash > 0 {
		return fullMethod[:lastSlash]
	}

	// Fallback: return the full method if we can't parse it
	return fullMethod
}

