package v1

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
	"github.com/nexlified/dam/internal/domain"
	"github.com/nexlified/dam/seed"
	"gopkg.in/yaml.v3"
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

type defaultStyleEntry struct {
	Name         string                  `yaml:"name"`
	Slug         string                  `yaml:"slug"`
	OutputFormat string                  `yaml:"output_format"`
	Quality      int                     `yaml:"quality"`
	Operations   []defaultStyleOperation `yaml:"operations"`
}

type defaultStyleOperation struct {
	Fit    domain.ResizeFit `yaml:"fit"`
	Width  *int             `yaml:"width"`
	Height *int             `yaml:"height"`
}

type defaultStylesFile struct {
	Styles []defaultStyleEntry `yaml:"styles"`
}

func (h *StylesHandler) ImportDefaults(c *fiber.Ctx) error {
	orgID := mustOrgID(c)

	var file defaultStylesFile
	if err := yaml.Unmarshal(seed.DefaultStylesYAML, &file); err != nil {
		return fiber.ErrInternalServerError
	}

	var results []*domain.ImageStyle
	imported, updated := 0, 0

	for _, entry := range file.Styles {
		ops := make([]domain.StyleOperation, len(entry.Operations))
		for i, op := range entry.Operations {
			ops[i] = domain.StyleOperation{
				Fit:    op.Fit,
				Width:  op.Width,
				Height: op.Height,
			}
		}
		req := inbound.CreateStyleRequest{
			Name:         entry.Name,
			Slug:         entry.Slug,
			OutputFormat: domain.OutputFormat(entry.OutputFormat),
			Quality:      entry.Quality,
			Operations:   ops,
		}

		style, err := h.styleSvc.CreateStyle(c.Context(), orgID, req)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
				existing, getErr := h.styleSvc.GetStyle(c.Context(), orgID, entry.Slug)
				if getErr != nil {
					return fiber.ErrInternalServerError
				}
				style, err = h.styleSvc.UpdateStyle(c.Context(), orgID, existing.ID, req)
				if err != nil {
					return fiber.ErrInternalServerError
				}
				updated++
			} else {
				return fiber.ErrInternalServerError
			}
		} else {
			imported++
		}
		results = append(results, style)
	}

	return c.JSON(fiber.Map{
		"data":     results,
		"imported": imported,
		"updated":  updated,
	})
}
