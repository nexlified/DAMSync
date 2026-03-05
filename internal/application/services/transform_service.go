package services

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/nexlified/dam/ports/inbound"
	"github.com/nexlified/dam/ports/outbound"
	"github.com/nexlified/dam/domain"
)

type StyleServiceImpl struct {
	styleRepo      outbound.StyleRepository
	transformCache outbound.TransformCacheRepository
	assetRepo      outbound.AssetRepository
	storage        outbound.StoragePort
	transformer    outbound.TransformerPort
	cache          outbound.CachePort
	publisher      outbound.EventPublisherPort
}

func NewStyleService(
	styleRepo outbound.StyleRepository,
	transformCache outbound.TransformCacheRepository,
	assetRepo outbound.AssetRepository,
	storage outbound.StoragePort,
	transformer outbound.TransformerPort,
	cache outbound.CachePort,
	publisher outbound.EventPublisherPort,
) *StyleServiceImpl {
	return &StyleServiceImpl{
		styleRepo:      styleRepo,
		transformCache: transformCache,
		assetRepo:      assetRepo,
		storage:        storage,
		transformer:    transformer,
		cache:          cache,
		publisher:      publisher,
	}
}

func (s *StyleServiceImpl) CreateStyle(ctx context.Context, orgID uuid.UUID, req inbound.CreateStyleRequest) (*domain.ImageStyle, error) {
	style := &domain.ImageStyle{
		ID:           uuid.New(),
		OrgID:        orgID,
		Name:         req.Name,
		Slug:         req.Slug,
		Operations:   req.Operations,
		OutputFormat: req.OutputFormat,
		Quality:      req.Quality,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	if style.Quality == 0 {
		style.Quality = 85
	}
	if err := s.styleRepo.Create(ctx, style); err != nil {
		return nil, err
	}
	return style, nil
}

func (s *StyleServiceImpl) GetStyle(ctx context.Context, orgID uuid.UUID, styleSlug string) (*domain.ImageStyle, error) {
	return s.styleRepo.GetBySlug(ctx, orgID, styleSlug)
}

func (s *StyleServiceImpl) ListStyles(ctx context.Context, orgID uuid.UUID) ([]*domain.ImageStyle, error) {
	return s.styleRepo.ListByOrg(ctx, orgID)
}

func (s *StyleServiceImpl) UpdateStyle(ctx context.Context, orgID, styleID uuid.UUID, req inbound.CreateStyleRequest) (*domain.ImageStyle, error) {
	style, err := s.styleRepo.GetByID(ctx, styleID)
	if err != nil {
		return nil, err
	}
	if style.OrgID != orgID {
		return nil, domain.ErrNotFound
	}
	style.Name = req.Name
	style.Operations = req.Operations
	style.OutputFormat = req.OutputFormat
	if req.Quality > 0 {
		style.Quality = req.Quality
	}
	style.UpdatedAt = time.Now().UTC()
	if err := s.styleRepo.Update(ctx, style); err != nil {
		return nil, err
	}
	// Invalidate cached transforms for this style
	_ = s.transformCache.DeleteByStyle(ctx, styleID)
	return style, nil
}

func (s *StyleServiceImpl) DeleteStyle(ctx context.Context, orgID, styleID uuid.UUID) error {
	style, err := s.styleRepo.GetByID(ctx, styleID)
	if err != nil {
		return err
	}
	if style.OrgID != orgID {
		return domain.ErrNotFound
	}
	_ = s.transformCache.DeleteByStyle(ctx, styleID)
	return s.styleRepo.Delete(ctx, styleID)
}

func (s *StyleServiceImpl) ServeStyled(ctx context.Context, orgID uuid.UUID, styleSlug, assetPath string) ([]byte, string, error) {
	style, err := s.styleRepo.GetBySlug(ctx, orgID, styleSlug)
	if err != nil {
		return nil, "", err
	}

	asset, err := s.assetRepo.GetByStorageKey(ctx, assetPath)
	if err != nil {
		return nil, "", err
	}
	if asset.OrgID != orgID {
		return nil, "", domain.ErrNotFound
	}

	paramsHash := hashStyleParams(style)
	cacheKey := fmt.Sprintf("transform:%s:%s", asset.ID, paramsHash)

	// Check Redis cache
	if cached, err := s.cache.GetBytes(ctx, cacheKey); err == nil && len(cached) > 0 {
		contentType := formatToMIME(string(style.OutputFormat))
		return cached, contentType, nil
	}

	// Check DB transform cache for storage key
	tc, err := s.transformCache.Get(ctx, asset.ID, paramsHash)
	if err == nil && tc != nil {
		// Fetch from storage
		rc, _, err := s.storage.Download(ctx, tc.StorageKey)
		if err == nil {
			defer rc.Close()
			var buf bytes.Buffer
			buf.ReadFrom(rc)
			data := buf.Bytes()
			// Warm Redis cache
			_ = s.cache.SetBytes(ctx, cacheKey, data, 24*time.Hour)
			return data, formatToMIME(tc.Format), nil
		}
	}

	// Apply transform
	data, contentType, err := s.applyStyleTransform(ctx, asset, style)
	if err != nil {
		return nil, "", err
	}

	// Store transformed file
	transformKey := buildTransformKey(orgID, asset.ID, style.ID, string(style.OutputFormat))
	_ = s.storage.Upload(ctx, transformKey, bytes.NewReader(data), int64(len(data)), contentType)

	// Save to DB cache
	_ = s.transformCache.Create(ctx, &domain.TransformCache{
		ID:         uuid.New(),
		AssetID:    asset.ID,
		StyleID:    &style.ID,
		ParamsHash: paramsHash,
		StorageKey: transformKey,
		SizeBytes:  int64(len(data)),
		Format:     string(style.OutputFormat),
		CreatedAt:  time.Now().UTC(),
	})

	// Cache in Redis
	_ = s.cache.SetBytes(ctx, cacheKey, data, 24*time.Hour)

	// Publish event
	if s.publisher != nil {
		event := domain.NewEvent(domain.EventAssetTransformed, orgID, &domain.AssetTransformedPayload{
			AssetID:    asset.ID,
			StyleID:    &style.ID,
			StorageKey: transformKey,
		})
		_ = s.publisher.Publish(ctx, event)
	}

	return data, contentType, nil
}

func (s *StyleServiceImpl) ServeAdHoc(ctx context.Context, orgID uuid.UUID, assetPath string, params domain.AdHocParams) ([]byte, string, error) {
	asset, err := s.assetRepo.GetByStorageKey(ctx, assetPath)
	if err != nil {
		return nil, "", err
	}
	if asset.OrgID != orgID {
		return nil, "", domain.ErrNotFound
	}

	paramsHash := hashAdHocParams(params)
	cacheKey := fmt.Sprintf("adhoc:%s:%s", asset.ID, paramsHash)

	if cached, err := s.cache.GetBytes(ctx, cacheKey); err == nil && len(cached) > 0 {
		return cached, formatToMIME(string(params.Format)), nil
	}

	// Download original
	rc, _, err := s.storage.Download(ctx, asset.StorageKey)
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()

	var origBuf bytes.Buffer
	origBuf.ReadFrom(rc)

	quality := 85
	if params.Quality != nil {
		quality = *params.Quality
	}

	req := &outbound.TransformRequest{
		Input:   origBuf.Bytes(),
		Width:   params.Width,
		Height:  params.Height,
		Fit:     params.Fit,
		Quality: &quality,
		Format:  params.Format,
	}
	result, err := s.transformer.Transform(ctx, req)
	if err != nil {
		return nil, "", err
	}

	_ = s.cache.SetBytes(ctx, cacheKey, result.Data, 24*time.Hour)
	return result.Data, result.ContentType, nil
}

// --- internal helpers ---

func (s *StyleServiceImpl) applyStyleTransform(ctx context.Context, asset *domain.Asset, style *domain.ImageStyle) ([]byte, string, error) {
	rc, _, err := s.storage.Download(ctx, asset.StorageKey)
	if err != nil {
		return nil, "", err
	}
	defer rc.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rc)

	req := &outbound.TransformRequest{
		Input:  buf.Bytes(),
		Format: style.OutputFormat,
	}

	// Apply first resize operation from style
	for _, op := range style.Operations {
		if op.Width != nil {
			req.Width = op.Width
		}
		if op.Height != nil {
			req.Height = op.Height
		}
		if op.Fit != "" {
			req.Fit = op.Fit
		}
		if op.Crop != "" {
			req.Crop = op.Crop
		}
		if op.Quality != nil {
			req.Quality = op.Quality
		}
		if op.Format != "" {
			req.Format = op.Format
		}
	}

	q := style.Quality
	if req.Quality == nil {
		req.Quality = &q
	}
	if req.Format == "" {
		req.Format = style.OutputFormat
	}
	if asset.FocalPoint != nil {
		req.FocalPoint = asset.FocalPoint
	}

	result, err := s.transformer.Transform(ctx, req)
	if err != nil {
		return nil, "", err
	}
	return result.Data, result.ContentType, nil
}

func hashStyleParams(style *domain.ImageStyle) string {
	data, _ := json.Marshal(struct {
		ID         string `json:"id"`
		Operations interface{} `json:"ops"`
		Format     string `json:"fmt"`
		Quality    int    `json:"q"`
	}{
		ID:         style.ID.String(),
		Operations: style.Operations,
		Format:     string(style.OutputFormat),
		Quality:    style.Quality,
	})
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8])
}

func hashAdHocParams(params domain.AdHocParams) string {
	data, _ := json.Marshal(params)
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8])
}

func buildTransformKey(orgID, assetID, styleID uuid.UUID, format string) string {
	return fmt.Sprintf("orgs/%s/transforms/%s/%s.%s", orgID, styleID, assetID, strings.ToLower(format))
}

func formatToMIME(format string) string {
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "webp":
		return "image/webp"
	case "avif":
		return "image/avif"
	default:
		return "application/octet-stream"
	}
}
