package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nexlified/dam/internal/domain"
)

type AssetRepository struct {
	db *sqlx.DB
}

func NewAssetRepository(db *sqlx.DB) *AssetRepository {
	return &AssetRepository{db: db}
}

type assetRow struct {
	ID           uuid.UUID  `db:"id"`
	OrgID        uuid.UUID  `db:"org_id"`
	FolderID     *uuid.UUID `db:"folder_id"`
	Filename     string     `db:"filename"`
	StorageKey   string     `db:"storage_key"`
	MIMEType     string     `db:"mime_type"`
	SizeBytes    int64      `db:"size_bytes"`
	Width        *int       `db:"width"`
	Height       *int       `db:"height"`
	DurationMS   *int64     `db:"duration_ms"`
	Metadata     []byte     `db:"metadata"`
	Visibility   string     `db:"visibility"`
	FocalPoint   []byte     `db:"focal_point"`
	SearchVector []byte     `db:"search_vector"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"`
}

func (r *AssetRepository) Create(ctx context.Context, asset *domain.Asset) error {
	meta, err := json.Marshal(asset.Metadata)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO assets (id, org_id, folder_id, filename, storage_key, mime_type, size_bytes, width, height, duration_ms, metadata, visibility, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		asset.ID, asset.OrgID, asset.FolderID, asset.Filename, asset.StorageKey,
		asset.MIMEType, asset.SizeBytes, asset.Width, asset.Height, asset.DurationMS,
		meta, string(asset.Visibility), asset.CreatedAt, asset.UpdatedAt,
	)
	return err
}

func (r *AssetRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Asset, error) {
	var row assetRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM assets WHERE id=$1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToAsset(row)
}

func (r *AssetRepository) GetByStorageKey(ctx context.Context, storageKey string) (*domain.Asset, error) {
	var row assetRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM assets WHERE storage_key=$1 AND deleted_at IS NULL`, storageKey)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToAsset(row)
}

func (r *AssetRepository) Update(ctx context.Context, asset *domain.Asset) error {
	meta, err := json.Marshal(asset.Metadata)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE assets SET folder_id=$1, filename=$2, metadata=$3, visibility=$4, updated_at=$5 WHERE id=$6`,
		asset.FolderID, asset.Filename, meta, string(asset.Visibility), asset.UpdatedAt, asset.ID,
	)
	return err
}

func (r *AssetRepository) SoftDelete(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `UPDATE assets SET deleted_at=$1, updated_at=$1 WHERE id=$2`, now, id)
	return err
}

func (r *AssetRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM assets WHERE id=$1`, id)
	return err
}

func (r *AssetRepository) Move(ctx context.Context, assetID uuid.UUID, folderID *uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE assets SET folder_id=$1, updated_at=NOW() WHERE id=$2`, folderID, assetID)
	return err
}

func (r *AssetRepository) List(ctx context.Context, filter domain.AssetListFilter) ([]*domain.Asset, string, error) {
	q, args := buildAssetQuery(filter, false)
	var rows []assetRow
	if err := r.db.SelectContext(ctx, &rows, q, args...); err != nil {
		return nil, "", err
	}
	var assets []*domain.Asset
	for _, row := range rows {
		a, err := rowToAsset(row)
		if err != nil {
			continue
		}
		assets = append(assets, a)
	}
	var cursor string
	if len(assets) == filter.Limit && filter.Limit > 0 {
		cursor = assets[len(assets)-1].ID.String()
	}
	return assets, cursor, nil
}

func (r *AssetRepository) Count(ctx context.Context, filter domain.AssetListFilter) (int, error) {
	q, args := buildAssetQuery(filter, true)
	var count int
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&count)
	return count, err
}

func (r *AssetRepository) AddTags(ctx context.Context, assetID uuid.UUID, tagIDs []uuid.UUID) error {
	for _, tagID := range tagIDs {
		_, err := r.db.ExecContext(ctx,
			`INSERT INTO asset_tags (asset_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			assetID, tagID,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *AssetRepository) RemoveTags(ctx context.Context, assetID uuid.UUID, tagIDs []uuid.UUID) error {
	for _, tagID := range tagIDs {
		_, err := r.db.ExecContext(ctx, `DELETE FROM asset_tags WHERE asset_id=$1 AND tag_id=$2`, assetID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *AssetRepository) GetTags(ctx context.Context, assetID uuid.UUID) ([]domain.Tag, error) {
	var tags []domain.Tag
	err := r.db.SelectContext(ctx, &tags, `
		SELECT t.* FROM tags t
		JOIN asset_tags at ON at.tag_id = t.id
		WHERE at.asset_id = $1`,
		assetID,
	)
	return tags, err
}

func buildAssetQuery(filter domain.AssetListFilter, count bool) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	i := 1

	conditions = append(conditions, fmt.Sprintf("org_id = $%d", i))
	args = append(args, filter.OrgID)
	i++

	conditions = append(conditions, "deleted_at IS NULL")

	if filter.FolderID != nil {
		conditions = append(conditions, fmt.Sprintf("folder_id = $%d", i))
		args = append(args, *filter.FolderID)
		i++
	}

	if filter.Visibility != nil {
		conditions = append(conditions, fmt.Sprintf("visibility = $%d", i))
		args = append(args, string(*filter.Visibility))
		i++
	}

	if filter.MIMEGroup != "" {
		conditions = append(conditions, fmt.Sprintf("mime_type LIKE $%d", i))
		args = append(args, filter.MIMEGroup+"/%")
		i++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", i))
		args = append(args, *filter.DateFrom)
		i++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", i))
		args = append(args, *filter.DateTo)
		i++
	}

	if filter.SizeMin != nil {
		conditions = append(conditions, fmt.Sprintf("size_bytes >= $%d", i))
		args = append(args, *filter.SizeMin)
		i++
	}

	if filter.SizeMax != nil {
		conditions = append(conditions, fmt.Sprintf("size_bytes <= $%d", i))
		args = append(args, *filter.SizeMax)
		i++
	}

	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf(
			"(search_vector @@ plainto_tsquery('english', $%d) OR filename ILIKE $%d)",
			i, i+1,
		))
		args = append(args, filter.Search, "%"+filter.Search+"%")
		i += 2
	}

	if filter.Cursor != "" {
		sortDir := ">"
		if filter.SortDir == "desc" {
			sortDir = "<"
		}
		conditions = append(conditions, fmt.Sprintf("id %s $%d", sortDir, i))
		args = append(args, filter.Cursor)
		i++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	if count {
		return fmt.Sprintf("SELECT COUNT(*) FROM assets %s", where), args
	}

	allowedSortCols := map[string]bool{
		"created_at": true, "updated_at": true, "filename": true, "size_bytes": true,
	}
	sortBy := "created_at"
	if filter.SortBy != "" && allowedSortCols[filter.SortBy] {
		sortBy = filter.SortBy
	}
	sortDir := "DESC"
	if strings.ToLower(filter.SortDir) == "asc" {
		sortDir = "ASC"
	}

	limit := 50
	if filter.Limit > 0 {
		limit = filter.Limit
	}

	return fmt.Sprintf(`
		SELECT * FROM assets %s
		ORDER BY %s %s, id %s
		LIMIT %d`, where, sortBy, sortDir, sortDir, limit), args
}

func rowToAsset(row assetRow) (*domain.Asset, error) {
	var meta domain.AssetMetadata
	if err := json.Unmarshal(row.Metadata, &meta); err != nil {
		return nil, err
	}
	return &domain.Asset{
		ID:         row.ID,
		OrgID:      row.OrgID,
		FolderID:   row.FolderID,
		Filename:   row.Filename,
		StorageKey: row.StorageKey,
		MIMEType:   row.MIMEType,
		SizeBytes:  row.SizeBytes,
		Width:      row.Width,
		Height:     row.Height,
		DurationMS: row.DurationMS,
		Metadata:   meta,
		Visibility: domain.Visibility(row.Visibility),
		CreatedAt:  row.CreatedAt,
		UpdatedAt:  row.UpdatedAt,
		DeletedAt:  row.DeletedAt,
	}, nil
}
