package v1

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
)

type StylesHandler struct {
	styleSvc inbound.StyleService
}

func NewStylesHandler(styleSvc inbound.StyleService) *StylesHandler {
	return &StylesHandler{styleSvc: styleSvc}
}

func (h *StylesHandler) Create(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	var req inbound.CreateStyleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	if req.Name == "" || req.Slug == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name and slug are required")
	}
	style, err := h.styleSvc.CreateStyle(c.Context(), orgID, req)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return fiber.NewError(fiber.StatusConflict, "style with this slug already exists")
		}
		return fiber.ErrInternalServerError
	}
	return c.Status(fiber.StatusCreated).JSON(style)
}

func (h *StylesHandler) List(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	styles, err := h.styleSvc.ListStyles(c.Context(), orgID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": styles})
}

func (h *StylesHandler) Get(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	// Accept either UUID or slug
	idOrSlug := c.Params("id")
	if _, err := uuid.Parse(idOrSlug); err != nil {
		// treat as slug
		style, err := h.styleSvc.GetStyle(c.Context(), orgID, idOrSlug)
		if err != nil {
			return err
		}
		return c.JSON(style)
	}
	style, err := h.styleSvc.GetStyle(c.Context(), orgID, idOrSlug)
	if err != nil {
		return err
	}
	return c.JSON(style)
}

func (h *StylesHandler) Update(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	styleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	var req inbound.CreateStyleRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	style, err := h.styleSvc.UpdateStyle(c.Context(), orgID, styleID, req)
	if err != nil {
		return err
	}
	return c.JSON(style)
}

func (h *StylesHandler) Delete(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	styleID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.styleSvc.DeleteStyle(c.Context(), orgID, styleID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
