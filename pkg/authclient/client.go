package authclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client is the gRPC client for Auth Service
type Client struct {
	conn    *grpc.ClientConn
	addr    string
	timeout time.Duration
}

// Config holds the configuration for the auth client
type Config struct {
	// GRPCAddr is the address of the Auth Service gRPC server (e.g., "localhost:50051")
	GRPCAddr string
	// Timeout is the timeout for gRPC calls (default: 3s)
	Timeout time.Duration
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		GRPCAddr: "localhost:50051",
		Timeout:  3 * time.Second,
	}
}

// NewClient creates a new Auth gRPC client
func NewClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 3 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth service at %s: %w", cfg.GRPCAddr, err)
	}

	return &Client{
		conn:    conn,
		addr:    cfg.GRPCAddr,
		timeout: cfg.Timeout,
	}, nil
}

// CheckPermission validates token and checks if user has permission for resource:action
func (c *Client) CheckPermission(ctx context.Context, token, resource, action string, contextData map[string]string) (*CheckPermissionResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &CheckPermissionRequest{
		Token:    token,
		Resource: resource,
		Action:   action,
		Context:  contextData,
	}

	var resp CheckPermissionResponse
	err := c.conn.Invoke(ctx, "/auth.v1.AuthService/CheckPermission", req, &resp)
	if err != nil {
		if err2 := c.conn.Invoke(ctx, "/auth.AuthService/CheckPermission", req, &resp); err2 == nil {
			return &resp, nil
		}
		return nil, fmt.Errorf("grpc CheckPermission failed: %w", err)
	}

	return &resp, nil
}

// ValidateToken validates a JWT token and returns user info
func (c *Client) ValidateToken(ctx context.Context, token string) (*ValidateTokenResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &ValidateTokenRequest{Token: token}

	var resp ValidateTokenResponse
	err := c.conn.Invoke(ctx, "/auth.v1.AuthService/VerifyToken", req, &resp)
	if err != nil {
		if err2 := c.conn.Invoke(ctx, "/auth.AuthService/ValidateToken", req, &resp); err2 == nil {
			return &resp, nil
		}
		return nil, fmt.Errorf("grpc ValidateToken failed: %w", err)
	}

	return &resp, nil
}

// GetUserInfo retrieves user information by user ID
func (c *Client) GetUserInfo(ctx context.Context, userID string) (*GetUserInfoResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &GetUserInfoRequest{UserId: userID}

	var resp GetUserInfoResponse
	err := c.conn.Invoke(ctx, "/auth.AuthService/GetUserInfo", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("grpc GetUserInfo failed: %w", err)
	}

	return &resp, nil
}

// Authorize checks if user has permission (legacy method)
func (c *Client) Authorize(ctx context.Context, userID, resource, action string) (*AuthorizeResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	req := &AuthorizeRequest{
		UserId:   userID,
		Resource: resource,
		Action:   action,
	}

	var resp AuthorizeResponse
	err := c.conn.Invoke(ctx, "/auth.AuthService/Authorize", req, &resp)
	if err != nil {
		return nil, fmt.Errorf("grpc Authorize failed: %w", err)
	}

	return &resp, nil
}

// Close closes the gRPC connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// IsConnected checks if the client is connected
func (c *Client) IsConnected() bool {
	return c.conn != nil
}

// Address returns the address of the Auth Service
func (c *Client) Address() string {
	return c.addr
}
