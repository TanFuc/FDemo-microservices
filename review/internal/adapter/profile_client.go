package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ProfileClient interface for interacting with Profile Service
type ProfileClient interface {
	GetUserInfo(ctx context.Context, userID string) (*UserInfo, error)
}

// UserInfo contains user display information
type UserInfo struct {
	DisplayName string `json:"displayName"`
	AvatarURL   string `json:"avatarUrl"`
	Email       string `json:"email,omitempty"`
}

// HTTPProfileClient implements ProfileClient using HTTP
type HTTPProfileClient struct {
	baseURL    string
	httpClient *http.Client
	serviceKey string
}

// ProfileClientConfig holds configuration for profile client
type ProfileClientConfig struct {
	BaseURL    string
	Timeout    time.Duration
	ServiceKey string
}

// NewHTTPProfileClient creates a new HTTP profile client
func NewHTTPProfileClient(cfg ProfileClientConfig) *HTTPProfileClient {
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}

	return &HTTPProfileClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		serviceKey: cfg.ServiceKey,
	}
}

// GetUserInfo fetches user display info from Profile Service
func (c *HTTPProfileClient) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	url := fmt.Sprintf("%s/internal/users/%s", c.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add internal service authentication header
	if c.serviceKey != "" {
		req.Header.Set("X-Internal-Service-Key", c.serviceKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call profile service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Return default user info if profile not found
		return &UserInfo{
			DisplayName: "User-" + userID[:8],
			AvatarURL:   "",
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("profile service returned status %d", resp.StatusCode)
	}

	var info UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &info, nil
}

// Ensure HTTPProfileClient implements ProfileClient
var _ ProfileClient = (*HTTPProfileClient)(nil)

// MockProfileClient is a mock implementation for development
type MockProfileClient struct{}

// GetUserInfo returns mock user info
func (c *MockProfileClient) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	shortID := userID
	if len(userID) > 8 {
		shortID = userID[:8]
	}
	return &UserInfo{
		DisplayName: "User-" + shortID,
		AvatarURL:   "",
	}, nil
}

// Ensure MockProfileClient implements ProfileClient
var _ ProfileClient = (*MockProfileClient)(nil)
