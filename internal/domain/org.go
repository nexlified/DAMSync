package domain

import (
	"time"

	"github.com/google/uuid"
)

type Org struct {
	ID               uuid.UUID  `db:"id"`
	Name             string     `db:"name"`
	Slug             string     `db:"slug"`
	Plan             string     `db:"plan"`
	StorageQuotaBytes int64     `db:"storage_quota_bytes"`
	StorageUsedBytes  int64     `db:"storage_used_bytes"`
	Settings         OrgSettings `db:"settings"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
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
	ID           uuid.UUID  `db:"id"`
	OrgID        uuid.UUID  `db:"org_id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	Role         Role       `db:"role"`
	Active       bool       `db:"active"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

type APIKey struct {
	ID          uuid.UUID  `db:"id"`
	OrgID       uuid.UUID  `db:"org_id"`
	UserID      *uuid.UUID `db:"user_id"`
	Name        string     `db:"name"`
	KeyPrefix   string     `db:"key_prefix"`
	KeyHash     string     `db:"key_hash"`
	Scopes      []string   `db:"scopes"`
	LastUsedAt  *time.Time `db:"last_used_at"`
	IPAllowlist []string   `db:"ip_allowlist"`
	CreatedAt   time.Time  `db:"created_at"`
	RevokedAt   *time.Time `db:"revoked_at"`
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
