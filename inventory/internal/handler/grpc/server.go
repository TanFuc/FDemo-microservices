package grpc

import (
	"net"

	"microservices/inventory/pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	server   *grpc.Server
	listener net.Listener
}

func NewServer(port string) (*Server, error) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	server := grpc.NewServer()
	reflection.Register(server)

	return &Server{
		server:   server,
		listener: listener,
	}, nil
}

func (s *Server) GetGRPCServer() *grpc.Server {
	return s.server
}

func (s *Server) Start() error {
	logger.Infof("Starting gRPC server on %s", s.listener.Addr().String())
	return s.server.Serve(s.listener)
}

func (s *Server) Stop() {
	logger.Info("Stopping gRPC server")
	s.server.GracefulStop()
}
