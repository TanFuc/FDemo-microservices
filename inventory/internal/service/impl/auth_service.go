package impl

import (
	"context"
	"time"

	"microservices/inventory/internal/service"

	authpb "microservices/auth/pkg/pb/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ service.AuthService = (*authService)(nil)

type authService struct {
	client  authpb.AuthServiceClient
	timeout time.Duration
}

func NewAuthService(addr string, timeout time.Duration) (service.AuthService, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &authService{
		client:  authpb.NewAuthServiceClient(conn),
		timeout: timeout,
	}, nil
}

func (s *authService) VerifyToken(ctx context.Context, token string) (*service.AuthUser, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
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
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
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
