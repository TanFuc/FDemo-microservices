package grpc

import (
	"context"
	"fmt"
	"time"

	pb "microservices/catalog/internal/infrastructure/grpc/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// AuthClient wraps gRPC connection to Auth Service
type AuthClient struct {
	conn *grpc.ClientConn
	addr string
}

// NewAuthClient creates a new gRPC client to Auth Service
func NewAuthClient(addr string) (*AuthClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service: %w", err)
	}

	return &AuthClient{
		conn: conn,
		addr: addr,
	}, nil
}

// CheckPermission validates token and checks if user has permission
func (c *AuthClient) CheckPermission(ctx context.Context, token, resource, action string, contextData map[string]string) (*pb.CheckPermissionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &pb.CheckPermissionRequest{
		Token:    token,
		Resource: resource,
		Action:   action,
		Context:  contextData,
	}

	// Invoke gRPC method
	var resp pb.CheckPermissionResponse
	err := c.conn.Invoke(ctx, "/auth.AuthService/CheckPermission", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("grpc call failed: %w", err)
	}

	return &resp, nil
}

// GetUserInfo retrieves user information by ID
func (c *AuthClient) GetUserInfo(ctx context.Context, userID string) (*pb.GetUserInfoResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &pb.GetUserInfoRequest{
		UserId: userID,
	}

	var resp pb.GetUserInfoResponse
	err := c.conn.Invoke(ctx, "/auth.AuthService/GetUserInfo", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("grpc call failed: %w", err)
	}

	return &resp, nil
}

// ValidateToken validates JWT token and returns token info
func (c *AuthClient) ValidateToken(ctx context.Context, token string) (*pb.ValidateTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &pb.ValidateTokenRequest{
		Token: token,
	}

	var resp pb.ValidateTokenResponse
	err := c.conn.Invoke(ctx, "/auth.AuthService/ValidateToken", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("grpc call failed: %w", err)
	}

	return &resp, nil
}

// Authorize checks if user has permission (legacy method)
func (c *AuthClient) Authorize(ctx context.Context, userID, resource, action string) (*pb.AuthorizeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req := &pb.AuthorizeRequest{
		UserId:   userID,
		Resource: resource,
		Action:   action,
	}

	var resp pb.AuthorizeResponse
	err := c.conn.Invoke(ctx, "/auth.AuthService/Authorize", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("grpc call failed: %w", err)
	}

	return &resp, nil
}

// Close closes the gRPC connection
func (c *AuthClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
