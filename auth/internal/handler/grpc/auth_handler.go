package grpc

import (
	"context"

	"github.com/google/uuid"

	"microservices/auth/internal/service"
	"microservices/auth/pkg/logger"
	pb "microservices/auth/pkg/pb/v1"
)

// AuthHandler implements the gRPC AuthServiceServer interface
type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	tokenService *service.TokenService
	userService  *service.UserService
}

// NewAuthHandler creates a new gRPC auth handler
func NewAuthHandler(tokenService *service.TokenService, userService *service.UserService) *AuthHandler {
	return &AuthHandler{
		tokenService: tokenService,
		userService:  userService,
	}
}

// VerifyToken validates an access token and returns user information
func (h *AuthHandler) VerifyToken(ctx context.Context, req *pb.VerifyTokenRequest) (*pb.VerifyTokenResponse, error) {
	if req.Token == "" {
		return &pb.VerifyTokenResponse{
			Valid: false,
		}, nil
	}

	// Verify the access token
	claims, err := h.tokenService.VerifyAccessToken(ctx, req.Token)
	if err != nil {
		logger.Debug().Err(err).Msg("Token verification failed")
		return &pb.VerifyTokenResponse{
			Valid: false,
		}, nil
	}

	// Parse user ID from claims
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		logger.Debug().Err(err).Msg("Failed to parse user ID from token claims")
		return &pb.VerifyTokenResponse{
			Valid: false,
		}, nil
	}

	// Get user roles
	roles, err := h.userService.GetUserRoles(ctx, userID)
	if err != nil {
		logger.Warn().Err(err).Str("userId", userID.String()).Msg("Failed to get user roles")
		roles = []string{}
	}

	// Get primary role (first role or empty)
	primaryRole := ""
	if len(roles) > 0 {
		primaryRole = roles[0]
	}

	// Get user permissions
	permissions, err := h.userService.GetUserPermissionsCached(ctx, userID)
	if err != nil {
		logger.Warn().Err(err).Str("userId", userID.String()).Msg("Failed to get user permissions")
		permissions = []string{}
	}

	return &pb.VerifyTokenResponse{
		Valid:       true,
		UserId:      userID.String(),
		Email:       claims.Email,
		Role:        primaryRole,
		Permissions: permissions,
	}, nil
}

// CheckPermission checks if a user has a specific permission
func (h *AuthHandler) CheckPermission(ctx context.Context, req *pb.CheckPermissionRequest) (*pb.CheckPermissionResponse, error) {
	if req.UserId == "" || req.Permission == "" {
		return &pb.CheckPermissionResponse{
			Allowed: false,
		}, nil
	}

	// Parse user ID
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		logger.Debug().Err(err).Str("userId", req.UserId).Msg("Invalid user ID format")
		return &pb.CheckPermissionResponse{
			Allowed: false,
		}, nil
	}

	// Check if user has the permission
	allowed, err := h.userService.HasPermission(ctx, userID, req.Permission)
	if err != nil {
		logger.Warn().Err(err).
			Str("userId", req.UserId).
			Str("permission", req.Permission).
			Msg("Failed to check user permission")
		return &pb.CheckPermissionResponse{
			Allowed: false,
		}, nil
	}

	return &pb.CheckPermissionResponse{
		Allowed: allowed,
	}, nil
}
