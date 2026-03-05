package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/nexlified/dam/ports/inbound"
	"github.com/nexlified/dam/internal/application/services"
	"github.com/nexlified/dam/domain"
)

type AuthHandler struct {
	authSvc *services.AuthServiceImpl
	orgSvc  inbound.OrgService
}

func NewAuthHandler(authSvc *services.AuthServiceImpl, orgSvc inbound.OrgService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, orgSvc: orgSvc}
}

type registerRequest struct {
	OrgName  string `json:"org_name"`
	OrgSlug  string `json:"org_slug"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	OrgSlug  string `json:"org_slug"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req registerRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	if req.OrgName == "" || req.OrgSlug == "" || req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "org_name, org_slug, email, and password are required")
	}

	org, err := h.orgSvc.CreateOrg(c.Context(), inbound.CreateOrgRequest{
		Name: req.OrgName,
		Slug: req.OrgSlug,
		Plan: "free",
	})
	if err != nil {
		return fiber.NewError(fiber.StatusConflict, err.Error())
	}

	_, err = h.orgSvc.CreateUser(c.Context(), org.ID, inbound.CreateUserRequest{
		Email:    req.Email,
		Password: req.Password,
		Role:     domain.RoleOwner,
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	pair, err := h.authSvc.LoginWithOrgID(c.Context(), org.ID, req.Email, req.Password)
	if err != nil {
		return fiber.ErrInternalServerError
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"org":    org,
		"tokens": pair,
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	if req.OrgSlug == "" || req.Email == "" || req.Password == "" {
		return fiber.NewError(fiber.StatusBadRequest, "org_slug, email, and password are required")
	}

	org, err := h.orgSvc.GetOrgBySlug(c.Context(), req.OrgSlug)
	if err != nil {
		return fiber.ErrUnauthorized
	}

	pair, err := h.authSvc.LoginWithOrgID(c.Context(), org.ID, req.Email, req.Password)
	if err != nil {
		return fiber.ErrUnauthorized
	}

	return c.JSON(pair)
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
		return fiber.ErrBadRequest
	}

	pair, err := h.authSvc.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return fiber.ErrUnauthorized
	}

	return c.JSON(pair)
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&req); err != nil || req.RefreshToken == "" {
		return fiber.ErrBadRequest
	}

	_ = h.authSvc.Logout(c.Context(), req.RefreshToken)
	return c.SendStatus(fiber.StatusNoContent)
}
