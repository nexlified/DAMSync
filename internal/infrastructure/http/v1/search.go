package v1

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
	"github.com/nexlified/dam/internal/domain"
)

type SearchHandler struct {
	searchSvc inbound.SearchService
}

func NewSearchHandler(searchSvc inbound.SearchService) *SearchHandler {
	return &SearchHandler{searchSvc: searchSvc}
}

func (h *SearchHandler) Search(c *fiber.Ctx) error {
	orgID := mustOrgID(c)

	filter := domain.AssetListFilter{
		OrgID:    orgID,
		Search:   c.Query("q"),
		MIMEGroup: c.Query("mime_group"),
		SortBy:   c.Query("sort_by"),
		SortDir:  c.Query("sort_dir"),
		Cursor:   c.Query("cursor"),
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

	if vis := c.Query("visibility"); vis != "" {
		v := domain.Visibility(vis)
		filter.Visibility = &v
	}

	if from := c.Query("date_from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err == nil {
			filter.DateFrom = &t
		}
	}

	if to := c.Query("date_to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err == nil {
			filter.DateTo = &t
		}
	}

	if sizeMin, err := strconv.ParseInt(c.Query("size_min"), 10, 64); err == nil {
		filter.SizeMin = &sizeMin
	}
	if sizeMax, err := strconv.ParseInt(c.Query("size_max"), 10, 64); err == nil {
		filter.SizeMax = &sizeMax
	}

	assets, cursor, total, err := h.searchSvc.Search(c.Context(), filter)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"data":        assets,
		"total":       total,
		"next_cursor": cursor,
	})
}
