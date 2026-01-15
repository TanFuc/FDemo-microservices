package authorization

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// InterceptorConfig represents the configuration for gRPC authorization interceptors
type InterceptorConfig struct {
	// Authorizer is the authorization instance
	Authorizer Authorizer

	// SubjectExtractor extracts the subject (user ID) from the context
	SubjectExtractor func(ctx context.Context) string

	// ResourceExtractor extracts the resource from the method name
	ResourceExtractor func(fullMethod string) string

	// ActionExtractor extracts the action from the method name
	ActionExtractor func(fullMethod string) string

	// DomainExtractor extracts the domain from the context
	DomainExtractor func(ctx context.Context) string

	// AttributesExtractor extracts attributes for ABAC from the context
	AttributesExtractor func(ctx context.Context) map[string]interface{}

	// Skipper defines methods to skip authorization
	Skipper func(fullMethod string) bool

	// UnauthorizedError is returned when the user is not authenticated
	UnauthorizedError error

	// ForbiddenError is returned when the user is not authorized
	ForbiddenError error

	// ErrorHandler handles authorization errors
	ErrorHandler func(ctx context.Context, err error) error

	// SuccessHandler is called after successful authorization
	SuccessHandler func(ctx context.Context, result *AuthResult)

	// EnableLogging enables logging of authorization decisions
	EnableLogging bool

	// LoggingFunc is called for each authorization decision when logging is enabled
	LoggingFunc func(ctx context.Context, fullMethod string, result *AuthResult, duration time.Duration)
}

// DefaultInterceptorConfig returns the default interceptor configuration
func DefaultInterceptorConfig() InterceptorConfig {
	return InterceptorConfig{
		SubjectExtractor:  defaultGRPCSubjectExtractor,
		ResourceExtractor: defaultGRPCResourceExtractor,
		ActionExtractor:   defaultGRPCActionExtractor,
		DomainExtractor:   defaultGRPCDomainExtractor,
		Skipper:           defaultGRPCSkipper,
		UnauthorizedError: status.Error(codes.Unauthenticated, "authentication required"),
		ForbiddenError:    status.Error(codes.PermissionDenied, "access denied"),
		ErrorHandler: func(ctx context.Context, err error) error {
			return status.Error(codes.Internal, "authorization check failed")
		},
		EnableLogging: false,
	}
}

// UnaryServerInterceptor creates a unary server interceptor for authorization
func UnaryServerInterceptor(auth Authorizer) grpc.UnaryServerInterceptor {
	config := DefaultInterceptorConfig()
	config.Authorizer = auth
	return UnaryServerInterceptorWithConfig(config)
}

// UnaryServerInterceptorWithConfig creates a unary server interceptor with custom config
func UnaryServerInterceptorWithConfig(config InterceptorConfig) grpc.UnaryServerInterceptor {
	applyDefaultInterceptorConfig(&config)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip if configured
		if config.Skipper(info.FullMethod) {
			return handler(ctx, req)
		}

		start := time.Now()

		// Check authorization
		result, err := checkAuthorization(ctx, info.FullMethod, config)
		if err != nil {
			return nil, err
		}

		// Log if enabled
		if config.EnableLogging && config.LoggingFunc != nil {
			config.LoggingFunc(ctx, info.FullMethod, result, time.Since(start))
		}

		// Store result in context
		ctx = context.WithValue(ctx, authResultContextKey, result)

		// Call success handler if configured
		if config.SuccessHandler != nil {
			config.SuccessHandler(ctx, result)
		}

		return handler(ctx, req)
	}
}

// StreamServerInterceptor creates a stream server interceptor for authorization
func StreamServerInterceptor(auth Authorizer) grpc.StreamServerInterceptor {
	config := DefaultInterceptorConfig()
	config.Authorizer = auth
	return StreamServerInterceptorWithConfig(config)
}

// StreamServerInterceptorWithConfig creates a stream server interceptor with custom config
func StreamServerInterceptorWithConfig(config InterceptorConfig) grpc.StreamServerInterceptor {
	applyDefaultInterceptorConfig(&config)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip if configured
		if config.Skipper(info.FullMethod) {
			return handler(srv, ss)
		}

		start := time.Now()
		ctx := ss.Context()

		// Check authorization
		result, err := checkAuthorization(ctx, info.FullMethod, config)
		if err != nil {
			return err
		}

		// Log if enabled
		if config.EnableLogging && config.LoggingFunc != nil {
			config.LoggingFunc(ctx, info.FullMethod, result, time.Since(start))
		}

		// Call success handler if configured
		if config.SuccessHandler != nil {
			config.SuccessHandler(ctx, result)
		}

		// Wrap the stream with a new context containing the result
		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ctx, authResultContextKey, result),
		}

		return handler(srv, wrapped)
	}
}

// UnaryClientInterceptor creates a unary client interceptor that adds authorization headers
func UnaryClientInterceptor(subjectID, domain string) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// Add authorization metadata
		md := metadata.Pairs(
			"x-user-id", subjectID,
			"x-domain", domain,
		)
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor creates a stream client interceptor that adds authorization headers
func StreamClientInterceptor(subjectID, domain string) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// Add authorization metadata
		md := metadata.Pairs(
			"x-user-id", subjectID,
			"x-domain", domain,
		)
		ctx = metadata.NewOutgoingContext(ctx, md)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// Context key for storing auth result
type contextKey string

const authResultContextKey contextKey = "authz_result"

// GetAuthResultFromContext retrieves the authorization result from a gRPC context
func GetAuthResultFromContext(ctx context.Context) *AuthResult {
	result := ctx.Value(authResultContextKey)
	if result == nil {
		return nil
	}
	return result.(*AuthResult)
}

// wrappedServerStream wraps a grpc.ServerStream with a custom context
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// Helper functions

func applyDefaultInterceptorConfig(config *InterceptorConfig) {
	if config.SubjectExtractor == nil {
		config.SubjectExtractor = defaultGRPCSubjectExtractor
	}
	if config.ResourceExtractor == nil {
		config.ResourceExtractor = defaultGRPCResourceExtractor
	}
	if config.ActionExtractor == nil {
		config.ActionExtractor = defaultGRPCActionExtractor
	}
	if config.DomainExtractor == nil {
		config.DomainExtractor = defaultGRPCDomainExtractor
	}
	if config.Skipper == nil {
		config.Skipper = defaultGRPCSkipper
	}
	if config.UnauthorizedError == nil {
		config.UnauthorizedError = status.Error(codes.Unauthenticated, "authentication required")
	}
	if config.ForbiddenError == nil {
		config.ForbiddenError = status.Error(codes.PermissionDenied, "access denied")
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = func(ctx context.Context, err error) error {
			return status.Error(codes.Internal, "authorization check failed")
		}
	}
}

func checkAuthorization(ctx context.Context, fullMethod string, config InterceptorConfig) (*AuthResult, error) {
	// Extract subject (user ID)
	subject := config.SubjectExtractor(ctx)
	if subject == "" {
		return nil, config.UnauthorizedError
	}

	// Extract resource, action, and domain
	resource := config.ResourceExtractor(fullMethod)
	action := config.ActionExtractor(fullMethod)
	domain := config.DomainExtractor(ctx)

	// Build auth request
	request := &AuthRequest{
		Subject: subject,
		Object:  resource,
		Action:  action,
		Domain:  domain,
	}

	// Extract attributes if configured
	if config.AttributesExtractor != nil {
		request.Attributes = config.AttributesExtractor(ctx)
	}

	// Perform authorization check
	result, err := config.Authorizer.Enforce(ctx, request)
	if err != nil {
		return nil, config.ErrorHandler(ctx, err)
	}

	// Check authorization result
	if !result.Allowed {
		return nil, config.ForbiddenError
	}

	return result, nil
}

// Default extractors for gRPC

func defaultGRPCSubjectExtractor(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	// Try x-user-id header
	if values := md.Get("x-user-id"); len(values) > 0 {
		return values[0]
	}

	// Try authorization header (Bearer token would need to be decoded)
	if values := md.Get("authorization"); len(values) > 0 {
		// This is a simplified version - in production, you'd decode the JWT
		return values[0]
	}

	return ""
}

func defaultGRPCResourceExtractor(fullMethod string) string {
	// fullMethod is in format /package.Service/Method
	// Extract the service name as the resource
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 2 {
		service := parts[1]
		// Extract just the service name without package
		serviceParts := strings.Split(service, ".")
		if len(serviceParts) > 0 {
			return strings.ToLower(serviceParts[len(serviceParts)-1])
		}
		return strings.ToLower(service)
	}
	return strings.ToLower(fullMethod)
}

func defaultGRPCActionExtractor(fullMethod string) string {
	// fullMethod is in format /package.Service/Method
	// Extract the method name as the action
	parts := strings.Split(fullMethod, "/")
	if len(parts) >= 3 {
		method := parts[2]
		// Convert common method names to CRUD actions
		methodLower := strings.ToLower(method)
		switch {
		case strings.HasPrefix(methodLower, "get") || strings.HasPrefix(methodLower, "list") || strings.HasPrefix(methodLower, "find"):
			return "read"
		case strings.HasPrefix(methodLower, "create") || strings.HasPrefix(methodLower, "add"):
			return "create"
		case strings.HasPrefix(methodLower, "update") || strings.HasPrefix(methodLower, "set"):
			return "update"
		case strings.HasPrefix(methodLower, "delete") || strings.HasPrefix(methodLower, "remove"):
			return "delete"
		default:
			return methodLower
		}
	}
	return "execute"
}

func defaultGRPCDomainExtractor(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "default"
	}

	// Try x-domain header
	if values := md.Get("x-domain"); len(values) > 0 {
		return values[0]
	}

	// Try x-tenant-id header
	if values := md.Get("x-tenant-id"); len(values) > 0 {
		return values[0]
	}

	return "default"
}

func defaultGRPCSkipper(fullMethod string) bool {
	// Skip health check and reflection methods
	return strings.Contains(fullMethod, "grpc.health") ||
		strings.Contains(fullMethod, "grpc.reflection") ||
		strings.Contains(fullMethod, "Health")
}

// RequireRoleInterceptor creates an interceptor that requires a specific role
func RequireRoleInterceptor(auth Authorizer, roleID string, domain ...string) grpc.UnaryServerInterceptor {
	dom := "default"
	if len(domain) > 0 {
		dom = domain[0]
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		subject := defaultGRPCSubjectExtractor(ctx)
		if subject == "" {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}

		hasRole, err := HasRole(ctx, auth, subject, roleID, dom)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to check role")
		}

		if !hasRole {
			return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("role '%s' required", roleID))
		}

		return handler(ctx, req)
	}
}

// RequirePermissionInterceptor creates an interceptor that requires a specific permission
func RequirePermissionInterceptor(auth Authorizer, resource, action string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		subject := defaultGRPCSubjectExtractor(ctx)
		if subject == "" {
			return nil, status.Error(codes.Unauthenticated, "authentication required")
		}

		allowed, err := CanExecute(ctx, auth, subject, resource, action)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to check permission")
		}

		if !allowed {
			return nil, status.Error(codes.PermissionDenied, fmt.Sprintf("permission '%s:%s' required", resource, action))
		}

		return handler(ctx, req)
	}
}
