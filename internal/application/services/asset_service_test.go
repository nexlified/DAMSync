package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nexlified/dam/domain"
	"github.com/nexlified/dam/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAssetSvc(
	orgRepo *testutil.MockOrgRepository,
	assetRepo *testutil.MockAssetRepository,
	storage *testutil.MockStoragePort,
) *AssetServiceImpl {
	return NewAssetService(
		assetRepo, orgRepo, storage,
		nil,
		nil,
		&testutil.MockAuditLogRepository{},
	)
}

// ── Upload ────────────────────────────────────────────────────────────────────

func TestUpload_QuotaExceeded(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &testutil.MockOrgRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Org, error) {
			org := testutil.NewOrg(id)
			org.StorageQuotaBytes = 100
			org.StorageUsedBytes = 90
			return org, nil
		},
	}
	svc := newAssetSvc(orgRepo, &testutil.MockAssetRepository{}, &testutil.MockStoragePort{})

	_, err := svc.Upload(context.Background(), orgID, nil, "file.jpg",
		bytes.NewReader([]byte("hello world!")), 12, "image/jpeg",
		domain.AssetMetadata{}, domain.VisibilityPublic)

	assert.ErrorIs(t, err, domain.ErrQuotaExceeded)
}

func TestUpload_FileTooLarge(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &testutil.MockOrgRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Org, error) {
			org := testutil.NewOrg(id)
			org.Settings.MaxFileSizeBytes = 10
			return org, nil
		},
	}
	svc := newAssetSvc(orgRepo, &testutil.MockAssetRepository{}, &testutil.MockStoragePort{})

	_, err := svc.Upload(context.Background(), orgID, nil, "file.jpg",
		bytes.NewReader([]byte("hello world!")), 12, "image/jpeg",
		domain.AssetMetadata{}, domain.VisibilityPublic)

	assert.ErrorIs(t, err, domain.ErrFileTooLarge)
}

func TestUpload_StorageError_ReturnsWrappedError(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &testutil.MockOrgRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Org, error) {
			return testutil.NewOrg(id), nil
		},
	}
	uploadCalled := false
	storage := &testutil.MockStoragePort{
		UploadFn: func(ctx context.Context, key string, r io.Reader, size int64, ct string) error {
			uploadCalled = true
			return errors.New("s3 unreachable")
		},
	}
	svc := newAssetSvc(orgRepo, &testutil.MockAssetRepository{}, storage)

	_, err := svc.Upload(context.Background(), orgID, nil, "file.jpg",
		bytes.NewReader([]byte("data")), 4, "image/jpeg",
		domain.AssetMetadata{}, domain.VisibilityPublic)

	require.Error(t, err)
	assert.True(t, uploadCalled)
	assert.Contains(t, err.Error(), "storage upload")
}

func TestUpload_Success_StorageKeyFormat(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &testutil.MockOrgRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Org, error) {
			return testutil.NewOrg(id), nil
		},
	}
	svc := newAssetSvc(orgRepo, &testutil.MockAssetRepository{}, &testutil.MockStoragePort{})

	asset, err := svc.Upload(context.Background(), orgID, nil, "photo.jpg",
		bytes.NewReader([]byte("imagedata")), 9, "image/jpeg",
		domain.AssetMetadata{}, domain.VisibilityPublic)

	require.NoError(t, err)
	assert.Equal(t, orgID, asset.OrgID)
	assert.True(t, strings.HasPrefix(asset.StorageKey, "orgs/"+orgID.String()+"/assets/"))
	assert.True(t, strings.HasSuffix(asset.StorageKey, ".jpg"))
}

func TestUpload_OrgNotFound(t *testing.T) {
	svc := newAssetSvc(&testutil.MockOrgRepository{}, &testutil.MockAssetRepository{}, &testutil.MockStoragePort{})

	_, err := svc.Upload(context.Background(), uuid.New(), nil, "file.jpg",
		bytes.NewReader([]byte("data")), 4, "image/jpeg",
		domain.AssetMetadata{}, domain.VisibilityPublic)

	assert.Error(t, err)
}

// ── GetAsset ─────────────────────────────────────────────────────────────────

func TestGetAsset_NotFound(t *testing.T) {
	svc := newAssetSvc(&testutil.MockOrgRepository{}, &testutil.MockAssetRepository{}, &testutil.MockStoragePort{})

	_, err := svc.GetAsset(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestGetAsset_WrongOrg_ReturnsNotFound(t *testing.T) {
	orgID := uuid.New()
	assetID := uuid.New()
	assetRepo := &testutil.MockAssetRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Asset, error) {
			return testutil.NewAsset(assetID, uuid.New()), nil // different org
		},
	}
	svc := newAssetSvc(&testutil.MockOrgRepository{}, assetRepo, &testutil.MockStoragePort{})

	_, err := svc.GetAsset(context.Background(), orgID, assetID)
	assert.ErrorIs(t, err, domain.ErrNotFound, "cross-org access must return ErrNotFound")
}

func TestGetAsset_SoftDeleted_ReturnsNotFound(t *testing.T) {
	orgID := uuid.New()
	assetID := uuid.New()
	deletedAt := time.Now()
	assetRepo := &testutil.MockAssetRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Asset, error) {
			a := testutil.NewAsset(assetID, orgID)
			a.DeletedAt = &deletedAt
			return a, nil
		},
	}
	svc := newAssetSvc(&testutil.MockOrgRepository{}, assetRepo, &testutil.MockStoragePort{})

	_, err := svc.GetAsset(context.Background(), orgID, assetID)
	assert.ErrorIs(t, err, domain.ErrNotFound, "soft-deleted asset must return ErrNotFound")
}

func TestGetAsset_Success(t *testing.T) {
	orgID := uuid.New()
	assetID := uuid.New()
	assetRepo := &testutil.MockAssetRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Asset, error) {
			return testutil.NewAsset(assetID, orgID), nil
		},
	}
	svc := newAssetSvc(&testutil.MockOrgRepository{}, assetRepo, &testutil.MockStoragePort{})

	asset, err := svc.GetAsset(context.Background(), orgID, assetID)
	require.NoError(t, err)
	assert.Equal(t, assetID, asset.ID)
}

// ── DeleteAsset ───────────────────────────────────────────────────────────────

func TestDeleteAsset_NotFound(t *testing.T) {
	svc := newAssetSvc(&testutil.MockOrgRepository{}, &testutil.MockAssetRepository{}, &testutil.MockStoragePort{})

	err := svc.DeleteAsset(context.Background(), uuid.New(), uuid.New())
	assert.ErrorIs(t, err, domain.ErrNotFound)
}

func TestDeleteAsset_CallsSoftDelete(t *testing.T) {
	orgID := uuid.New()
	assetID := uuid.New()
	softDeleted := false
	assetRepo := &testutil.MockAssetRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Asset, error) {
			return testutil.NewAsset(assetID, orgID), nil
		},
		SoftDeleteFn: func(_ context.Context, id uuid.UUID) error {
			softDeleted = true
			assert.Equal(t, assetID, id)
			return nil
		},
	}
	svc := newAssetSvc(&testutil.MockOrgRepository{}, assetRepo, &testutil.MockStoragePort{})

	require.NoError(t, svc.DeleteAsset(context.Background(), orgID, assetID))
	assert.True(t, softDeleted)
}

// ── GenerateSignedURL ─────────────────────────────────────────────────────────

func TestGenerateSignedURL_OrgNotFound(t *testing.T) {
	svc := newAssetSvc(&testutil.MockOrgRepository{}, &testutil.MockAssetRepository{}, &testutil.MockStoragePort{})

	_, err := svc.GenerateSignedURL(context.Background(), uuid.New(), uuid.New(), time.Hour)
	assert.Error(t, err)
}

func TestGenerateSignedURL_AssetNotFound(t *testing.T) {
	orgID := uuid.New()
	orgRepo := &testutil.MockOrgRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Org, error) {
			return testutil.NewOrg(id), nil
		},
	}
	svc := newAssetSvc(orgRepo, &testutil.MockAssetRepository{}, &testutil.MockStoragePort{})

	_, err := svc.GenerateSignedURL(context.Background(), orgID, uuid.New(), time.Hour)
	assert.Error(t, err)
}

func TestGenerateSignedURL_URLFormat(t *testing.T) {
	orgID := uuid.New()
	assetID := uuid.New()
	orgRepo := &testutil.MockOrgRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Org, error) {
			org := testutil.NewOrg(id)
			org.Settings.SignedURLSecret = "test-secret"
			return org, nil
		},
	}
	assetRepo := &testutil.MockAssetRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Asset, error) {
			return testutil.NewAsset(assetID, orgID), nil
		},
	}
	svc := newAssetSvc(orgRepo, assetRepo, &testutil.MockStoragePort{})

	url, err := svc.GenerateSignedURL(context.Background(), orgID, assetID, time.Hour)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(url, "/secure/"), "got: %s", url)
	parts := strings.SplitN(strings.TrimPrefix(url, "/secure/"), "/", 3)
	assert.Len(t, parts, 3, "URL should have token/expires/path segments")
}

// ── ValidateSignedURL ─────────────────────────────────────────────────────────

func TestValidateSignedURL_Expired(t *testing.T) {
	svc := newAssetSvc(&testutil.MockOrgRepository{}, &testutil.MockAssetRepository{}, &testutil.MockStoragePort{})

	past := time.Now().Add(-time.Hour).Unix()
	err := svc.ValidateSignedURL(context.Background(), uuid.New(), "anytoken", past)
	assert.ErrorIs(t, err, domain.ErrSignedURLExpired)
}

func TestValidateSignedURL_WrongToken(t *testing.T) {
	orgID := uuid.New()
	assetID := uuid.New()
	orgRepo := &testutil.MockOrgRepository{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*domain.Org, error) {
			org := testutil.NewOrg(id)
			org.Settings.SignedURLSecret = "test-secret"
			return org, nil
		},
	}
	assetRepo := &testutil.MockAssetRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Asset, error) {
			return testutil.NewAsset(assetID, orgID), nil
		},
	}
	svc := newAssetSvc(orgRepo, assetRepo, &testutil.MockStoragePort{})

	future := time.Now().Add(time.Hour).Unix()
	err := svc.ValidateSignedURL(context.Background(), assetID, "wrong-token", future)
	assert.ErrorIs(t, err, domain.ErrSignedURLInvalid)
}

// ── buildStorageKey ───────────────────────────────────────────────────────────

func TestBuildStorageKey(t *testing.T) {
	orgID := uuid.New()
	assetID := uuid.New()

	tests := []struct {
		filename string
		wantExt  string
	}{
		{"photo.jpg", ".jpg"},
		{"document.pdf", ".pdf"},
		{"noextension", ""},
		{"archive.tar.gz", ".gz"},
	}
	for _, tt := range tests {
		key := buildStorageKey(orgID, assetID, tt.filename)
		assert.True(t, strings.HasPrefix(key, "orgs/"+orgID.String()+"/assets/"), "key=%s", key)
		assert.True(t, strings.HasSuffix(key, tt.wantExt), "filename=%s wantExt=%s key=%s", tt.filename, tt.wantExt, key)
	}
}
