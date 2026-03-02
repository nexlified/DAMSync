// Package testutil provides mock implementations of all outbound ports and
// inbound service interfaces for use in unit and handler tests.
package testutil

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/nexlified/dam/internal/application/ports/inbound"
	"github.com/nexlified/dam/internal/application/ports/outbound"
	"github.com/nexlified/dam/internal/domain"
)

// ── Outbound repository mocks ────────────────────────────────────────────────

type MockAssetRepository struct {
	CreateFn          func(ctx context.Context, asset *domain.Asset) error
	GetByIDFn         func(ctx context.Context, id uuid.UUID) (*domain.Asset, error)
	GetByStorageKeyFn func(ctx context.Context, key string) (*domain.Asset, error)
	UpdateFn          func(ctx context.Context, asset *domain.Asset) error
	SoftDeleteFn      func(ctx context.Context, id uuid.UUID) error
	HardDeleteFn      func(ctx context.Context, id uuid.UUID) error
	ListFn            func(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, error)
	CountFn           func(ctx context.Context, filter domain.AssetListFilter) (int, error)
	AddTagsFn         func(ctx context.Context, assetID uuid.UUID, tagIDs []uuid.UUID) error
	RemoveTagsFn      func(ctx context.Context, assetID uuid.UUID, tagIDs []uuid.UUID) error
	GetTagsFn         func(ctx context.Context, assetID uuid.UUID) ([]domain.Tag, error)
	MoveFn            func(ctx context.Context, assetID uuid.UUID, folderID *uuid.UUID) error
}

func (m *MockAssetRepository) Create(ctx context.Context, a *domain.Asset) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, a)
	}
	return nil
}
func (m *MockAssetRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Asset, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}
func (m *MockAssetRepository) GetByStorageKey(ctx context.Context, key string) (*domain.Asset, error) {
	if m.GetByStorageKeyFn != nil {
		return m.GetByStorageKeyFn(ctx, key)
	}
	return nil, domain.ErrNotFound
}
func (m *MockAssetRepository) Update(ctx context.Context, a *domain.Asset) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, a)
	}
	return nil
}
func (m *MockAssetRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if m.SoftDeleteFn != nil {
		return m.SoftDeleteFn(ctx, id)
	}
	return nil
}
func (m *MockAssetRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	if m.HardDeleteFn != nil {
		return m.HardDeleteFn(ctx, id)
	}
	return nil
}
func (m *MockAssetRepository) List(ctx context.Context, f domain.AssetListFilter) ([]*domain.Asset, string, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, f)
	}
	return nil, "", nil
}
func (m *MockAssetRepository) Count(ctx context.Context, f domain.AssetListFilter) (int, error) {
	if m.CountFn != nil {
		return m.CountFn(ctx, f)
	}
	return 0, nil
}
func (m *MockAssetRepository) AddTags(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) error {
	if m.AddTagsFn != nil {
		return m.AddTagsFn(ctx, id, tagIDs)
	}
	return nil
}
func (m *MockAssetRepository) RemoveTags(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) error {
	if m.RemoveTagsFn != nil {
		return m.RemoveTagsFn(ctx, id, tagIDs)
	}
	return nil
}
func (m *MockAssetRepository) GetTags(ctx context.Context, id uuid.UUID) ([]domain.Tag, error) {
	if m.GetTagsFn != nil {
		return m.GetTagsFn(ctx, id)
	}
	return nil, nil
}
func (m *MockAssetRepository) Move(ctx context.Context, assetID uuid.UUID, folderID *uuid.UUID) error {
	if m.MoveFn != nil {
		return m.MoveFn(ctx, assetID, folderID)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────

type MockOrgRepository struct {
	GetByIDFn              func(ctx context.Context, id uuid.UUID) (*domain.Org, error)
	GetBySlugFn            func(ctx context.Context, slug string) (*domain.Org, error)
	CreateFn               func(ctx context.Context, org *domain.Org) error
	UpdateFn               func(ctx context.Context, org *domain.Org) error
	DeleteFn               func(ctx context.Context, id uuid.UUID) error
	IncrementStorageUsedFn func(ctx context.Context, orgID uuid.UUID, delta int64) error
	ListFn                 func(ctx context.Context, limit, offset int) ([]*domain.Org, int, error)
}

func (m *MockOrgRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Org, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}
func (m *MockOrgRepository) GetBySlug(ctx context.Context, slug string) (*domain.Org, error) {
	if m.GetBySlugFn != nil {
		return m.GetBySlugFn(ctx, slug)
	}
	return nil, domain.ErrNotFound
}
func (m *MockOrgRepository) Create(ctx context.Context, org *domain.Org) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, org)
	}
	return nil
}
func (m *MockOrgRepository) Update(ctx context.Context, org *domain.Org) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, org)
	}
	return nil
}
func (m *MockOrgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
func (m *MockOrgRepository) IncrementStorageUsed(ctx context.Context, orgID uuid.UUID, delta int64) error {
	if m.IncrementStorageUsedFn != nil {
		return m.IncrementStorageUsedFn(ctx, orgID, delta)
	}
	return nil
}
func (m *MockOrgRepository) List(ctx context.Context, limit, offset int) ([]*domain.Org, int, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, limit, offset)
	}
	return nil, 0, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type MockFolderRepository struct {
	CreateFn       func(ctx context.Context, f *domain.Folder) error
	GetByIDFn      func(ctx context.Context, id uuid.UUID) (*domain.Folder, error)
	GetByPathFn    func(ctx context.Context, orgID uuid.UUID, path string) (*domain.Folder, error)
	UpdateFn       func(ctx context.Context, f *domain.Folder) error
	DeleteFn       func(ctx context.Context, id uuid.UUID) error
	ListByOrgFn    func(ctx context.Context, orgID uuid.UUID) ([]*domain.Folder, error)
	ListChildrenFn func(ctx context.Context, parentID uuid.UUID) ([]*domain.Folder, error)
}

func (m *MockFolderRepository) Create(ctx context.Context, f *domain.Folder) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, f)
	}
	return nil
}
func (m *MockFolderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Folder, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}
func (m *MockFolderRepository) GetByPath(ctx context.Context, orgID uuid.UUID, path string) (*domain.Folder, error) {
	if m.GetByPathFn != nil {
		return m.GetByPathFn(ctx, orgID, path)
	}
	return nil, domain.ErrNotFound
}
func (m *MockFolderRepository) Update(ctx context.Context, f *domain.Folder) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, f)
	}
	return nil
}
func (m *MockFolderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
func (m *MockFolderRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Folder, error) {
	if m.ListByOrgFn != nil {
		return m.ListByOrgFn(ctx, orgID)
	}
	return nil, nil
}
func (m *MockFolderRepository) ListChildren(ctx context.Context, parentID uuid.UUID) ([]*domain.Folder, error) {
	if m.ListChildrenFn != nil {
		return m.ListChildrenFn(ctx, parentID)
	}
	return nil, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type MockUserRepository struct {
	CreateFn     func(ctx context.Context, u *domain.User) error
	GetByIDFn    func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmailFn func(ctx context.Context, orgID uuid.UUID, email string) (*domain.User, error)
	UpdateFn     func(ctx context.Context, u *domain.User) error
	DeleteFn     func(ctx context.Context, id uuid.UUID) error
	ListByOrgFn  func(ctx context.Context, orgID uuid.UUID) ([]*domain.User, error)
}

func (m *MockUserRepository) Create(ctx context.Context, u *domain.User) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, u)
	}
	return nil
}
func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.GetByIDFn != nil {
		return m.GetByIDFn(ctx, id)
	}
	return nil, domain.ErrNotFound
}
func (m *MockUserRepository) GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*domain.User, error) {
	if m.GetByEmailFn != nil {
		return m.GetByEmailFn(ctx, orgID, email)
	}
	return nil, domain.ErrNotFound
}
func (m *MockUserRepository) Update(ctx context.Context, u *domain.User) error {
	if m.UpdateFn != nil {
		return m.UpdateFn(ctx, u)
	}
	return nil
}
func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, id)
	}
	return nil
}
func (m *MockUserRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.User, error) {
	if m.ListByOrgFn != nil {
		return m.ListByOrgFn(ctx, orgID)
	}
	return nil, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type MockAPIKeyRepository struct {
	CreateFn         func(ctx context.Context, k *domain.APIKey) error
	GetByPrefixFn    func(ctx context.Context, prefix string) (*domain.APIKey, error)
	ListByOrgFn      func(ctx context.Context, orgID uuid.UUID) ([]*domain.APIKey, error)
	RevokeFn         func(ctx context.Context, id uuid.UUID) error
	UpdateLastUsedFn func(ctx context.Context, id uuid.UUID) error
}

func (m *MockAPIKeyRepository) Create(ctx context.Context, k *domain.APIKey) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, k)
	}
	return nil
}
func (m *MockAPIKeyRepository) GetByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error) {
	if m.GetByPrefixFn != nil {
		return m.GetByPrefixFn(ctx, prefix)
	}
	return nil, domain.ErrNotFound
}
func (m *MockAPIKeyRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.APIKey, error) {
	if m.ListByOrgFn != nil {
		return m.ListByOrgFn(ctx, orgID)
	}
	return nil, nil
}
func (m *MockAPIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	if m.RevokeFn != nil {
		return m.RevokeFn(ctx, id)
	}
	return nil
}
func (m *MockAPIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	if m.UpdateLastUsedFn != nil {
		return m.UpdateLastUsedFn(ctx, id)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────

type MockStoragePort struct {
	UploadFn    func(ctx context.Context, key string, r io.Reader, size int64, ct string) error
	DownloadFn  func(ctx context.Context, key string) (io.ReadCloser, int64, error)
	DeleteFn    func(ctx context.Context, key string) error
	ExistsFn    func(ctx context.Context, key string) (bool, error)
	SignedURLFn func(ctx context.Context, key string, ttl time.Duration) (string, error)
	PublicURLFn func(key string) string
	CopyFn      func(ctx context.Context, src, dst string) error
}

func (m *MockStoragePort) Upload(ctx context.Context, key string, r io.Reader, size int64, ct string) error {
	if m.UploadFn != nil {
		return m.UploadFn(ctx, key, r, size, ct)
	}
	return nil
}
func (m *MockStoragePort) Download(ctx context.Context, key string) (io.ReadCloser, int64, error) {
	if m.DownloadFn != nil {
		return m.DownloadFn(ctx, key)
	}
	return nil, 0, nil
}
func (m *MockStoragePort) Delete(ctx context.Context, key string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, key)
	}
	return nil
}
func (m *MockStoragePort) Exists(ctx context.Context, key string) (bool, error) {
	if m.ExistsFn != nil {
		return m.ExistsFn(ctx, key)
	}
	return false, nil
}
func (m *MockStoragePort) SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	if m.SignedURLFn != nil {
		return m.SignedURLFn(ctx, key, ttl)
	}
	return "", nil
}
func (m *MockStoragePort) PublicURL(key string) string {
	if m.PublicURLFn != nil {
		return m.PublicURLFn(key)
	}
	return "/files/" + key
}
func (m *MockStoragePort) Copy(ctx context.Context, src, dst string) error {
	if m.CopyFn != nil {
		return m.CopyFn(ctx, src, dst)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────

type MockTransformerPort struct {
	TransformFn       func(ctx context.Context, req *outbound.TransformRequest) (*outbound.TransformResult, error)
	ExtractMetadataFn func(data []byte) (*outbound.ImageMetadata, error)
}

func (m *MockTransformerPort) Transform(ctx context.Context, req *outbound.TransformRequest) (*outbound.TransformResult, error) {
	if m.TransformFn != nil {
		return m.TransformFn(ctx, req)
	}
	return &outbound.TransformResult{}, nil
}
func (m *MockTransformerPort) ExtractMetadata(data []byte) (*outbound.ImageMetadata, error) {
	if m.ExtractMetadataFn != nil {
		return m.ExtractMetadataFn(data)
	}
	return &outbound.ImageMetadata{Width: 100, Height: 100}, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type MockEventPublisher struct {
	PublishFn func(ctx context.Context, event *domain.Event) error
}

func (m *MockEventPublisher) Publish(ctx context.Context, event *domain.Event) error {
	if m.PublishFn != nil {
		return m.PublishFn(ctx, event)
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────

type MockAuditLogRepository struct {
	CreateFn func(ctx context.Context, log *domain.AuditLog) error
	ListFn   func(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error)
}

func (m *MockAuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	if m.CreateFn != nil {
		return m.CreateFn(ctx, log)
	}
	return nil
}
func (m *MockAuditLogRepository) List(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error) {
	if m.ListFn != nil {
		return m.ListFn(ctx, orgID, limit, offset)
	}
	return nil, nil
}

// ── Inbound service mocks (for handler tests) ────────────────────────────────

type MockAssetService struct {
	UploadFn            func(ctx context.Context, orgID uuid.UUID, folderID *uuid.UUID, filename string, r io.Reader, size int64, ct string, meta domain.AssetMetadata, vis domain.Visibility) (*domain.Asset, error)
	ListAssetsFn        func(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, error)
	GetAssetFn          func(ctx context.Context, orgID, assetID uuid.UUID) (*domain.Asset, error)
	UpdateMetadataFn    func(ctx context.Context, orgID, assetID uuid.UUID, meta domain.AssetMetadata, vis *domain.Visibility) (*domain.Asset, error)
	DeleteAssetFn       func(ctx context.Context, orgID, assetID uuid.UUID) error
	MoveAssetFn         func(ctx context.Context, orgID, assetID uuid.UUID, folderID *uuid.UUID) error
	GenerateSignedURLFn func(ctx context.Context, orgID, assetID uuid.UUID, ttl time.Duration) (string, error)
	ValidateSignedURLFn func(ctx context.Context, assetID uuid.UUID, token string, ts int64) error
	BulkUploadFn        func(ctx context.Context, orgID uuid.UUID, folderID *uuid.UUID, files []inbound.UploadFile) ([]*domain.Asset, []inbound.BulkUploadError, error)
}

func (m *MockAssetService) Upload(ctx context.Context, orgID uuid.UUID, folderID *uuid.UUID, filename string, r io.Reader, size int64, ct string, meta domain.AssetMetadata, vis domain.Visibility) (*domain.Asset, error) {
	if m.UploadFn != nil {
		return m.UploadFn(ctx, orgID, folderID, filename, r, size, ct, meta, vis)
	}
	return &domain.Asset{ID: uuid.New(), OrgID: orgID, Filename: filename}, nil
}
func (m *MockAssetService) ListAssets(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, error) {
	if m.ListAssetsFn != nil {
		return m.ListAssetsFn(ctx, filter)
	}
	return nil, "", nil
}
func (m *MockAssetService) GetAsset(ctx context.Context, orgID, assetID uuid.UUID) (*domain.Asset, error) {
	if m.GetAssetFn != nil {
		return m.GetAssetFn(ctx, orgID, assetID)
	}
	return nil, domain.ErrNotFound
}
func (m *MockAssetService) UpdateMetadata(ctx context.Context, orgID, assetID uuid.UUID, meta domain.AssetMetadata, vis *domain.Visibility) (*domain.Asset, error) {
	if m.UpdateMetadataFn != nil {
		return m.UpdateMetadataFn(ctx, orgID, assetID, meta, vis)
	}
	return nil, domain.ErrNotFound
}
func (m *MockAssetService) DeleteAsset(ctx context.Context, orgID, assetID uuid.UUID) error {
	if m.DeleteAssetFn != nil {
		return m.DeleteAssetFn(ctx, orgID, assetID)
	}
	return nil
}
func (m *MockAssetService) MoveAsset(ctx context.Context, orgID, assetID uuid.UUID, folderID *uuid.UUID) error {
	if m.MoveAssetFn != nil {
		return m.MoveAssetFn(ctx, orgID, assetID, folderID)
	}
	return nil
}
func (m *MockAssetService) GenerateSignedURL(ctx context.Context, orgID, assetID uuid.UUID, ttl time.Duration) (string, error) {
	if m.GenerateSignedURLFn != nil {
		return m.GenerateSignedURLFn(ctx, orgID, assetID, ttl)
	}
	return "/secure/token/123/key", nil
}
func (m *MockAssetService) ValidateSignedURL(ctx context.Context, assetID uuid.UUID, token string, ts int64) error {
	if m.ValidateSignedURLFn != nil {
		return m.ValidateSignedURLFn(ctx, assetID, token, ts)
	}
	return nil
}
func (m *MockAssetService) BulkUpload(ctx context.Context, orgID uuid.UUID, folderID *uuid.UUID, files []inbound.UploadFile) ([]*domain.Asset, []inbound.BulkUploadError, error) {
	if m.BulkUploadFn != nil {
		return m.BulkUploadFn(ctx, orgID, folderID, files)
	}
	return nil, nil, nil
}

// ─────────────────────────────────────────────────────────────────────────────

type MockFolderService struct {
	CreateFolderFn  func(ctx context.Context, orgID uuid.UUID, parentID *uuid.UUID, name string) (*domain.Folder, error)
	GetFolderFn     func(ctx context.Context, orgID, folderID uuid.UUID) (*domain.Folder, error)
	UpdateFolderFn  func(ctx context.Context, orgID, folderID uuid.UUID, name string) (*domain.Folder, error)
	DeleteFolderFn  func(ctx context.Context, orgID, folderID uuid.UUID) error
	ListFoldersFn   func(ctx context.Context, orgID uuid.UUID) ([]*domain.Folder, error)
	GetFolderTreeFn func(ctx context.Context, orgID uuid.UUID) ([]*inbound.FolderNode, error)
}

func (m *MockFolderService) CreateFolder(ctx context.Context, orgID uuid.UUID, parentID *uuid.UUID, name string) (*domain.Folder, error) {
	if m.CreateFolderFn != nil {
		return m.CreateFolderFn(ctx, orgID, parentID, name)
	}
	return &domain.Folder{ID: uuid.New(), OrgID: orgID, Name: name}, nil
}
func (m *MockFolderService) GetFolder(ctx context.Context, orgID, folderID uuid.UUID) (*domain.Folder, error) {
	if m.GetFolderFn != nil {
		return m.GetFolderFn(ctx, orgID, folderID)
	}
	return nil, domain.ErrNotFound
}
func (m *MockFolderService) UpdateFolder(ctx context.Context, orgID, folderID uuid.UUID, name string) (*domain.Folder, error) {
	if m.UpdateFolderFn != nil {
		return m.UpdateFolderFn(ctx, orgID, folderID, name)
	}
	return nil, domain.ErrNotFound
}
func (m *MockFolderService) DeleteFolder(ctx context.Context, orgID, folderID uuid.UUID) error {
	if m.DeleteFolderFn != nil {
		return m.DeleteFolderFn(ctx, orgID, folderID)
	}
	return nil
}
func (m *MockFolderService) ListFolders(ctx context.Context, orgID uuid.UUID) ([]*domain.Folder, error) {
	if m.ListFoldersFn != nil {
		return m.ListFoldersFn(ctx, orgID)
	}
	return nil, nil
}
func (m *MockFolderService) GetFolderTree(ctx context.Context, orgID uuid.UUID) ([]*inbound.FolderNode, error) {
	if m.GetFolderTreeFn != nil {
		return m.GetFolderTreeFn(ctx, orgID)
	}
	return nil, nil
}

// ── Test fixtures ─────────────────────────────────────────────────────────────

// NewOrg builds a minimal domain.Org suitable for tests.
func NewOrg(id uuid.UUID) *domain.Org {
	return &domain.Org{
		ID:                id,
		Name:              "Test Org",
		Slug:              "test-org",
		StorageQuotaBytes: 10 * 1024 * 1024 * 1024,
		StorageUsedBytes:  0,
		Settings:          domain.OrgSettings{},
	}
}

// NewAsset builds a minimal domain.Asset suitable for tests.
func NewAsset(id, orgID uuid.UUID) *domain.Asset {
	return &domain.Asset{
		ID:         id,
		OrgID:      orgID,
		Filename:   "test.jpg",
		StorageKey: "orgs/" + orgID.String() + "/assets/" + id.String() + ".jpg",
		MIMEType:   "image/jpeg",
		SizeBytes:  1024,
		Visibility: domain.VisibilityPublic,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}
}

// NewFolder builds a minimal domain.Folder suitable for tests.
func NewFolder(id, orgID uuid.UUID, name string) *domain.Folder {
	return &domain.Folder{
		ID:        id,
		OrgID:     orgID,
		Name:      name,
		Path:      "/" + name,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
}
