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

type OrgRepository struct {
	db *sqlx.DB
}

func NewOrgRepository(db *sqlx.DB) *OrgRepository {
	return &OrgRepository{db: db}
}

type orgRow struct {
	ID                uuid.UUID `db:"id"`
	Name              string    `db:"name"`
	Slug              string    `db:"slug"`
	Plan              string    `db:"plan"`
	StorageQuotaBytes int64     `db:"storage_quota_bytes"`
	StorageUsedBytes  int64     `db:"storage_used_bytes"`
	Settings          []byte    `db:"settings"`
	Active            bool      `db:"active"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

func (r *OrgRepository) Create(ctx context.Context, org *domain.Org) error {
	settings, err := json.Marshal(org.Settings)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO organizations (id, name, slug, plan, storage_quota_bytes, storage_used_bytes, settings, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		org.ID, org.Name, org.Slug, org.Plan,
		org.StorageQuotaBytes, org.StorageUsedBytes,
		settings, org.Active, org.CreatedAt, org.UpdatedAt,
	)
	return err
}

func (r *OrgRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Org, error) {
	var row orgRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM organizations WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToOrg(row)
}

func (r *OrgRepository) GetBySlug(ctx context.Context, slug string) (*domain.Org, error) {
	var row orgRow
	err := r.db.GetContext(ctx, &row, `SELECT * FROM organizations WHERE slug = $1`, slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToOrg(row)
}

func (r *OrgRepository) Update(ctx context.Context, org *domain.Org) error {
	settings, err := json.Marshal(org.Settings)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
		UPDATE organizations SET name=$1, plan=$2, storage_quota_bytes=$3, settings=$4, active=$5, updated_at=$6
		WHERE id=$7`,
		org.Name, org.Plan, org.StorageQuotaBytes, settings, org.Active, org.UpdatedAt, org.ID,
	)
	return err
}

func (r *OrgRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM organizations WHERE id=$1`, id)
	return err
}

func (r *OrgRepository) IncrementStorageUsed(ctx context.Context, orgID uuid.UUID, delta int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE organizations SET storage_used_bytes = storage_used_bytes + $1 WHERE id = $2`,
		delta, orgID,
	)
	return err
}

func (r *OrgRepository) List(ctx context.Context, limit, offset int) ([]*domain.Org, int, error) {
	var rows []orgRow
	if err := r.db.SelectContext(ctx, &rows, `SELECT * FROM organizations ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset); err != nil {
		return nil, 0, err
	}
	var total int
	_ = r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM organizations`).Scan(&total)

	var orgs []*domain.Org
	for _, row := range rows {
		org, err := rowToOrg(row)
		if err != nil {
			return nil, 0, err
		}
		orgs = append(orgs, org)
	}
	return orgs, total, nil
}

func rowToOrg(row orgRow) (*domain.Org, error) {
	var settings domain.OrgSettings
	if err := json.Unmarshal(row.Settings, &settings); err != nil {
		return nil, err
	}
	return &domain.Org{
		ID:                row.ID,
		Name:              row.Name,
		Slug:              row.Slug,
		Plan:              row.Plan,
		StorageQuotaBytes: row.StorageQuotaBytes,
		StorageUsedBytes:  row.StorageUsedBytes,
		Settings:          settings,
		Active:            row.Active,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}
