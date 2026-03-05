package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/ports/inbound"
)

type FoldersHandler struct {
	folderSvc inbound.FolderService
}

func NewFoldersHandler(folderSvc inbound.FolderService) *FoldersHandler {
	return &FoldersHandler{folderSvc: folderSvc}
}

func (h *FoldersHandler) Create(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	var req struct {
		ParentID *uuid.UUID `json:"parent_id"`
		Name     string     `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return fiber.ErrBadRequest
	}
	folder, err := h.folderSvc.CreateFolder(c.Context(), orgID, req.ParentID, req.Name)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(folder)
}

func (h *FoldersHandler) List(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	folders, err := h.folderSvc.ListFolders(c.Context(), orgID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": folders})
}

func (h *FoldersHandler) Tree(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	tree, err := h.folderSvc.GetFolderTree(c.Context(), orgID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": tree})
}

func (h *FoldersHandler) Get(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	folder, err := h.folderSvc.GetFolder(c.Context(), orgID, folderID)
	if err != nil {
		return err
	}
	return c.JSON(folder)
}

func (h *FoldersHandler) Update(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BodyParser(&req); err != nil || req.Name == "" {
		return fiber.ErrBadRequest
	}
	folder, err := h.folderSvc.UpdateFolder(c.Context(), orgID, folderID, req.Name)
	if err != nil {
		return err
	}
	return c.JSON(folder)
}

func (h *FoldersHandler) Delete(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.folderSvc.DeleteFolder(c.Context(), orgID, folderID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
