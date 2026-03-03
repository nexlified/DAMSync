package domain

import (
	"time"

	"github.com/google/uuid"
)

type Org struct {
	ID                uuid.UUID   `db:"id"                  json:"id"`
	Name              string      `db:"name"                json:"name"`
	Slug              string      `db:"slug"                json:"slug"`
	Plan              string      `db:"plan"                json:"plan"`
	StorageQuotaBytes int64       `db:"storage_quota_bytes" json:"storage_quota_bytes"`
	StorageUsedBytes  int64       `db:"storage_used_bytes"  json:"storage_used_bytes"`
	Settings          OrgSettings `db:"settings"            json:"settings"`
	CreatedAt         time.Time   `db:"created_at"          json:"created_at"`
	UpdatedAt         time.Time   `db:"updated_at"          json:"updated_at"`
}

type OrgSettings struct {
	AllowedOrigins       []string `json:"allowed_origins,omitempty"`
	MaxFileSizeBytes     int64    `json:"max_file_size_bytes,omitempty"`
	AllowedMIMETypes     []string `json:"allowed_mime_types,omitempty"`
	StripEXIF            bool     `json:"strip_exif"`
	SignedURLSecret      string   `json:"signed_url_secret,omitempty"`
	WebhookRetentionDays int      `json:"webhook_retention_days,omitempty"`
	SoftDeleteDays       int      `json:"soft_delete_days,omitempty"`
}

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

type User struct {
	ID           uuid.UUID `db:"id"            json:"id"`
	OrgID        uuid.UUID `db:"org_id"        json:"org_id"`
	Email        string    `db:"email"         json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Role         Role      `db:"role"          json:"role"`
	Active       bool      `db:"active"        json:"active"`
	CreatedAt    time.Time `db:"created_at"    json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"    json:"updated_at"`
}

type APIKey struct {
	ID          uuid.UUID  `db:"id"           json:"id"`
	OrgID       uuid.UUID  `db:"org_id"       json:"org_id"`
	UserID      *uuid.UUID `db:"user_id"      json:"user_id"`
	Name        string     `db:"name"         json:"name"`
	KeyPrefix   string     `db:"key_prefix"   json:"key_prefix"`
	KeyHash     string     `db:"key_hash"     json:"-"`
	Scopes      []string   `db:"scopes"       json:"scopes"`
	LastUsedAt  *time.Time `db:"last_used_at" json:"last_used_at"`
	IPAllowlist []string   `db:"ip_allowlist" json:"ip_allowlist"`
	CreatedAt   time.Time  `db:"created_at"   json:"created_at"`
	RevokedAt   *time.Time `db:"revoked_at"   json:"revoked_at"`
}

// APIKey scope constants
const (
	ScopeAssetsRead   = "assets:read"
	ScopeAssetsWrite  = "assets:write"
	ScopeAssetsDelete = "assets:delete"
	ScopeAdmin        = "admin"
)

func (k *APIKey) HasScope(scope string) bool {
	for _, s := range k.Scopes {
		if s == scope || s == ScopeAdmin {
			return true
		}
	}
	return false
}

func (k *APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}
