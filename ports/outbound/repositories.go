package outbound

import (
	"context"

	"github.com/google/uuid"
	"github.com/nexlified/dam/domain"
)

// OrgRepository manages organization persistence.
type OrgRepository interface {
	Create(ctx context.Context, org *domain.Org) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Org, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Org, error)
	Update(ctx context.Context, org *domain.Org) error
	Delete(ctx context.Context, id uuid.UUID) error
	IncrementStorageUsed(ctx context.Context, orgID uuid.UUID, delta int64) error
	List(ctx context.Context, limit, offset int) ([]*domain.Org, int, error)
}

// UserRepository manages user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.User, error)
}

// APIKeyRepository manages API key persistence.
type APIKeyRepository interface {
	Create(ctx context.Context, key *domain.APIKey) error
	GetByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.APIKey, error)
	Revoke(ctx context.Context, id uuid.UUID) error
	UpdateLastUsed(ctx context.Context, id uuid.UUID) error
}

// AssetRepository manages asset persistence.
type AssetRepository interface {
	Create(ctx context.Context, asset *domain.Asset) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Asset, error)
	GetByStorageKey(ctx context.Context, storageKey string) (*domain.Asset, error)
	Update(ctx context.Context, asset *domain.Asset) error
	SoftDelete(ctx context.Context, id uuid.UUID) error
	HardDelete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, error)
	Count(ctx context.Context, filter domain.AssetListFilter) (int, error)
	AddTags(ctx context.Context, assetID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTags(ctx context.Context, assetID uuid.UUID, tagIDs []uuid.UUID) error
	GetTags(ctx context.Context, assetID uuid.UUID) ([]domain.Tag, error)
	Move(ctx context.Context, assetID uuid.UUID, folderID *uuid.UUID) error
}

// FolderRepository manages folder persistence.
type FolderRepository interface {
	Create(ctx context.Context, folder *domain.Folder) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Folder, error)
	GetByPath(ctx context.Context, orgID uuid.UUID, path string) (*domain.Folder, error)
	Update(ctx context.Context, folder *domain.Folder) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Folder, error)
	ListChildren(ctx context.Context, parentID uuid.UUID) ([]*domain.Folder, error)
}

// TagRepository manages tag persistence.
type TagRepository interface {
	Create(ctx context.Context, tag *domain.Tag) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error)
	GetBySlug(ctx context.Context, orgID uuid.UUID, slug string) (*domain.Tag, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Tag, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// CollectionRepository manages collection persistence.
type CollectionRepository interface {
	Create(ctx context.Context, c *domain.Collection) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Collection, error)
	Update(ctx context.Context, c *domain.Collection) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Collection, error)
	AddAsset(ctx context.Context, collectionID, assetID uuid.UUID, sortOrder int) error
	RemoveAsset(ctx context.Context, collectionID, assetID uuid.UUID) error
	ListAssets(ctx context.Context, collectionID uuid.UUID, limit, offset int) ([]*domain.Asset, error)
}

// DomainRepository manages custom domain records.
type DomainRepository interface {
	Create(ctx context.Context, dr *domain.DomainRecord) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.DomainRecord, error)
	GetByDomain(ctx context.Context, domainName string) (*domain.DomainRecord, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.DomainRecord, error)
	Update(ctx context.Context, dr *domain.DomainRecord) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// StyleRepository manages image style persistence.
type StyleRepository interface {
	Create(ctx context.Context, s *domain.ImageStyle) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ImageStyle, error)
	GetBySlug(ctx context.Context, orgID uuid.UUID, slug string) (*domain.ImageStyle, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.ImageStyle, error)
	Update(ctx context.Context, s *domain.ImageStyle) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// TransformCacheRepository manages transform cache records.
type TransformCacheRepository interface {
	Get(ctx context.Context, assetID uuid.UUID, paramsHash string) (*domain.TransformCache, error)
	Create(ctx context.Context, tc *domain.TransformCache) error
	DeleteByAsset(ctx context.Context, assetID uuid.UUID) error
	DeleteByStyle(ctx context.Context, styleID uuid.UUID) error
}

// WebhookRepository manages webhook persistence.
type WebhookRepository interface {
	Create(ctx context.Context, w *domain.Webhook) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Webhook, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Webhook, error)
	ListActiveByEvent(ctx context.Context, orgID uuid.UUID, event string) ([]*domain.Webhook, error)
	Update(ctx context.Context, w *domain.Webhook) error
	Delete(ctx context.Context, id uuid.UUID) error
	CreateDelivery(ctx context.Context, d *domain.WebhookDelivery) error
	UpdateDelivery(ctx context.Context, d *domain.WebhookDelivery) error
	ListDeliveries(ctx context.Context, webhookID uuid.UUID, limit, offset int) ([]*domain.WebhookDelivery, error)
	ListPendingDeliveries(ctx context.Context, limit int) ([]*domain.WebhookDelivery, error)
}

// AuditLogRepository manages audit log persistence.
type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error
	List(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error)
}
