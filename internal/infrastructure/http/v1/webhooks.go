package v1

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
)

type WebhooksHandler struct {
	webhookSvc inbound.WebhookService
}

func NewWebhooksHandler(webhookSvc inbound.WebhookService) *WebhooksHandler {
	return &WebhooksHandler{webhookSvc: webhookSvc}
}

func (h *WebhooksHandler) Create(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	var req struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
	}
	if err := c.BodyParser(&req); err != nil || req.URL == "" {
		return fiber.ErrBadRequest
	}
	wh, secret, err := h.webhookSvc.CreateWebhook(c.Context(), orgID, req.URL, req.Events)
	if err != nil {
		return err
	}
	wh.Secret = secret // shown only once
	return c.Status(fiber.StatusCreated).JSON(wh)
}

func (h *WebhooksHandler) List(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	webhooks, err := h.webhookSvc.ListWebhooks(c.Context(), orgID)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": webhooks})
}

func (h *WebhooksHandler) Get(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	webhookID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	wh, err := h.webhookSvc.GetWebhook(c.Context(), orgID, webhookID)
	if err != nil {
		return err
	}
	return c.JSON(wh)
}

func (h *WebhooksHandler) Update(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	webhookID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	var req struct {
		URL    string   `json:"url"`
		Events []string `json:"events"`
		Active bool     `json:"active"`
	}
	if err := c.BodyParser(&req); err != nil {
		return fiber.ErrBadRequest
	}
	wh, err := h.webhookSvc.UpdateWebhook(c.Context(), orgID, webhookID, req.URL, req.Events, req.Active)
	if err != nil {
		return err
	}
	return c.JSON(wh)
}

func (h *WebhooksHandler) Delete(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	webhookID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.webhookSvc.DeleteWebhook(c.Context(), orgID, webhookID); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *WebhooksHandler) Test(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	webhookID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	if err := h.webhookSvc.TestWebhook(c.Context(), orgID, webhookID); err != nil {
		return err
	}
	return c.JSON(fiber.Map{"message": "test delivery sent"})
}

func (h *WebhooksHandler) ListDeliveries(c *fiber.Ctx) error {
	orgID := mustOrgID(c)
	webhookID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.ErrBadRequest
	}
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	deliveries, err := h.webhookSvc.ListDeliveries(c.Context(), orgID, webhookID, limit, offset)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": deliveries})
}
