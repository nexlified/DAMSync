package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/nexlified/dam/internal/domain"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, org_id, email, password_hash, role, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		user.ID, user.OrgID, user.Email, user.PasswordHash, user.Role, user.Active, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var user domain.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE id = $1`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.GetContext(ctx, &user, `SELECT * FROM users WHERE org_id = $1 AND email = $2`, orgID, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET role=$1, active=$2, updated_at=$3 WHERE id=$4`,
		user.Role, user.Active, user.UpdatedAt, user.ID,
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id=$1`, id)
	return err
}

func (r *UserRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.User, error) {
	var users []*domain.User
	err := r.db.SelectContext(ctx, &users, `SELECT * FROM users WHERE org_id=$1 ORDER BY created_at DESC`, orgID)
	return users, err
}

// APIKeyRepository

type APIKeyRepository struct {
	db *sqlx.DB
}

func NewAPIKeyRepository(db *sqlx.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, key *domain.APIKey) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO api_keys (id, org_id, user_id, name, key_prefix, key_hash, scopes, ip_allowlist, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		key.ID, key.OrgID, key.UserID, key.Name, key.KeyPrefix, key.KeyHash,
		sqlStringArray(key.Scopes), sqlStringArray(key.IPAllowlist), key.CreatedAt,
	)
	return err
}

func (r *APIKeyRepository) GetByPrefix(ctx context.Context, prefix string) (*domain.APIKey, error) {
	var row struct {
		domain.APIKey
		Scopes      string `db:"scopes"`
		IPAllowlist string `db:"ip_allowlist"`
	}
	err := r.db.GetContext(ctx, &row, `SELECT * FROM api_keys WHERE key_prefix=$1`, prefix)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &row.APIKey, nil
}

func (r *APIKeyRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.APIKey, error) {
	var keys []*domain.APIKey
	err := r.db.SelectContext(ctx, &keys, `SELECT * FROM api_keys WHERE org_id=$1 AND revoked_at IS NULL ORDER BY created_at DESC`, orgID)
	return keys, err
}

func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET revoked_at=NOW() WHERE id=$1`, id)
	return err
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE api_keys SET last_used_at=NOW() WHERE id=$1`, id)
	return err
}

// sqlStringArray converts []string to PostgreSQL array literal for TEXT[] columns.
func sqlStringArray(ss []string) interface{} {
	if ss == nil {
		return "{}"
	}
	// Use pq.Array or manual encoding
	return ss
}
