package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
	"github.com/nexlified/dam/internal/application/services"
	"github.com/nexlified/dam/internal/domain"
	"github.com/nexlified/dam/internal/infrastructure/http/middleware"
)

type OrgsHandler struct {
	orgSvc  inbound.OrgService
	authSvc *services.AuthServiceImpl
}

func NewOrgsHandler(orgSvc inbound.OrgService, authSvc *services.AuthServiceImpl) *OrgsHandler {
	return &OrgsHandler{orgSvc: orgSvc, authSvc: authSvc}
}

func (h *OrgsHandler) GetCurrentOrg(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	org, err := h.orgSvc.GetOrg(c.Context(), orgID)
	if err != nil {
		return err
	}
	return c.JSON(org)
}

func (h *OrgsHandler) UpdateOrg(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	var req inbound.UpdateOrgRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	org, err := h.orgSvc.UpdateOrg(c.Context(), orgID, req)
	if err != nil {
		return err
	}
	return c.JSON(org)
}

func (h *OrgsHandler) GetStorageUsage(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	usage, err := h.orgSvc.GetStorageUsage(c.Context(), orgID)
	if err != nil {
		return err
	}
	return c.JSON(usage)
}

func (h *OrgsHandler) CreateUser(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	var req inbound.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	user, err := h.orgSvc.CreateUser(c.Context(), orgID, req)
	if err != nil {
		return err
	}
	user.PasswordHash = ""
	return c.Status(fiber.StatusCreated).JSON(user)
}

func (h *OrgsHandler) ListUsers(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	users, err := h.orgSvc.ListUsers(c.Context(), orgID)
	if err != nil {
		return err
	}
	for _, u := range users {
		u.PasswordHash = ""
	}
	return c.JSON(fiber.Map{"data": users})
}

func (h *OrgsHandler) GetUser(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	user, err := h.orgSvc.GetUser(c.Context(), orgID, userID)
	if err != nil {
		return err
	}
	user.PasswordHash = ""
	return c.JSON(user)
}

func (h *OrgsHandler) UpdateUser(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	var req inbound.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	user, err := h.orgSvc.UpdateUser(c.Context(), orgID, userID, req)
	if err != nil {
		return err
	}
	user.PasswordHash = ""
	return c.JSON(user)
}

func (h *OrgsHandler) DeleteUser(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	userID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.orgSvc.DeleteUser(c.Context(), orgID, userID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *OrgsHandler) CreateAPIKey(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	var req inbound.CreateAPIKeyRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	if len(req.Scopes) == 0 {
		req.Scopes = []string{domain.ScopeAssetsRead}
	}

	var userID *uuid.UUID
	if claims := middleware.GetClaims(c); claims != nil {
		uid := claims.UserID
		userID = &uid
	}

	key, rawKey, err := h.authSvc.CreateAPIKey(c.Context(), orgID, userID, req)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"api_key": key,
		"key":     rawKey,
	})
}

func (h *OrgsHandler) ListAPIKeys(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	keys, err := h.authSvc.ListAPIKeys(c.Context(), orgID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": keys})
}

func (h *OrgsHandler) RevokeAPIKey(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	keyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.authSvc.RevokeAPIKey(c.Context(), orgID, keyID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// mustOrgID extracts org ID from JWT claims or API key.
func mustOrgID(c *fiber.Ctx) uuid.UUID {
	if claims := middleware.GetClaims(c); claims != nil {
		return claims.OrgID
	}
	if key := middleware.GetAPIKey(c); key != nil {
		return key.OrgID
	}
	if oid, ok := c.Locals(middleware.ContextKeyOrgID).(uuid.UUID); ok {
		return oid
	}
	return uuid.Nil
}
