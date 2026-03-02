package inbound

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/domain"
)

// AuthService handles authentication and authorization.
type AuthService interface {
	Login(ctx context.Context, orgSlug, email, password string) (*TokenPair, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
	Logout(ctx context.Context, refreshToken string) error
	ValidateAccessToken(ctx context.Context, token string) (*Claims, error)
	ValidateAPIKey(ctx context.Context, rawKey string) (*domain.APIKey, error)
	CreateAPIKey(ctx context.Context, orgID uuid.UUID, userID *uuid.UUID, req CreateAPIKeyRequest) (*domain.APIKey, string, error)
	ListAPIKeys(ctx context.Context, orgID uuid.UUID) ([]*domain.APIKey, error)
	RevokeAPIKey(ctx context.Context, orgID, keyID uuid.UUID) error
}

type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type Claims struct {
	UserID uuid.UUID
	OrgID  uuid.UUID
	Email  string
	Role   domain.Role
}

type CreateAPIKeyRequest struct {
	Name        string
	Scopes      []string
	IPAllowlist []string
}

// OrgService manages organizations and users.
type OrgService interface {
	CreateOrg(ctx context.Context, req CreateOrgRequest) (*domain.Org, error)
	GetOrg(ctx context.Context, id uuid.UUID) (*domain.Org, error)
	UpdateOrg(ctx context.Context, id uuid.UUID, req UpdateOrgRequest) (*domain.Org, error)
	DeleteOrg(ctx context.Context, id uuid.UUID) error
	GetOrgBySlug(ctx context.Context, slug string) (*domain.Org, error)
	CreateUser(ctx context.Context, orgID uuid.UUID, req CreateUserRequest) (*domain.User, error)
	ListUsers(ctx context.Context, orgID uuid.UUID) ([]*domain.User, error)
	GetUser(ctx context.Context, orgID, userID uuid.UUID) (*domain.User, error)
	UpdateUser(ctx context.Context, orgID, userID uuid.UUID, req UpdateUserRequest) (*domain.User, error)
	DeleteUser(ctx context.Context, orgID, userID uuid.UUID) error
	GetStorageUsage(ctx context.Context, orgID uuid.UUID) (*StorageUsage, error)
}

type CreateOrgRequest struct {
	Name string
	Slug string
	Plan string
}

type UpdateOrgRequest struct {
	Name     *string
	Settings *domain.OrgSettings
}

type CreateUserRequest struct {
	Email    string
	Password string
	Role     domain.Role
}

type UpdateUserRequest struct {
	Role   *domain.Role
	Active *bool
}

type StorageUsage struct {
	UsedBytes  int64
	QuotaBytes int64
	Percent    float64
}

// AssetService manages assets.
type AssetService interface {
	Upload(ctx context.Context, orgID uuid.UUID, folderID *uuid.UUID, filename string, r io.Reader, size int64, contentType string, metadata domain.AssetMetadata, visibility domain.Visibility) (*domain.Asset, error)
	BulkUpload(ctx context.Context, orgID uuid.UUID, folderID *uuid.UUID, files []UploadFile) ([]*domain.Asset, []BulkUploadError, error)
	GetAsset(ctx context.Context, orgID, assetID uuid.UUID) (*domain.Asset, error)
	ListAssets(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, error)
	UpdateMetadata(ctx context.Context, orgID, assetID uuid.UUID, metadata domain.AssetMetadata, visibility *domain.Visibility) (*domain.Asset, error)
	DeleteAsset(ctx context.Context, orgID, assetID uuid.UUID) error
	MoveAsset(ctx context.Context, orgID, assetID uuid.UUID, folderID *uuid.UUID) error
	GenerateSignedURL(ctx context.Context, orgID, assetID uuid.UUID, ttl time.Duration) (string, error)
	ValidateSignedURL(ctx context.Context, assetID uuid.UUID, token string, timestamp int64) error
}

type UploadFile struct {
	Filename    string
	Reader      io.Reader
	Size        int64
	ContentType string
	Metadata    domain.AssetMetadata
	Visibility  domain.Visibility
}

type BulkUploadError struct {
	Filename string
	Error    string
}

// FolderService manages folders.
type FolderService interface {
	CreateFolder(ctx context.Context, orgID uuid.UUID, parentID *uuid.UUID, name string) (*domain.Folder, error)
	GetFolder(ctx context.Context, orgID, folderID uuid.UUID) (*domain.Folder, error)
	UpdateFolder(ctx context.Context, orgID, folderID uuid.UUID, name string) (*domain.Folder, error)
	DeleteFolder(ctx context.Context, orgID, folderID uuid.UUID) error
	ListFolders(ctx context.Context, orgID uuid.UUID) ([]*domain.Folder, error)
	GetFolderTree(ctx context.Context, orgID uuid.UUID) ([]*FolderNode, error)
}

type FolderNode struct {
	*domain.Folder
	Children []*FolderNode `json:"children"`
}

// TagService manages tags.
type TagService interface {
	CreateTag(ctx context.Context, orgID uuid.UUID, name string) (*domain.Tag, error)
	ListTags(ctx context.Context, orgID uuid.UUID) ([]*domain.Tag, error)
	DeleteTag(ctx context.Context, orgID, tagID uuid.UUID) error
	TagAsset(ctx context.Context, orgID, assetID uuid.UUID, tagIDs []uuid.UUID) error
	UntagAsset(ctx context.Context, orgID, assetID uuid.UUID, tagIDs []uuid.UUID) error
}

// CollectionService manages collections.
type CollectionService interface {
	CreateCollection(ctx context.Context, orgID uuid.UUID, name, description string) (*domain.Collection, error)
	GetCollection(ctx context.Context, orgID, collectionID uuid.UUID) (*domain.Collection, error)
	UpdateCollection(ctx context.Context, orgID, collectionID uuid.UUID, name, description string) (*domain.Collection, error)
	DeleteCollection(ctx context.Context, orgID, collectionID uuid.UUID) error
	ListCollections(ctx context.Context, orgID uuid.UUID) ([]*domain.Collection, error)
	AddAsset(ctx context.Context, orgID, collectionID, assetID uuid.UUID) error
	RemoveAsset(ctx context.Context, orgID, collectionID, assetID uuid.UUID) error
	ListAssets(ctx context.Context, orgID, collectionID uuid.UUID, limit, offset int) ([]*domain.Asset, error)
}

// StyleService manages image styles and transform delivery.
type StyleService interface {
	CreateStyle(ctx context.Context, orgID uuid.UUID, req CreateStyleRequest) (*domain.ImageStyle, error)
	GetStyle(ctx context.Context, orgID uuid.UUID, styleSlug string) (*domain.ImageStyle, error)
	ListStyles(ctx context.Context, orgID uuid.UUID) ([]*domain.ImageStyle, error)
	UpdateStyle(ctx context.Context, orgID, styleID uuid.UUID, req CreateStyleRequest) (*domain.ImageStyle, error)
	DeleteStyle(ctx context.Context, orgID, styleID uuid.UUID) error
	// ServeStyled returns the transformed image bytes and content type.
	ServeStyled(ctx context.Context, orgID uuid.UUID, styleSlug, assetPath string) ([]byte, string, error)
	// ServeAdHoc returns transformed image bytes using ad-hoc params.
	ServeAdHoc(ctx context.Context, orgID uuid.UUID, assetPath string, params domain.AdHocParams) ([]byte, string, error)
}

type CreateStyleRequest struct {
	Name         string
	Slug         string
	Operations   []domain.StyleOperation
	OutputFormat domain.OutputFormat
	Quality      int
}

// DomainService manages custom domains.
type DomainService interface {
	AddDomain(ctx context.Context, orgID uuid.UUID, domainName string) (*domain.DomainRecord, error)
	ListDomains(ctx context.Context, orgID uuid.UUID) ([]*domain.DomainRecord, error)
	InitiateVerification(ctx context.Context, orgID, domainID uuid.UUID) (*domain.DomainRecord, error)
	VerifyDomain(ctx context.Context, orgID, domainID uuid.UUID) (*domain.DomainRecord, error)
	RemoveDomain(ctx context.Context, orgID, domainID uuid.UUID) error
	ResolveOrgByDomain(ctx context.Context, domainName string) (*domain.Org, error)
}

// WebhookService manages webhooks and deliveries.
type WebhookService interface {
	CreateWebhook(ctx context.Context, orgID uuid.UUID, url string, events []string) (*domain.Webhook, string, error)
	ListWebhooks(ctx context.Context, orgID uuid.UUID) ([]*domain.Webhook, error)
	GetWebhook(ctx context.Context, orgID, webhookID uuid.UUID) (*domain.Webhook, error)
	UpdateWebhook(ctx context.Context, orgID, webhookID uuid.UUID, url string, events []string, active bool) (*domain.Webhook, error)
	DeleteWebhook(ctx context.Context, orgID, webhookID uuid.UUID) error
	TestWebhook(ctx context.Context, orgID, webhookID uuid.UUID) error
	ListDeliveries(ctx context.Context, orgID, webhookID uuid.UUID, limit, offset int) ([]*domain.WebhookDelivery, error)
	DispatchEvent(ctx context.Context, event *domain.Event) error
}

// SearchService handles full-text and filtered search.
type SearchService interface {
	Search(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, int, error)
}
