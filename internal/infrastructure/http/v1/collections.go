package v1

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/ports/inbound"
	"github.com/nexlified/dam/ports/outbound"
)

type CollectionsHandler struct {
	collectionSvc inbound.CollectionService
	storage       outbound.StoragePort
}

func NewCollectionsHandler(collectionSvc inbound.CollectionService) *CollectionsHandler {
	return &CollectionsHandler{collectionSvc: collectionSvc}
}

func (h *CollectionsHandler) Create(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return fiber.ErrBadRequest
	}
	col, err := h.collectionSvc.CreateCollection(c.Context(), orgID, req.Name, req.Description)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(col)
}

func (h *CollectionsHandler) List(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	cols, err := h.collectionSvc.ListCollections(c.Context(), orgID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": cols})
}

func (h *CollectionsHandler) Get(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	colID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	col, err := h.collectionSvc.GetCollection(c.Context(), orgID, colID)
	if err != nil {
		return err
	}
	return c.JSON(col)
}

func (h *CollectionsHandler) Update(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	colID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	col, err := h.collectionSvc.UpdateCollection(c.Context(), orgID, colID, req.Name, req.Description)
	if err != nil {
		return err
	}
	return c.JSON(col)
}

func (h *CollectionsHandler) Delete(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	colID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.collectionSvc.DeleteCollection(c.Context(), orgID, colID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *CollectionsHandler) AddAsset(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	colID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	assetID, err := uuid.Parse(c.Params("assetId"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.collectionSvc.AddAsset(c.Context(), orgID, colID, assetID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *CollectionsHandler) RemoveAsset(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	colID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	assetID, err := uuid.Parse(c.Params("assetId"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.collectionSvc.RemoveAsset(c.Context(), orgID, colID, assetID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *CollectionsHandler) ListAssets(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	colID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	assets, err := h.collectionSvc.ListAssets(c.Context(), orgID, colID, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": assets})
}
