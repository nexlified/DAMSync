package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
)

type TagsHandler struct {
	tagSvc inbound.TagService
}

func NewTagsHandler(tagSvc inbound.TagService) *TagsHandler {
	return &TagsHandler{tagSvc: tagSvc}
}

func (h *TagsHandler) Create(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return fiber.ErrBadRequest
	}
	tag, err := h.tagSvc.CreateTag(c.Context(), orgID, req.Name)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(tag)
}

func (h *TagsHandler) List(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	tags, err := h.tagSvc.ListTags(c.Context(), orgID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": tags})
}

func (h *TagsHandler) Delete(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	tagID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.tagSvc.DeleteTag(c.Context(), orgID, tagID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *TagsHandler) TagAsset(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	var req struct {
		TagIDs []uuid.UUID `json:"tag_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.tagSvc.TagAsset(c.Context(), orgID, assetID, req.TagIDs); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *TagsHandler) UntagAsset(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	var req struct {
		TagIDs []uuid.UUID `json:"tag_ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.tagSvc.UntagAsset(c.Context(), orgID, assetID, req.TagIDs); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
