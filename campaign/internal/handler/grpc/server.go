package grpc

import (
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"microservices/campaign/internal/config"
	"microservices/campaign/pkg/logger"
)

// Server represents the gRPC server.
type Server struct {
	cfg        *config.Config
	grpcServer *grpc.Server
	handler    *CampaignHandler
}

// NewServer creates a new gRPC server.
func NewServer(cfg *config.Config, handler *CampaignHandler) *Server {
	grpcServer := grpc.NewServer()

	// Register campaign service handler
	// NOTE: You need to generate pb files from proto and register here
	// pb.RegisterCampaignServiceServer(grpcServer, handler)

	// Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	return &Server{
		cfg:        cfg,
		grpcServer: grpcServer,
		handler:    handler,
	}
}

// Run starts the gRPC server.
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

// GracefulStop gracefully stops the gRPC server.
func (s *Server) GracefulStop() {
	logger.Info().Msg("Stopping gRPC server...")
	s.grpcServer.GracefulStop()
	logger.Info().Msg("gRPC server stopped")
}

// Stop immediately stops the gRPC server.
func (s *Server) Stop() {
	s.grpcServer.Stop()
}
