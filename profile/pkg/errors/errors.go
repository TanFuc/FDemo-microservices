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

var (
	ErrProfileNotFound = &AppError{
		Code:       "PROFILE_NOT_FOUND",
		Message:    "Profile not found",
		StatusCode: http.StatusNotFound,
	}
	ErrAddressNotFound = &AppError{
		Code:       "ADDRESS_NOT_FOUND",
		Message:    "Address not found",
		StatusCode: http.StatusNotFound,
	}
	ErrShopNameExists = &AppError{
		Code:       "SHOP_NAME_EXISTS",
		Message:    "Shop name already exists",
		StatusCode: http.StatusConflict,
	}
	ErrShopNotRegistered = &AppError{
		Code:       "SHOP_NOT_REGISTERED",
		Message:    "Shop is not registered",
		StatusCode: http.StatusNotFound,
	}
	ErrUnauthorized = &AppError{
		Code:       "UNAUTHORIZED",
		Message:    "Unauthorized",
		StatusCode: http.StatusUnauthorized,
	}
	ErrInternal = &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "Internal server error",
		StatusCode: http.StatusInternalServerError,
	}
	ErrAlreadyAffiliate = &AppError{
		Code:       "ALREADY_AFFILIATE",
		Message:    "User is already registered as an affiliate",
		StatusCode: http.StatusConflict,
	}
	ErrSelfReferral = &AppError{
		Code:       "SELF_REFERRAL",
		Message:    "Cannot use your own referral code",
		StatusCode: http.StatusBadRequest,
	}
	ErrInvalidReferralCode = &AppError{
		Code:       "INVALID_REFERRAL_CODE",
		Message:    "Invalid referral code",
		StatusCode: http.StatusBadRequest,
	}
	ErrBusinessRequiresImage = &AppError{
		Code:       "BUSINESS_REQUIRES_IMAGE",
		Message:    "Business type shop requires at least one image (logo or banner)",
		StatusCode: http.StatusBadRequest,
	}
	ErrBusinessRequiresAddress = &AppError{
		Code:       "BUSINESS_REQUIRES_ADDRESS",
		Message:    "Business type shop requires a complete shop address",
		StatusCode: http.StatusBadRequest,
	}
	ErrAlreadyHasShop = &AppError{
		Code:       "ALREADY_HAS_SHOP",
		Message:    "User already has a registered shop",
		StatusCode: http.StatusConflict,
	}
)

func GetAppError(err error) *AppError {
	if appErr, ok := err.(*AppError); ok {
		return appErr
	}
	return Wrap(err, CodeInternal, "An internal error occurred", http.StatusInternalServerError)
}
