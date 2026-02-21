package grpc

import (
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"microservices/auth/internal/config"
	"microservices/auth/pkg/logger"
	pb "microservices/auth/pkg/pb/v1"
)

// Server represents the gRPC server
type Server struct {
	cfg        *config.Config
	grpcServer *grpc.Server
	handler    *AuthHandler
}

// NewServer creates a new gRPC server
func NewServer(cfg *config.Config, authHandler *AuthHandler) *Server {
	grpcServer := grpc.NewServer()

	// Register the AuthService
	pb.RegisterAuthServiceServer(grpcServer, authHandler)

	// Enable reflection for debugging tools like grpcurl
	reflection.Register(grpcServer)

	return &Server{
		cfg:        cfg,
		grpcServer: grpcServer,
		handler:    authHandler,
	}
}

// Run starts the gRPC server
func (s *Server) Run() error {
	addr := ":" + s.cfg.App.GRPCPort
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	logger.Info().
		Str("port", s.cfg.App.GRPCPort).
		Msg("gRPC server started")

	return s.grpcServer.Serve(listener)
}

// GracefulStop gracefully stops the gRPC server
func (s *Server) GracefulStop() {
	logger.Info().Msg("Stopping gRPC server...")
	s.grpcServer.GracefulStop()
	logger.Info().Msg("gRPC server stopped")
}

// Stop immediately stops the gRPC server
func (s *Server) Stop() {
	s.grpcServer.Stop()
}
