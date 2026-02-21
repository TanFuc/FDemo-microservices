package http

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"microservices/profile/internal/model"
	"microservices/profile/internal/service"
	"microservices/profile/pkg/errors"
	"microservices/profile/pkg/response"
)

type ProfileHandler struct {
	profileService service.ProfileService
	validate       *validator.Validate
}

func NewProfileHandler(profileService service.ProfileService) *ProfileHandler {
	return &ProfileHandler{
		profileService: profileService,
		validate:       validator.New(),
	}
}

// GetMyProfile godoc
// @Summary Get current user profile
// @Tags profiles
// @Produce json
// @Param X-User-ID header string true "User ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.ErrorResponse
// @Router /profiles/me [get]
func (h *ProfileHandler) GetMyProfile(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return response.Error(c, errors.ErrUnauthorized)
	}

	profile, err := h.profileService.GetOrCreateProfile(c.Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, profile)
}

// UpdateMyProfile godoc
// @Summary Update current user profile
// @Tags profiles
// @Accept json
// @Produce json
// @Param X-User-ID header string true "User ID"
// @Param request body map[string]interface{} true "Profile updates"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.ErrorResponse
// @Failure 401 {object} response.ErrorResponse
// @Router /profiles/me [patch]
func (h *ProfileHandler) UpdateMyProfile(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return response.Error(c, errors.ErrUnauthorized)
	}

	var updates map[string]interface{}
	if err := c.BodyParser(&updates); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	profile, err := h.profileService.UpdateProfile(c.Context(), userID, updates)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, profile)
}

// RegisterShop godoc
// @Summary Register a shop
// @Tags profiles
// @Accept json
// @Produce json
// @Param X-User-ID header string true "User ID"
// @Param request body model.RegisterShopRequest true "Shop details"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.ErrorResponse
// @Failure 409 {object} response.ErrorResponse
// @Router /profiles/me/shop [post]
func (h *ProfileHandler) RegisterShop(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return response.Error(c, errors.ErrUnauthorized)
	}

	var req model.RegisterShopRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	profile, err := h.profileService.RegisterShop(c.Context(), userID, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, profile)
}

// UpdateShop godoc
// @Summary Update shop details
// @Tags profiles
// @Accept json
// @Produce json
// @Param X-User-ID header string true "User ID"
// @Param request body model.UpdateShopRequest true "Shop updates"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.ErrorResponse
// @Failure 404 {object} response.ErrorResponse
// @Router /profiles/me/shop [patch]
func (h *ProfileHandler) UpdateShop(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return response.Error(c, errors.ErrUnauthorized)
	}

	var req model.UpdateShopRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	profile, err := h.profileService.UpdateShop(c.Context(), userID, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, profile)
}

// GetAddresses godoc
// @Summary Get all addresses
// @Tags profiles
// @Produce json
// @Param X-User-ID header string true "User ID"
// @Success 200 {object} response.Response
// @Failure 401 {object} response.ErrorResponse
// @Router /profiles/me/addresses [get]
func (h *ProfileHandler) GetAddresses(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return response.Error(c, errors.ErrUnauthorized)
	}

	addresses, err := h.profileService.GetAddresses(c.Context(), userID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, addresses)
}

// CreateAddress godoc
// @Summary Create new address
// @Tags profiles
// @Accept json
// @Produce json
// @Param X-User-ID header string true "User ID"
// @Param request body model.CreateAddressRequest true "Address details"
// @Success 201 {object} response.Response
// @Failure 400 {object} response.ErrorResponse
// @Router /profiles/me/addresses [post]
func (h *ProfileHandler) CreateAddress(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return response.Error(c, errors.ErrUnauthorized)
	}

	var req model.CreateAddressRequest
	if err := c.BodyParser(&req); err != nil {
		return response.BadRequest(c, "Invalid request body")
	}

	if err := h.validate.Struct(&req); err != nil {
		return response.ValidationError(c, formatValidationErrors(err))
	}

	address, err := h.profileService.AddAddress(c.Context(), userID, &req)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Created(c, address)
}

// GetAddress godoc
// @Summary Get address by ID
// @Tags profiles
// @Produce json
// @Param X-User-ID header string true "User ID"
// @Param id path string true "Address ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.ErrorResponse
// @Router /profiles/me/addresses/{id} [get]
func (h *ProfileHandler) GetAddress(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return response.Error(c, errors.ErrUnauthorized)
	}

	addressID := c.Params("id")
	address, err := h.profileService.GetAddressByID(c.Context(), userID, addressID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, address)
}

// SetDefaultAddress godoc
// @Summary Set address as default
// @Tags profiles
// @Produce json
// @Param X-User-ID header string true "User ID"
// @Param id path string true "Address ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.ErrorResponse
// @Router /profiles/me/addresses/{id}/set-default [patch]
func (h *ProfileHandler) SetDefaultAddress(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return response.Error(c, errors.ErrUnauthorized)
	}

	addressID := c.Params("id")
	address, err := h.profileService.SetDefaultAddress(c.Context(), userID, addressID)
	if err != nil {
		return response.Error(c, err)
	}

	return response.Success(c, address)
}

// DeleteAddress godoc
// @Summary Delete address
// @Tags profiles
// @Produce json
// @Param X-User-ID header string true "User ID"
// @Param id path string true "Address ID"
// @Success 200 {object} response.Response
// @Failure 404 {object} response.ErrorResponse
// @Router /profiles/me/addresses/{id} [delete]
func (h *ProfileHandler) DeleteAddress(c *fiber.Ctx) error {
	userID := c.Get("X-User-ID")
	if userID == "" {
		return response.Error(c, errors.ErrUnauthorized)
	}

	addressID := c.Params("id")
	if err := h.profileService.DeleteAddress(c.Context(), userID, addressID); err != nil {
		return response.Error(c, err)
	}

	return response.SuccessWithMessage(c, nil, "Address deleted successfully")
}

func formatValidationErrors(err error) []string {
	var errs []string
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, e := range validationErrors {
			errs = append(errs, e.Field()+" is invalid")
		}
	}
	return errs
}
