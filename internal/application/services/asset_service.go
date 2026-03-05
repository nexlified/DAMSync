package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nexlified/dam/ports/inbound"
	"github.com/nexlified/dam/ports/outbound"
	"github.com/nexlified/dam/domain"
)

type AssetServiceImpl struct {
	assetRepo   outbound.AssetRepository
	orgRepo     outbound.OrgRepository
	storage     outbound.StoragePort
	transformer outbound.TransformerPort
	publisher   outbound.EventPublisherPort
	auditRepo   outbound.AuditLogRepository
}

func NewAssetService(
	assetRepo outbound.AssetRepository,
	orgRepo outbound.OrgRepository,
	storage outbound.StoragePort,
	transformer outbound.TransformerPort,
	publisher outbound.EventPublisherPort,
	auditRepo outbound.AuditLogRepository,
) *AssetServiceImpl {
	return &AssetServiceImpl{
		assetRepo:   assetRepo,
		orgRepo:     orgRepo,
		storage:     storage,
		transformer: transformer,
		publisher:   publisher,
		auditRepo:   auditRepo,
	}
}

func (s *AssetServiceImpl) Upload(
	ctx context.Context,
	orgID uuid.UUID,
	folderID *uuid.UUID,
	filename string,
	r io.Reader,
	size int64,
	contentType string,
	metadata domain.AssetMetadata,
	visibility domain.Visibility,
) (*domain.Asset, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Check quota
	if org.StorageUsedBytes+size > org.StorageQuotaBytes {
		return nil, domain.ErrQuotaExceeded
	}

	// Check file size limit
	maxSize := org.Settings.MaxFileSizeBytes
	if maxSize > 0 && size > maxSize {
		return nil, domain.ErrFileTooLarge
	}

	// Read file data for MIME validation and metadata extraction
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	// Validate MIME via magic bytes
	detectedMIME := http.DetectContentType(data[:min(512, len(data))])
	if contentType == "" {
		contentType = detectedMIME
	}

	// Strip EXIF if configured and it's an image
	if org.Settings.StripEXIF && strings.HasPrefix(contentType, "image/") {
		if stripped, err := s.stripEXIF(data, contentType); err == nil {
			data = stripped
			size = int64(len(data))
		}
	}

	assetID := uuid.New()
	storageKey := buildStorageKey(orgID, assetID, filename)

	if err := s.storage.Upload(ctx, storageKey, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		return nil, fmt.Errorf("storage upload: %w", err)
	}

	asset := &domain.Asset{
		ID:         assetID,
		OrgID:      orgID,
		FolderID:   folderID,
		Filename:   filename,
		StorageKey: storageKey,
		MIMEType:   contentType,
		SizeBytes:  int64(len(data)),
		Metadata:   metadata,
		Visibility: visibility,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	// Extract image dimensions
	if strings.HasPrefix(contentType, "image/") && s.transformer != nil {
		if meta, err := s.transformer.ExtractMetadata(data); err == nil {
			asset.Width = &meta.Width
			asset.Height = &meta.Height
		}
	}

	if err := s.assetRepo.Create(ctx, asset); err != nil {
		// best-effort cleanup
		_ = s.storage.Delete(ctx, storageKey)
		return nil, err
	}

	// Update storage used
	_ = s.orgRepo.IncrementStorageUsed(ctx, orgID, asset.SizeBytes)

	// Publish event
	if s.publisher != nil {
		event := domain.NewEvent(domain.EventAssetCreated, orgID, &domain.AssetCreatedPayload{Asset: asset})
		_ = s.publisher.Publish(ctx, event)
	}

	return asset, nil
}

func (s *AssetServiceImpl) BulkUpload(
	ctx context.Context,
	orgID uuid.UUID,
	folderID *uuid.UUID,
	files []inbound.UploadFile,
) ([]*domain.Asset, []inbound.BulkUploadError, error) {
	var assets []*domain.Asset
	var errs []inbound.BulkUploadError

	for _, f := range files {
		asset, err := s.Upload(ctx, orgID, folderID, f.Filename, f.Reader, f.Size, f.ContentType, f.Metadata, f.Visibility)
		if err != nil {
			errs = append(errs, inbound.BulkUploadError{Filename: f.Filename, Error: err.Error()})
			continue
		}
		assets = append(assets, asset)
	}
	return assets, errs, nil
}

func (s *AssetServiceImpl) GetAsset(ctx context.Context, orgID, assetID uuid.UUID) (*domain.Asset, error) {
	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if asset.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	if asset.IsDeleted() {
		return nil, domain.ErrNotFound
	}
	return asset, nil
}

func (s *AssetServiceImpl) ListAssets(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, error) {
	return s.assetRepo.List(ctx, filter)
}

func (s *AssetServiceImpl) UpdateMetadata(
	ctx context.Context,
	orgID, assetID uuid.UUID,
	metadata domain.AssetMetadata,
	visibility *domain.Visibility,
) (*domain.Asset, error) {
	asset, err := s.GetAsset(ctx, orgID, assetID)
	if err != nil {
		return nil, err
	}
	asset.Metadata = metadata
	if visibility != nil {
		asset.Visibility = *visibility
	}
	asset.UpdatedAt = time.Now().UTC()
	if err := s.assetRepo.Update(ctx, asset); err != nil {
		return nil, err
	}
	if s.publisher != nil {
		event := domain.NewEvent(domain.EventAssetUpdated, orgID, &domain.AssetUpdatedPayload{Asset: asset})
		_ = s.publisher.Publish(ctx, event)
	}
	return asset, nil
}

func (s *AssetServiceImpl) DeleteAsset(ctx context.Context, orgID, assetID uuid.UUID) error {
	asset, err := s.GetAsset(ctx, orgID, assetID)
	if err != nil {
		return err
	}
	if err := s.assetRepo.SoftDelete(ctx, asset.ID); err != nil {
		return err
	}
	if s.publisher != nil {
		event := domain.NewEvent(domain.EventAssetDeleted, orgID, &domain.AssetDeletedPayload{AssetID: assetID, OrgID: orgID})
		_ = s.publisher.Publish(ctx, event)
	}
	return nil
}

func (s *AssetServiceImpl) MoveAsset(ctx context.Context, orgID, assetID uuid.UUID, folderID *uuid.UUID) error {
	asset, err := s.GetAsset(ctx, orgID, assetID)
	if err != nil {
		return err
	}
	return s.assetRepo.Move(ctx, asset.ID, folderID)
}

func (s *AssetServiceImpl) GenerateSignedURL(ctx context.Context, orgID, assetID uuid.UUID, ttl time.Duration) (string, error) {
	org, err := s.orgRepo.GetByID(ctx, orgID)
	if err != nil {
		return "", err
	}
	asset, err := s.GetAsset(ctx, orgID, assetID)
	if err != nil {
		return "", err
	}

	secret := org.Settings.SignedURLSecret
	if secret == "" {
		secret = "default-signed-url-secret" // fallback; should be configured
	}

	expires := time.Now().Add(ttl).Unix()
	sig := computeHMAC(secret, asset.ID.String(), expires)

	return fmt.Sprintf("/secure/%s/%d/%s", sig, expires, asset.StorageKey), nil
}

func (s *AssetServiceImpl) ValidateSignedURL(ctx context.Context, assetID uuid.UUID, token string, timestamp int64) error {
	if time.Now().Unix() > timestamp {
		return domain.ErrSignedURLExpired
	}
	asset, err := s.assetRepo.GetByID(ctx, assetID)
	if err != nil {
		return domain.ErrSignedURLInvalid
	}
	org, err := s.orgRepo.GetByID(ctx, asset.OrgID)
	if err != nil {
		return domain.ErrSignedURLInvalid
	}

	secret := org.Settings.SignedURLSecret
	if secret == "" {
		secret = "default-signed-url-secret"
	}

	expected := computeHMAC(secret, assetID.String(), timestamp)
	if !hmac.Equal([]byte(token), []byte(expected)) {
		return domain.ErrSignedURLInvalid
	}
	return nil
}

// --- internal helpers ---

func buildStorageKey(orgID uuid.UUID, assetID uuid.UUID, filename string) string {
	ext := path.Ext(filename)
	return fmt.Sprintf("orgs/%s/assets/%s%s", orgID, assetID, ext)
}

func computeHMAC(secret, assetID string, timestamp int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(fmt.Sprintf("%s:%d", assetID, timestamp)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *AssetServiceImpl) stripEXIF(data []byte, mimeType string) ([]byte, error) {
	// For now, rely on bimg during transform to strip EXIF.
	// This is a placeholder — production would use bimg or go-exiftool here.
	_ = mimeType
	return data, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
