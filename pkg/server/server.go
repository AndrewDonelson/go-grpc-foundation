package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/AndrewDonelson/go-grpc-foundation/pkg/logging"
	"github.com/AndrewDonelson/go-grpc-foundation/pkg/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server represents a gRPC service with common functionality
type Server struct {
	config         *Config
	logger         *zap.Logger
	grpcServer     *grpc.Server
	healthServer   *health.Server
	metricsServer  *http.Server
	shutdownFuncs  []func(context.Context) error
	mu             sync.Mutex
}

// Config holds server configuration
type Config struct {
	ServiceName      string
	Host             string
	Port             int
	MetricsPort      int
	Environment      string // "development" or "production"
	ShutdownTimeout  time.Duration
	EnableReflection bool
	EnableMetrics    bool
}

// New creates a new server instance
func New(config *Config) (*Server, error) {
	// Setup logger based on environment
	logger, err := logging.NewLogger(config.Environment)
	if err != nil {
		return nil, fmt.Errorf("failed to setup logger: %w", err)
	}

	logger.Info("initializing server",
		zap.String("service", config.ServiceName),
		zap.String("environment", config.Environment),
		zap.String("host", config.Host),
		zap.Int("port", config.Port),
	)

	// Create health server
	healthServer := health.NewServer()

	// Setup gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			logging.UnaryLoggingInterceptor(logger),
			middleware.RecoveryInterceptor(logger),
			metricsInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			logging.StreamLoggingInterceptor(logger),
			middleware.StreamRecoveryInterceptor(logger),
		),
	)

	// Register health service
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

	// Enable reflection if requested (useful for development)
	if config.EnableReflection {
		reflection.Register(grpcServer)
		logger.Info("gRPC reflection enabled")
	}

	server := &Server{
		config:        config,
		logger:        logger,
		grpcServer:    grpcServer,
		healthServer:  healthServer,
		shutdownFuncs: make([]func(context.Context) error, 0),
	}

	// Setup metrics server if enabled
	if config.EnableMetrics {
		server.setupMetricsServer()
	}

	return server, nil
}

// RegisterService registers a gRPC service
func (s *Server) RegisterService(serviceDesc *grpc.ServiceDesc, impl interface{}) {
	s.grpcServer.RegisterService(serviceDesc, impl)
	s.logger.Info("registered gRPC service", zap.String("service", serviceDesc.ServiceName))
}

// AddShutdownFunc registers a cleanup function to be called on shutdown
func (s *Server) AddShutdownFunc(fn func(context.Context) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shutdownFuncs = append(s.shutdownFuncs, fn)
}

// Start starts the server and blocks until shutdown
func (s *Server) Start() error {
	// Create listener
	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.logger.Info("server listening", zap.String("address", address))

	// Mark service as serving
	s.healthServer.SetServingStatus(s.config.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)

	// Start metrics server if enabled
	if s.metricsServer != nil {
		go s.startMetricsServer()
	}

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		s.logger.Info("shutdown signal received")
	case err := <-errChan:
		return err
	}

	// Perform graceful shutdown
	return s.Shutdown()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.logger.Info("shutting down server")

	// Mark service as not serving
	s.healthServer.SetServingStatus(s.config.ServiceName, grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	// Create shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	// Stop accepting new connections
	s.grpcServer.GracefulStop()

	// Shutdown metrics server
	if s.metricsServer != nil {
		if err := s.metricsServer.Shutdown(ctx); err != nil {
			s.logger.Error("failed to shutdown metrics server", zap.Error(err))
		}
	}

	// Execute shutdown functions
	s.mu.Lock()
	shutdownFuncs := s.shutdownFuncs
	s.mu.Unlock()

	for _, fn := range shutdownFuncs {
		if err := fn(ctx); err != nil {
			s.logger.Error("shutdown function error", zap.Error(err))
		}
	}

	// Sync logger
	_ = s.logger.Sync()

	s.logger.Info("server shutdown complete")
	return nil
}

// GetLogger returns the server's logger
func (s *Server) GetLogger() *zap.Logger {
	return s.logger
}

// GetGRPCServer returns the underlying gRPC server
func (s *Server) GetGRPCServer() *grpc.Server {
	return s.grpcServer
}

// setupMetricsServer configures the Prometheus metrics HTTP server
func (s *Server) setupMetricsServer() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	s.metricsServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.config.MetricsPort),
		Handler: mux,
	}

	s.logger.Info("metrics server configured", zap.Int("port", s.config.MetricsPort))
}

// startMetricsServer starts the metrics HTTP server
func (s *Server) startMetricsServer() {
	s.logger.Info("starting metrics server")
	if err := s.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("metrics server error", zap.Error(err))
	}
}

// DefaultConfig returns default server configuration
func DefaultConfig(serviceName string) *Config {
	return &Config{
		ServiceName:      serviceName,
		Host:             "0.0.0.0",
		Port:             50051,
		MetricsPort:      9090,
		Environment:      getEnv("ENVIRONMENT", "production"),
		ShutdownTimeout:  30 * time.Second,
		EnableReflection: getEnv("ENVIRONMENT", "production") == "development",
		EnableMetrics:    true,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

