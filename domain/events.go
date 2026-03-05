package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventAssetCreated     EventType = "asset.created"
	EventAssetUpdated     EventType = "asset.updated"
	EventAssetDeleted     EventType = "asset.deleted"
	EventAssetTransformed EventType = "asset.transformed"
	EventUploadFailed     EventType = "upload.failed"
)

type Event struct {
	ID        uuid.UUID   `json:"id"`
	Type      EventType   `json:"type"`
	OrgID     uuid.UUID   `json:"org_id"`
	Payload   interface{} `json:"payload"`
	CreatedAt time.Time   `json:"created_at"`
}

type AssetCreatedPayload struct {
	Asset *Asset `json:"asset"`
}

type AssetUpdatedPayload struct {
	Asset *Asset `json:"asset"`
}

type AssetDeletedPayload struct {
	AssetID uuid.UUID `json:"asset_id"`
	OrgID   uuid.UUID `json:"org_id"`
}

type AssetTransformedPayload struct {
	AssetID    uuid.UUID `json:"asset_id"`
	StyleID    *uuid.UUID `json:"style_id,omitempty"`
	StorageKey string    `json:"storage_key"`
}

type UploadFailedPayload struct {
	Filename string `json:"filename"`
	Error    string `json:"error"`
}

func NewEvent(eventType EventType, orgID uuid.UUID, payload interface{}) *Event {
	return &Event{
		ID:        uuid.New(),
		Type:      eventType,
		OrgID:     orgID,
		Payload:   payload,
		CreatedAt: time.Now().UTC(),
	}
}

// Webhook stores webhook configuration.
type Webhook struct {
	ID         uuid.UUID `db:"id"          json:"id"`
	OrgID      uuid.UUID `db:"org_id"      json:"org_id"`
	URL        string    `db:"url"         json:"url"`
	Events     []string  `db:"events"      json:"events"`
	SecretHash string    `db:"secret_hash" json:"-"`
	Secret     string    `db:"-"           json:"secret,omitempty"` // only set on create
	Active     bool      `db:"active"      json:"active"`
	CreatedAt  time.Time `db:"created_at"  json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"  json:"updated_at"`
}

type WebhookDelivery struct {
	ID          uuid.UUID  `db:"id"            json:"id"`
	WebhookID   uuid.UUID  `db:"webhook_id"    json:"webhook_id"`
	Event       string     `db:"event"         json:"event"`
	Payload     string     `db:"payload"       json:"payload"`
	Status      string     `db:"status"        json:"status"` // "pending", "delivered", "failed"
	Attempts    int        `db:"attempts"      json:"attempts"`
	NextRetryAt *time.Time `db:"next_retry_at" json:"next_retry_at"`
	CreatedAt   time.Time  `db:"created_at"    json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"    json:"updated_at"`
}

type AuditLog struct {
	ID           uuid.UUID  `db:"id"`
	OrgID        uuid.UUID  `db:"org_id"`
	UserID       *uuid.UUID `db:"user_id"`
	Action       string     `db:"action"`
	ResourceType string     `db:"resource_type"`
	ResourceID   *string    `db:"resource_id"`
	Metadata     map[string]interface{} `db:"metadata"`
	IP           string     `db:"ip"`
	CreatedAt    time.Time  `db:"created_at"`
}
