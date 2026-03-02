package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/outbound"
	"github.com/nexlified/dam/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type WebhookServiceImpl struct {
	webhookRepo outbound.WebhookRepository
}

func NewWebhookService(webhookRepo outbound.WebhookRepository) *WebhookServiceImpl {
	return &WebhookServiceImpl{webhookRepo: webhookRepo}
}

func (s *WebhookServiceImpl) CreateWebhook(ctx context.Context, orgID uuid.UUID, url string, events []string) (*domain.Webhook, string, error) {
	secret, err := generateSecureToken(32)
	if err != nil {
		return nil, "", err
	}
	secretHash, err := bcrypt.GenerateFromPassword([]byte(secret), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	wh := &domain.Webhook{
		ID:         uuid.New(),
		OrgID:      orgID,
		URL:        url,
		Events:     events,
		SecretHash: string(secretHash),
		Active:     true,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
	if err := s.webhookRepo.Create(ctx, wh); err != nil {
		return nil, "", err
	}
	return wh, secret, nil
}

func (s *WebhookServiceImpl) ListWebhooks(ctx context.Context, orgID uuid.UUID) ([]*domain.Webhook, error) {
	return s.webhookRepo.ListByOrg(ctx, orgID)
}

func (s *WebhookServiceImpl) GetWebhook(ctx context.Context, orgID, webhookID uuid.UUID) (*domain.Webhook, error) {
	wh, err := s.webhookRepo.GetByID(ctx, webhookID)
	if err != nil {
		return nil, err
	}
	if wh.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	return wh, nil
}

func (s *WebhookServiceImpl) UpdateWebhook(ctx context.Context, orgID, webhookID uuid.UUID, url string, events []string, active bool) (*domain.Webhook, error) {
	wh, err := s.GetWebhook(ctx, orgID, webhookID)
	if err != nil {
		return nil, err
	}
	wh.URL = url
	wh.Events = events
	wh.Active = active
	wh.UpdatedAt = time.Now().UTC()
	if err := s.webhookRepo.Update(ctx, wh); err != nil {
		return nil, err
	}
	return wh, nil
}

func (s *WebhookServiceImpl) DeleteWebhook(ctx context.Context, orgID, webhookID uuid.UUID) error {
	wh, err := s.GetWebhook(ctx, orgID, webhookID)
	if err != nil {
		return err
	}
	return s.webhookRepo.Delete(ctx, wh.ID)
}

func (s *WebhookServiceImpl) TestWebhook(ctx context.Context, orgID, webhookID uuid.UUID) error {
	wh, err := s.GetWebhook(ctx, orgID, webhookID)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{
		"event": "test",
		"org_id": orgID.String(),
		"webhook_id": webhookID.String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	return s.deliverWebhook(ctx, wh, "test", payload)
}

func (s *WebhookServiceImpl) ListDeliveries(ctx context.Context, orgID, webhookID uuid.UUID, limit, offset int) ([]*domain.WebhookDelivery, error) {
	wh, err := s.GetWebhook(ctx, orgID, webhookID)
	if err != nil {
		return nil, err
	}
	return s.webhookRepo.ListDeliveries(ctx, wh.ID, limit, offset)
}

func (s *WebhookServiceImpl) DispatchEvent(ctx context.Context, event *domain.Event) error {
	webhooks, err := s.webhookRepo.ListActiveByEvent(ctx, event.OrgID, string(event.Type))
	if err != nil {
		return err
	}
	for _, wh := range webhooks {
		payload := map[string]interface{}{
			"event":    string(event.Type),
			"id":       event.ID.String(),
			"org_id":   event.OrgID.String(),
			"payload":  event.Payload,
			"created_at": event.CreatedAt.Format(time.RFC3339),
		}
		// Create delivery record (async delivery handled by background worker)
		payloadBytes, _ := json.Marshal(payload)
		delivery := &domain.WebhookDelivery{
			ID:        uuid.New(),
			WebhookID: wh.ID,
			Event:     string(event.Type),
			Payload:   string(payloadBytes),
			Status:    "pending",
			Attempts:  0,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		_ = s.webhookRepo.CreateDelivery(ctx, delivery)
		// For MVP, attempt immediate delivery
		go func(wh *domain.Webhook, d *domain.WebhookDelivery) {
			bgCtx := context.Background()
			if err := s.deliverWebhook(bgCtx, wh, d.Event, payload); err != nil {
				d.Status = "failed"
				d.Attempts++
				nextRetry := time.Now().Add(time.Duration(d.Attempts*d.Attempts) * time.Minute)
				d.NextRetryAt = &nextRetry
			} else {
				d.Status = "delivered"
				d.Attempts++
			}
			d.UpdatedAt = time.Now().UTC()
			_ = s.webhookRepo.UpdateDelivery(bgCtx, d)
		}(wh, delivery)
	}
	return nil
}

func (s *WebhookServiceImpl) deliverWebhook(ctx context.Context, wh *domain.Webhook, event string, payload interface{}) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	sig := computeWebhookSignature(wh.SecretHash, body)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-DAM-Event", event)
	req.Header.Set("X-DAM-Signature", fmt.Sprintf("sha256=%s", sig))
	req.Header.Set("X-DAM-Delivery", uuid.NewString())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook endpoint returned status %d", resp.StatusCode)
	}
	return nil
}

func computeWebhookSignature(secretHash string, body []byte) string {
	// Use the stored secret hash as HMAC key (simplified; production should store raw secret encrypted)
	mac := hmac.New(sha256.New, []byte(secretHash))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
