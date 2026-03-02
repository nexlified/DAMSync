package domain

import (
	"time"

	"github.com/google/uuid"
)

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
	VisibilityOrg     Visibility = "org"
)

type Asset struct {
	ID          uuid.UUID        `db:"id"`
	OrgID       uuid.UUID        `db:"org_id"`
	FolderID    *uuid.UUID       `db:"folder_id"`
	Filename    string           `db:"filename"`
	StorageKey  string           `db:"storage_key"`
	MIMEType    string           `db:"mime_type"`
	SizeBytes   int64            `db:"size_bytes"`
	Width       *int             `db:"width"`
	Height      *int             `db:"height"`
	DurationMS  *int64           `db:"duration_ms"`
	Metadata    AssetMetadata    `db:"metadata"`
	Visibility  Visibility       `db:"visibility"`
	FocalPoint  *FocalPoint      `db:"-"`
	Tags        []Tag            `db:"-"`
	CreatedAt   time.Time        `db:"created_at"`
	UpdatedAt   time.Time        `db:"updated_at"`
	DeletedAt   *time.Time       `db:"deleted_at"`
}

type AssetMetadata struct {
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	AltText     string            `json:"alt_text,omitempty"`
	Author      string            `json:"author,omitempty"`
	Custom      map[string]string `json:"custom,omitempty"`
}

type FocalPoint struct {
	X float64 `json:"x"` // 0.0–1.0 relative to width
	Y float64 `json:"y"` // 0.0–1.0 relative to height
}

func (a *Asset) IsImage() bool {
	return len(a.MIMEType) > 6 && a.MIMEType[:6] == "image/"
}

func (a *Asset) IsDeleted() bool {
	return a.DeletedAt != nil
}

type Folder struct {
	ID        uuid.UUID  `db:"id"         json:"id"`
	OrgID     uuid.UUID  `db:"org_id"     json:"org_id"`
	ParentID  *uuid.UUID `db:"parent_id"  json:"parent_id"`
	Name      string     `db:"name"       json:"name"`
	Path      string     `db:"path"       json:"path"` // materialized path e.g. "/photos/2024"
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
}

type Tag struct {
	ID        uuid.UUID `db:"id"`
	OrgID     uuid.UUID `db:"org_id"`
	Name      string    `db:"name"`
	Slug      string    `db:"slug"`
	CreatedAt time.Time `db:"created_at"`
}

type Collection struct {
	ID          uuid.UUID  `db:"id"`
	OrgID       uuid.UUID  `db:"org_id"`
	Name        string     `db:"name"`
	Description string     `db:"description"`
	AssetCount  int        `db:"-"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

type CollectionAsset struct {
	CollectionID uuid.UUID `db:"collection_id"`
	AssetID      uuid.UUID `db:"asset_id"`
	SortOrder    int       `db:"sort_order"`
}

// AssetListFilter holds search/filter parameters for listing assets.
type AssetListFilter struct {
	OrgID      uuid.UUID
	FolderID   *uuid.UUID
	TagIDs     []uuid.UUID
	MIMEGroup  string // "image", "video", "document"
	Search     string
	Visibility *Visibility
	DateFrom   *time.Time
	DateTo     *time.Time
	SizeMin    *int64
	SizeMax    *int64
	SortBy     string // "created_at", "name", "size", "updated_at"
	SortDir    string // "asc", "desc"
	Cursor     string
	Limit      int
}
