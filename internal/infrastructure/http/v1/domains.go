package v1

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/ports/inbound"
)

type DomainsHandler struct {
	domainSvc inbound.DomainService
}

func NewDomainsHandler(domainSvc inbound.DomainService) *DomainsHandler {
	return &DomainsHandler{domainSvc: domainSvc}
}

func (h *DomainsHandler) Add(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	var req struct {
		Domain string `json:"domain"`
	}
	if err := c.BodyParser(&req); err != nil || req.Domain == "" {
		return fiber.ErrBadRequest
	}
	dr, err := h.domainSvc.AddDomain(c.Context(), orgID, req.Domain)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(dr)
}

func (h *DomainsHandler) List(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	domains, err := h.domainSvc.ListDomains(c.Context(), orgID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": domains})
}

func (h *DomainsHandler) Verify(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	domainID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	dr, err := h.domainSvc.VerifyDomain(c.Context(), orgID, domainID)
	if err != nil {
		return err
	}
	return c.JSON(dr)
}

func (h *DomainsHandler) Remove(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	domainID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.domainSvc.RemoveDomain(c.Context(), orgID, domainID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}
