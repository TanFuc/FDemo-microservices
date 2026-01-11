package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s - %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
	}
}

func Wrap(err error, code, message string, statusCode int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		StatusCode: statusCode,
		Err:        err,
	}
}

// Common error codes
const (
	CodeSuccess      = "SUCCESS"
	CodeCreated      = "CREATED"
	CodeBadRequest   = "BAD_REQUEST"
	CodeUnauthorized = "UNAUTHORIZED"
	CodeForbidden    = "FORBIDDEN"
	CodeNotFound     = "NOT_FOUND"
	CodeConflict     = "CONFLICT"
	CodeInternal     = "INTERNAL_ERROR"
	CodeValidation   = "VALIDATION_FAILED"
)

// Authentication errors
var (
	ErrInvalidCredentials = &AppError{
		Code:       "INVALID_CREDENTIALS",
		Message:    "Invalid email or password",
		StatusCode: http.StatusUnauthorized,
	}
	ErrUserNotFound = &AppError{
		Code:       "USER_NOT_FOUND",
		Message:    "User not found",
		StatusCode: http.StatusUnauthorized,
	}
	ErrUserInactive = &AppError{
		Code:       "USER_INACTIVE",
		Message:    "User account is inactive",
		StatusCode: http.StatusUnauthorized,
	}
	ErrUserSuspended = &AppError{
		Code:       "USER_SUSPENDED",
		Message:    "User account is suspended",
		StatusCode: http.StatusUnauthorized,
	}
	ErrUserBanned = &AppError{
		Code:       "USER_BANNED",
		Message:    "User account is banned",
		StatusCode: http.StatusUnauthorized,
	}
	ErrAccountLocked = &AppError{
		Code:       "ACCOUNT_LOCKED",
		Message:    "Account is temporarily locked due to too many failed login attempts",
		StatusCode: http.StatusUnauthorized,
	}
	ErrEmailExists = &AppError{
		Code:       "EMAIL_EXISTS",
		Message:    "Email already exists",
		StatusCode: http.StatusConflict,
	}
	ErrPhoneExists = &AppError{
		Code:       "PHONE_EXISTS",
		Message:    "Phone number already exists",
		StatusCode: http.StatusConflict,
	}
)

// Token errors
var (
	ErrTokenExpired = &AppError{
		Code:       "TOKEN_EXPIRED",
		Message:    "Token has expired",
		StatusCode: http.StatusUnauthorized,
	}
	ErrTokenInvalid = &AppError{
		Code:       "TOKEN_INVALID",
		Message:    "Token is invalid",
		StatusCode: http.StatusUnauthorized,
	}
	ErrTokenRevoked = &AppError{
		Code:       "TOKEN_REVOKED",
		Message:    "Token has been revoked",
		StatusCode: http.StatusUnauthorized,
	}
	ErrRefreshTokenInvalid = &AppError{
		Code:       "REFRESH_TOKEN_INVALID",
		Message:    "Refresh token is invalid",
		StatusCode: http.StatusUnauthorized,
	}
	ErrRefreshTokenExpired = &AppError{
		Code:       "REFRESH_TOKEN_EXPIRED",
		Message:    "Refresh token has expired",
		StatusCode: http.StatusUnauthorized,
	}
	ErrTokenBlacklisted = &AppError{
		Code:       "TOKEN_BLACKLISTED",
		Message:    "Token has been blacklisted",
		StatusCode: http.StatusUnauthorized,
	}
)

// Permission errors
var (
	ErrPermissionDenied = &AppError{
		Code:       "PERMISSION_DENIED",
		Message:    "Permission denied",
		StatusCode: http.StatusForbidden,
	}
	ErrInsufficientPermissions = &AppError{
		Code:       "INSUFFICIENT_PERMISSIONS",
		Message:    "Insufficient permissions to perform this action",
		StatusCode: http.StatusForbidden,
	}
)

// Validation errors
var (
	ErrValidationFailed = &AppError{
		Code:       "VALIDATION_FAILED",
		Message:    "Validation failed",
		StatusCode: http.StatusBadRequest,
	}
	ErrInvalidInput = &AppError{
		Code:       "INVALID_INPUT",
		Message:    "Invalid input data",
		StatusCode: http.StatusBadRequest,
	}
)

// Resource errors
var (
	ErrResourceNotFound = &AppError{
		Code:       "RESOURCE_NOT_FOUND",
		Message:    "Resource not found",
		StatusCode: http.StatusNotFound,
	}
	ErrRoleNotFound = &AppError{
		Code:       "ROLE_NOT_FOUND",
		Message:    "Role not found",
		StatusCode: http.StatusNotFound,
	}
)

// Internal errors
var (
	ErrInternal = &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "An internal error occurred",
		StatusCode: http.StatusInternalServerError,
	}
	ErrDatabaseError = &AppError{
		Code:       "DATABASE_ERROR",
		Message:    "Database error occurred",
		StatusCode: http.StatusInternalServerError,
	}
	ErrCacheError = &AppError{
		Code:       "CACHE_ERROR",
		Message:    "Cache error occurred",
		StatusCode: http.StatusInternalServerError,
	}
)

func IsAppError(err error) bool {
	_, ok := err.(*AppError)
	return ok
}

func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return Wrap(err, CodeInternal, "An internal error occurred", http.StatusInternalServerError)
}
