package v1

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
	"github.com/nexlified/dam/internal/application/ports/outbound"
	"github.com/nexlified/dam/internal/domain"
)

type AssetsHandler struct {
	assetSvc   inbound.AssetService
	storage    outbound.StoragePort
	cdnBaseURL string
}

func NewAssetsHandler(assetSvc inbound.AssetService, storage outbound.StoragePort, cdnBaseURL string) *AssetsHandler {
	return &AssetsHandler{assetSvc: assetSvc, storage: storage, cdnBaseURL: cdnBaseURL}
}

// assetResponse wraps domain.Asset and adds a synthesised CDN URL field.
type assetResponse struct {
	*domain.Asset
	URL string `json:"url"`
}

func (h *AssetsHandler) enrichAsset(c *fiber.Ctx, asset *domain.Asset) *assetResponse {
	base := h.cdnBaseURL
	if base == "" {
		base = c.BaseURL()
	}
	return &assetResponse{
		Asset: asset,
		URL:   base + "/files/" + asset.StorageKey,
	}
}

func (h *AssetsHandler) Upload(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	if orgID == uuid.Nil {
		return fiber.ErrUnauthorized
	}

	file, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "file is required")
	}

	f, err := file.Open()
	if err != nil {
		return fiber.ErrInternalServerError
	}
	defer f.Close()

	var folderID *uuid.UUID
	if fid := c.FormValue("folder_id"); fid != "" {
		id, err := uuid.Parse(fid)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "invalid folder_id")
		}
		folderID = &id
	}

	visibility := domain.VisibilityPublic
	if v := c.FormValue("visibility"); v != "" {
		visibility = domain.Visibility(v)
	}

	meta := domain.AssetMetadata{
		Title:       c.FormValue("title"),
		Description: c.FormValue("description"),
		AltText:     c.FormValue("alt_text"),
		Author:      c.FormValue("author"),
	}

	asset, err := h.assetSvc.Upload(
		c.Context(),
		orgID,
		folderID,
		file.Filename,
		f,
		file.Size,
		file.Header.Get("Content-Type"),
		meta,
		visibility,
	)
	if err != nil {
		return err
	}

	return c.Status(fiber.StatusCreated).JSON(h.enrichAsset(c, asset))
}

func (h *AssetsHandler) BulkUpload(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	if orgID == uuid.Nil {
		return fiber.ErrUnauthorized
	}

	form, err := c.MultipartForm()
	if err != nil {
		return fiber.ErrBadRequest
	}

	files := form.File["files"]
	if len(files) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "files field is required")
	}

	var uploadFiles []inbound.UploadFile
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			continue
		}
		uploadFiles = append(uploadFiles, inbound.UploadFile{
			Filename:    fh.Filename,
			Reader:      f,
			Size:        fh.Size,
			ContentType: fh.Header.Get("Content-Type"),
			Visibility:  domain.VisibilityPublic,
		})
	}

	assets, errs, err := h.assetSvc.BulkUpload(c.Context(), orgID, nil, uploadFiles)
	if err != nil {
		return err
	}

	enriched := make([]*assetResponse, len(assets))
	for i, a := range assets {
		enriched[i] = h.enrichAsset(c, a)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"data":   enriched,
		"errors": errs,
	})
}

func (h *AssetsHandler) List(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	filter := domain.AssetListFilter{
		OrgID:  orgID,
		Search: c.Query("q"),
		Cursor: c.Query("cursor"),
		SortBy: c.Query("sort_by"),
		SortDir: c.Query("sort_dir"),
	}

	if limit, err := strconv.Atoi(c.Query("limit", "50")); err == nil {
		filter.Limit = limit
	}

	if fid := c.Query("folder_id"); fid != "" {
		id, err := uuid.Parse(fid)
		if err == nil {
			filter.FolderID = &id
		}
	}

	if mime := c.Query("mime_group"); mime != "" {
		filter.MIMEGroup = mime
	}

	assets, cursor, err := h.assetSvc.ListAssets(c.Context(), filter)
	if err != nil {
		return err
	}

	enriched := make([]*assetResponse, len(assets))
	for i, a := range assets {
		enriched[i] = h.enrichAsset(c, a)
	}

	var nextCursor *string
	if cursor != "" {
		nextCursor = &cursor
	}
	return c.JSON(fiber.Map{
		"data":        enriched,
		"next_cursor": nextCursor,
	})
}

func (h *AssetsHandler) Get(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	asset, err := h.assetSvc.GetAsset(c.Context(), orgID, assetID)
	if err != nil {
		return err
	}

	return c.JSON(h.enrichAsset(c, asset))
}

func (h *AssetsHandler) UpdateMetadata(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	var req struct {
		Metadata   domain.AssetMetadata `json:"metadata"`
		Visibility *domain.Visibility   `json:"visibility"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}

	asset, err := h.assetSvc.UpdateMetadata(c.Context(), orgID, assetID, req.Metadata, req.Visibility)
	if err != nil {
		return err
	}

	return c.JSON(h.enrichAsset(c, asset))
}

func (h *AssetsHandler) Delete(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	if err := h.assetSvc.DeleteAsset(c.Context(), orgID, assetID); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *AssetsHandler) Move(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	var req struct {
		FolderID *uuid.UUID `json:"folder_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}

	if err := h.assetSvc.MoveAsset(c.Context(), orgID, assetID, req.FolderID); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *AssetsHandler) GetSignedURL(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	assetID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}

	ttl := 3600 * time.Second
	if ttlStr := c.Query("ttl"); ttlStr != "" {
		if secs, err := strconv.Atoi(ttlStr); err == nil {
			ttl = time.Duration(secs) * time.Second
		}
	}

	url, err := h.assetSvc.GenerateSignedURL(c.Context(), orgID, assetID, ttl)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"url": url, "expires_in": ttl.Seconds()})
}

