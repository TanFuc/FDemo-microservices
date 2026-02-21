package impl

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "microservices/auth/pkg/pb/v1"
	"microservices/catalog/internal/service"
)

type authService struct {
	client authpb.AuthServiceClient
	conn   *grpc.ClientConn
}

// NewAuthService creates a new auth service client
func NewAuthService(grpcAddr string) (service.AuthService, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, err
	}

	client := authpb.NewAuthServiceClient(conn)

	return &authService{
		client: client,
		conn:   conn,
	}, nil
}

func (s *authService) VerifyToken(ctx context.Context, token string) (*service.AuthUser, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := s.client.VerifyToken(ctx, &authpb.VerifyTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, err
	}

	if !resp.Valid {
		return nil, nil
	}

	return &service.AuthUser{
		UserID:      resp.UserId,
		Email:       resp.Email,
		Role:        resp.Role,
		Permissions: resp.Permissions,
	}, nil
}

func (s *authService) CheckPermission(ctx context.Context, userID string, permission string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := s.client.CheckPermission(ctx, &authpb.CheckPermissionRequest{
		UserId:     userID,
		Permission: permission,
	})
	if err != nil {
		return false, err
	}

	return resp.Allowed, nil
}

// Close closes the gRPC connection
func (s *authService) Close() error {
	if s.conn != nil {
		return s.conn.Close()
	}
	return nil
}
