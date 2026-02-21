package grpc

import (
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"microservices/template-service/internal/config"
	"microservices/template-service/pkg/logger"
	"microservices/template-service/pkg/pb"
)

type Server struct {
	cfg        *config.Config
	grpcServer *grpc.Server
	handler    *ItemHandler
}

func NewServer(cfg *config.Config, handler *ItemHandler) *Server {
	grpcServer := grpc.NewServer()

	pb.RegisterTemplateServiceServer(grpcServer, handler)

	// Enable reflection for tools like grpcurl
	reflection.Register(grpcServer)

	return &Server{
		cfg:        cfg,
		grpcServer: grpcServer,
		handler:    handler,
	}
}

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

func (s *Server) GracefulStop() {
	logger.Info().Msg("Stopping gRPC server...")
	s.grpcServer.GracefulStop()
	logger.Info().Msg("gRPC server stopped")
}

func (s *Server) Stop() {
	s.grpcServer.Stop()
}
