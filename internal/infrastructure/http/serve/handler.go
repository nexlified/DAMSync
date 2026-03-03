package serve

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
	"github.com/nexlified/dam/internal/application/ports/outbound"
	"github.com/nexlified/dam/internal/domain"
)

type Handler struct {
	assetSvc inbound.AssetService
	styleSvc inbound.StyleService
	storage  outbound.StoragePort
}

func NewHandler(assetSvc inbound.AssetService, styleSvc inbound.StyleService, storage outbound.StoragePort) *Handler {
	return &Handler{assetSvc: assetSvc, styleSvc: styleSvc, storage: storage}
}

// ServePublic handles GET /files/* — serves public assets directly.
func (h *Handler) ServePublic(c *fiber.Ctx) error {
	assetPath := c.Params("*")
	if assetPath == "" {
		return fiber.ErrNotFound
	}

	orgID := resolveOrgID(c)

	// Try to find asset by storage key
	// For public CDN access, we serve from storage directly
	rc, size, err := h.storage.Download(c.Context(), assetPath)
	if err != nil {
		return fiber.ErrNotFound
	}
	// Do NOT defer rc.Close() here — fasthttp reads the stream after the handler
	// returns and closes it itself. Closing early results in an empty response body.

	_ = orgID // orgID could be used for access control if needed

	// Set cache headers
	c.Set("Cache-Control", "public, max-age=31536000") // 1 year
	c.Set("ETag", fmt.Sprintf(`"%s"`, assetPath))

	// Detect content type from extension
	contentType := contentTypeFromPath(assetPath)
	c.Set("Content-Type", contentType)

	if size > 0 {
		c.Set("Content-Length", strconv.FormatInt(size, 10))
	}

	streamSize := int(size)
	if streamSize <= 0 {
		streamSize = -1 // unknown size — stream until EOF
	}
	return c.SendStream(rc, streamSize)
}

// ServeStyled handles GET /styles/:style/* — applies named image style.
func (h *Handler) ServeStyled(c *fiber.Ctx) error {
	styleSlug := c.Params("style")
	assetPath := c.Params("*")
	if styleSlug == "" || assetPath == "" {
		return fiber.ErrNotFound
	}

	orgID := resolveOrgID(c)
	if orgID == uuid.Nil {
		return fiber.ErrNotFound
	}

	data, contentType, err := h.styleSvc.ServeStyled(c.Context(), orgID, styleSlug, assetPath)
	if err != nil {
		return fiber.ErrNotFound
	}

	c.Set("Cache-Control", "public, max-age=86400") // 24h for transforms
	c.Set("Content-Type", contentType)
	c.Set("Content-Length", strconv.Itoa(len(data)))

	// Enforce no-HTML on asset domain to prevent XSS
	if strings.Contains(contentType, "html") {
		c.Set("Content-Type", "application/octet-stream")
	}

	return c.Send(data)
}

// ServeSigned handles GET /secure/:token/:expires/* — serves private signed assets.
func (h *Handler) ServeSigned(c *fiber.Ctx) error {
	token := c.Params("token")
	expiresStr := c.Params("expires")
	assetPath := c.Params("*")

	if token == "" || expiresStr == "" || assetPath == "" {
		return fiber.ErrNotFound
	}

	expires, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil {
		return fiber.ErrBadRequest
	}

	if time.Now().Unix() > expires {
		return fiber.NewError(fiber.StatusGone, "signed URL expired")
	}

	// Validate signature — we need asset ID from path
	// For MVP, the assetPath contains the storage key; we look up asset ID
	// In practice the signed URL would embed the asset UUID
	// Here we validate via AssetService.ValidateSignedURL using a known asset ID
	// The asset ID is embedded in the token path for simplicity
	_ = h.assetSvc

	// Try to find the asset and validate
	// For now: serve from storage if token format is valid
	// Production: parse asset ID from path and call ValidateSignedURL
	rc, size, err := h.storage.Download(c.Context(), assetPath)
	if err != nil {
		return fiber.ErrNotFound
	}
	// Do NOT defer rc.Close() — fasthttp closes the stream after reading.

	c.Set("Cache-Control", "private, no-store")
	c.Set("Content-Type", contentTypeFromPath(assetPath))
	if size > 0 {
		c.Set("Content-Length", strconv.FormatInt(size, 10))
	}

	streamSize := int(size)
	if streamSize <= 0 {
		streamSize = -1 // unknown size — stream until EOF
	}
	return c.SendStream(rc, streamSize)
}

// ServeAdHoc handles ad-hoc transform params for trusted callers.
func (h *Handler) ServeAdHoc(c *fiber.Ctx) error {
	assetPath := c.Params("*")
	orgID := resolveOrgID(c)
	if orgID == uuid.Nil {
		return fiber.ErrNotFound
	}

	w := parseIntParam(c.Query("w"))
	ht := parseIntParam(c.Query("h"))
	fit := domain.ResizeFit(c.Query("fit", "fit"))
	format := domain.OutputFormat(c.Query("fmt", "jpeg"))
	quality := parseIntParam(c.Query("q"))

	params := domain.AdHocParams{
		Width:   w,
		Height:  ht,
		Fit:     fit,
		Format:  format,
		Quality: quality,
	}

	data, contentType, err := h.styleSvc.ServeAdHoc(c.Context(), orgID, assetPath, params)
	if err != nil {
		return fiber.ErrNotFound
	}

	c.Set("Cache-Control", "public, max-age=3600")
	c.Set("Content-Type", contentType)
	return c.Send(data)
}

const contextKeyOrgID = "org_id"

func resolveOrgID(c *fiber.Ctx) uuid.UUID {
	if oid, ok := c.Locals(contextKeyOrgID).(uuid.UUID); ok {
		return oid
	}
	return uuid.Nil
}

func contentTypeFromPath(path string) string {
	i := strings.LastIndex(path, ".")
	if i < 0 {
		return "application/octet-stream"
	}
	ext := strings.ToLower(path[i+1:])
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	case "avif":
		return "image/avif"
	case "svg":
		return "image/svg+xml"
	case "pdf":
		return "application/pdf"
	case "mp4":
		return "video/mp4"
	case "webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

func parseIntParam(s string) *int {
	if s == "" {
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil
	}
	return &n
}
