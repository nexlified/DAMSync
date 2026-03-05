package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nexlified/dam/domain"
)

// FolderRepository

type FolderRepository struct{ db *sqlx.DB }

func NewFolderRepository(db *sqlx.DB) *FolderRepository { return &FolderRepository{db: db} }

func (r *FolderRepository) Create(ctx context.Context, f *domain.Folder) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO folders (id, org_id, parent_id, name, path, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		f.ID, f.OrgID, f.ParentID, f.Name, f.Path, f.CreatedAt, f.UpdatedAt,
	)
	return err
}

func (r *FolderRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Folder, error) {
	var f domain.Folder
	if err := r.db.GetContext(ctx, &f, `SELECT * FROM folders WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *FolderRepository) GetByPath(ctx context.Context, orgID uuid.UUID, path string) (*domain.Folder, error) {
	var f domain.Folder
	if err := r.db.GetContext(ctx, &f, `SELECT * FROM folders WHERE org_id=$1 AND path=$2`, orgID, path); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &f, nil
}

func (r *FolderRepository) Update(ctx context.Context, f *domain.Folder) error {
	_, err := r.db.ExecContext(ctx, `UPDATE folders SET name=$1, updated_at=$2 WHERE id=$3`, f.Name, f.UpdatedAt, f.ID)
	return err
}

func (r *FolderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM folders WHERE id=$1`, id)
	return err
}

func (r *FolderRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Folder, error) {
	var folders []*domain.Folder
	err := r.db.SelectContext(ctx, &folders, `SELECT * FROM folders WHERE org_id=$1 ORDER BY path`, orgID)
	return folders, err
}

func (r *FolderRepository) ListChildren(ctx context.Context, parentID uuid.UUID) ([]*domain.Folder, error) {
	var folders []*domain.Folder
	err := r.db.SelectContext(ctx, &folders, `SELECT * FROM folders WHERE parent_id=$1 ORDER BY name`, parentID)
	return folders, err
}

// TagRepository

type TagRepository struct{ db *sqlx.DB }

func NewTagRepository(db *sqlx.DB) *TagRepository { return &TagRepository{db: db} }

func (r *TagRepository) Create(ctx context.Context, t *domain.Tag) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO tags (id, org_id, name, slug, created_at) VALUES ($1,$2,$3,$4,$5)`,
		t.ID, t.OrgID, t.Name, t.Slug, t.CreatedAt,
	)
	return err
}

func (r *TagRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tag, error) {
	var t domain.Tag
	if err := r.db.GetContext(ctx, &t, `SELECT * FROM tags WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *TagRepository) GetBySlug(ctx context.Context, orgID uuid.UUID, slug string) (*domain.Tag, error) {
	var t domain.Tag
	if err := r.db.GetContext(ctx, &t, `SELECT * FROM tags WHERE org_id=$1 AND slug=$2`, orgID, slug); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &t, nil
}

func (r *TagRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Tag, error) {
	var tags []*domain.Tag
	err := r.db.SelectContext(ctx, &tags, `SELECT * FROM tags WHERE org_id=$1 ORDER BY name`, orgID)
	return tags, err
}

func (r *TagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tags WHERE id=$1`, id)
	return err
}

// CollectionRepository

type CollectionRepository struct{ db *sqlx.DB }

func NewCollectionRepository(db *sqlx.DB) *CollectionRepository {
	return &CollectionRepository{db: db}
}

func (r *CollectionRepository) Create(ctx context.Context, c *domain.Collection) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO collections (id, org_id, name, description, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6)`,
		c.ID, c.OrgID, c.Name, c.Description, c.CreatedAt, c.UpdatedAt,
	)
	return err
}

func (r *CollectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Collection, error) {
	var c domain.Collection
	if err := r.db.GetContext(ctx, &c, `SELECT * FROM collections WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *CollectionRepository) Update(ctx context.Context, c *domain.Collection) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE collections SET name=$1, description=$2, updated_at=$3 WHERE id=$4`,
		c.Name, c.Description, c.UpdatedAt, c.ID,
	)
	return err
}

func (r *CollectionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM collections WHERE id=$1`, id)
	return err
}

func (r *CollectionRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Collection, error) {
	var cols []*domain.Collection
	err := r.db.SelectContext(ctx, &cols, `SELECT * FROM collections WHERE org_id=$1 ORDER BY name`, orgID)
	return cols, err
}

func (r *CollectionRepository) AddAsset(ctx context.Context, collectionID, assetID uuid.UUID, sortOrder int) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO collection_assets (collection_id, asset_id, sort_order) VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
		collectionID, assetID, sortOrder,
	)
	return err
}

func (r *CollectionRepository) RemoveAsset(ctx context.Context, collectionID, assetID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM collection_assets WHERE collection_id=$1 AND asset_id=$2`,
		collectionID, assetID,
	)
	return err
}

func (r *CollectionRepository) ListAssets(ctx context.Context, collectionID uuid.UUID, limit, offset int) ([]*domain.Asset, error) {
	var rows []assetRow
	err := r.db.SelectContext(ctx, &rows, `
		SELECT a.* FROM assets a
		JOIN collection_assets ca ON ca.asset_id = a.id
		WHERE ca.collection_id = $1 AND a.deleted_at IS NULL
		ORDER BY ca.sort_order, a.created_at DESC
		LIMIT $2 OFFSET $3`,
		collectionID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	var assets []*domain.Asset
	for _, row := range rows {
		a, err := rowToAsset(row)
		if err != nil {
			continue
		}
		assets = append(assets, a)
	}
	return assets, nil
}

// DomainRepository

type DomainRepository struct{ db *sqlx.DB }

func NewDomainRepository(db *sqlx.DB) *DomainRepository { return &DomainRepository{db: db} }

func (r *DomainRepository) Create(ctx context.Context, dr *domain.DomainRecord) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO domains (id, org_id, domain, is_primary, tls_status, challenge_token, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		dr.ID, dr.OrgID, dr.Domain, dr.IsPrimary, string(dr.TLSStatus), dr.ChallengeToken, dr.CreatedAt, dr.UpdatedAt,
	)
	return err
}

func (r *DomainRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.DomainRecord, error) {
	var dr domain.DomainRecord
	if err := r.db.GetContext(ctx, &dr, `SELECT * FROM domains WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &dr, nil
}

func (r *DomainRepository) GetByDomain(ctx context.Context, domainName string) (*domain.DomainRecord, error) {
	var dr domain.DomainRecord
	if err := r.db.GetContext(ctx, &dr, `SELECT * FROM domains WHERE domain=$1`, domainName); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &dr, nil
}

func (r *DomainRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.DomainRecord, error) {
	var records []*domain.DomainRecord
	err := r.db.SelectContext(ctx, &records, `SELECT * FROM domains WHERE org_id=$1 ORDER BY created_at`, orgID)
	return records, err
}

func (r *DomainRepository) Update(ctx context.Context, dr *domain.DomainRecord) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE domains SET is_primary=$1, verified_at=$2, tls_status=$3, updated_at=$4 WHERE id=$5`,
		dr.IsPrimary, dr.VerifiedAt, string(dr.TLSStatus), dr.UpdatedAt, dr.ID,
	)
	return err
}

func (r *DomainRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM domains WHERE id=$1`, id)
	return err
}

// StyleRepository

type StyleRepository struct{ db *sqlx.DB }

func NewStyleRepository(db *sqlx.DB) *StyleRepository { return &StyleRepository{db: db} }

func (r *StyleRepository) Create(ctx context.Context, s *domain.ImageStyle) error {
	ops, _ := json.Marshal(s.Operations)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO image_styles (id, org_id, name, slug, operations, output_format, quality, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		s.ID, s.OrgID, s.Name, s.Slug, ops, string(s.OutputFormat), s.Quality, s.CreatedAt, s.UpdatedAt,
	)
	return err
}

func (r *StyleRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.ImageStyle, error) {
	return r.getStyle(ctx, `SELECT * FROM image_styles WHERE id=$1`, id)
}

func (r *StyleRepository) GetBySlug(ctx context.Context, orgID uuid.UUID, slug string) (*domain.ImageStyle, error) {
	return r.getStyle(ctx, `SELECT * FROM image_styles WHERE org_id=$1 AND slug=$2`, orgID, slug)
}

func (r *StyleRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.ImageStyle, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT * FROM image_styles WHERE org_id=$1 ORDER BY name`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var styles []*domain.ImageStyle
	for rows.Next() {
		s, err := r.scanStyle(rows)
		if err != nil {
			continue
		}
		styles = append(styles, s)
	}
	return styles, nil
}

func (r *StyleRepository) Update(ctx context.Context, s *domain.ImageStyle) error {
	ops, _ := json.Marshal(s.Operations)
	_, err := r.db.ExecContext(ctx, `
		UPDATE image_styles SET name=$1, operations=$2, output_format=$3, quality=$4, updated_at=$5 WHERE id=$6`,
		s.Name, ops, string(s.OutputFormat), s.Quality, s.UpdatedAt, s.ID,
	)
	return err
}

func (r *StyleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM image_styles WHERE id=$1`, id)
	return err
}

func (r *StyleRepository) getStyle(ctx context.Context, query string, args ...interface{}) (*domain.ImageStyle, error) {
	row := r.db.QueryRowxContext(ctx, query, args...)
	s, err := r.scanStyle(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return s, nil
}

type styleScanner interface {
	Scan(dest ...interface{}) error
	StructScan(dest interface{}) error
}

func (r *StyleRepository) scanStyle(row styleScanner) (*domain.ImageStyle, error) {
	var raw struct {
		ID           uuid.UUID `db:"id"`
		OrgID        uuid.UUID `db:"org_id"`
		Name         string    `db:"name"`
		Slug         string    `db:"slug"`
		Operations   []byte    `db:"operations"`
		OutputFormat string    `db:"output_format"`
		Quality      int       `db:"quality"`
		CreatedAt    time.Time `db:"created_at"`
		UpdatedAt    time.Time `db:"updated_at"`
	}
	if err := row.StructScan(&raw); err != nil {
		return nil, err
	}
	var ops []domain.StyleOperation
	_ = json.Unmarshal(raw.Operations, &ops)
	return &domain.ImageStyle{
		ID:           raw.ID,
		OrgID:        raw.OrgID,
		Name:         raw.Name,
		Slug:         raw.Slug,
		Operations:   ops,
		OutputFormat: domain.OutputFormat(raw.OutputFormat),
		Quality:      raw.Quality,
		CreatedAt:    raw.CreatedAt,
		UpdatedAt:    raw.UpdatedAt,
	}, nil
}

// TransformCacheRepository

type TransformCacheRepository struct{ db *sqlx.DB }

func NewTransformCacheRepository(db *sqlx.DB) *TransformCacheRepository {
	return &TransformCacheRepository{db: db}
}

func (r *TransformCacheRepository) Get(ctx context.Context, assetID uuid.UUID, paramsHash string) (*domain.TransformCache, error) {
	var tc domain.TransformCache
	if err := r.db.GetContext(ctx, &tc, `
		SELECT * FROM transform_cache WHERE asset_id=$1 AND params_hash=$2`, assetID, paramsHash,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &tc, nil
}

func (r *TransformCacheRepository) Create(ctx context.Context, tc *domain.TransformCache) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO transform_cache (id, asset_id, style_id, params_hash, storage_key, size_bytes, format, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (asset_id, params_hash) DO UPDATE SET storage_key=$5, size_bytes=$6, created_at=$8`,
		tc.ID, tc.AssetID, tc.StyleID, tc.ParamsHash, tc.StorageKey, tc.SizeBytes, tc.Format, tc.CreatedAt,
	)
	return err
}

func (r *TransformCacheRepository) DeleteByAsset(ctx context.Context, assetID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM transform_cache WHERE asset_id=$1`, assetID)
	return err
}

func (r *TransformCacheRepository) DeleteByStyle(ctx context.Context, styleID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM transform_cache WHERE style_id=$1`, styleID)
	return err
}

// WebhookRepository

type WebhookRepository struct{ db *sqlx.DB }

func NewWebhookRepository(db *sqlx.DB) *WebhookRepository { return &WebhookRepository{db: db} }

func (r *WebhookRepository) Create(ctx context.Context, w *domain.Webhook) error {
	events, _ := json.Marshal(w.Events)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO webhooks (id, org_id, url, events, secret_hash, active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		w.ID, w.OrgID, w.URL, events, w.SecretHash, w.Active, w.CreatedAt, w.UpdatedAt,
	)
	return err
}

func (r *WebhookRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Webhook, error) {
	var raw struct {
		domain.Webhook
		Events []byte `db:"events"`
	}
	if err := r.db.GetContext(ctx, &raw, `SELECT * FROM webhooks WHERE id=$1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	_ = json.Unmarshal(raw.Events, &raw.Webhook.Events)
	return &raw.Webhook, nil
}

func (r *WebhookRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Webhook, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT * FROM webhooks WHERE org_id=$1 ORDER BY created_at`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanWebhooks(rows)
}

func (r *WebhookRepository) ListActiveByEvent(ctx context.Context, orgID uuid.UUID, event string) ([]*domain.Webhook, error) {
	rows, err := r.db.QueryxContext(ctx, `
		SELECT * FROM webhooks WHERE org_id=$1 AND active=true AND events @> $2::jsonb`,
		orgID, `["`+event+`"]`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanWebhooks(rows)
}

func (r *WebhookRepository) Update(ctx context.Context, w *domain.Webhook) error {
	events, _ := json.Marshal(w.Events)
	_, err := r.db.ExecContext(ctx, `
		UPDATE webhooks SET url=$1, events=$2, active=$3, updated_at=$4 WHERE id=$5`,
		w.URL, events, w.Active, w.UpdatedAt, w.ID,
	)
	return err
}

func (r *WebhookRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM webhooks WHERE id=$1`, id)
	return err
}

func (r *WebhookRepository) CreateDelivery(ctx context.Context, d *domain.WebhookDelivery) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO webhook_deliveries (id, webhook_id, event, payload, status, attempts, next_retry_at, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		d.ID, d.WebhookID, d.Event, d.Payload, d.Status, d.Attempts, d.NextRetryAt, d.CreatedAt, d.UpdatedAt,
	)
	return err
}

func (r *WebhookRepository) UpdateDelivery(ctx context.Context, d *domain.WebhookDelivery) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE webhook_deliveries SET status=$1, attempts=$2, next_retry_at=$3, updated_at=$4 WHERE id=$5`,
		d.Status, d.Attempts, d.NextRetryAt, d.UpdatedAt, d.ID,
	)
	return err
}

func (r *WebhookRepository) ListDeliveries(ctx context.Context, webhookID uuid.UUID, limit, offset int) ([]*domain.WebhookDelivery, error) {
	var deliveries []*domain.WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, `
		SELECT * FROM webhook_deliveries WHERE webhook_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		webhookID, limit, offset,
	)
	return deliveries, err
}

func (r *WebhookRepository) ListPendingDeliveries(ctx context.Context, limit int) ([]*domain.WebhookDelivery, error) {
	var deliveries []*domain.WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, `
		SELECT * FROM webhook_deliveries WHERE status='pending' AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at LIMIT $1`,
		limit,
	)
	return deliveries, err
}

func (r *WebhookRepository) scanWebhooks(rows *sqlx.Rows) ([]*domain.Webhook, error) {
	var webhooks []*domain.Webhook
	for rows.Next() {
		var raw struct {
			domain.Webhook
			Events []byte `db:"events"`
		}
		if err := rows.StructScan(&raw); err != nil {
			continue
		}
		_ = json.Unmarshal(raw.Events, &raw.Webhook.Events)
		wh := raw.Webhook
		webhooks = append(webhooks, &wh)
	}
	return webhooks, nil
}

// AuditLogRepository

type AuditLogRepository struct{ db *sqlx.DB }

func NewAuditLogRepository(db *sqlx.DB) *AuditLogRepository { return &AuditLogRepository{db: db} }

func (r *AuditLogRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	meta, _ := json.Marshal(log.Metadata)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, org_id, user_id, action, resource_type, resource_id, metadata, ip, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		log.ID, log.OrgID, log.UserID, log.Action, log.ResourceType, log.ResourceID, meta, log.IP, log.CreatedAt,
	)
	return err
}

func (r *AuditLogRepository) List(ctx context.Context, orgID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	err := r.db.SelectContext(ctx, &logs, `
		SELECT * FROM audit_logs WHERE org_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		orgID, limit, offset,
	)
	return logs, err
}
