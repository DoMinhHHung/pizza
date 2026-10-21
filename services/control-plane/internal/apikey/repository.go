package apikey

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, key *ApiKey) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO control_plane.api_keys
			(id, project_id, name, key_prefix, key_hash, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		key.ID, key.ProjectID, key.Name, key.KeyPrefix, key.KeyHash, key.CreatedBy, key.CreatedAt,
	)
	return err
}

func (r *Repository) ListByProjectID(ctx context.Context, projectID uuid.UUID) ([]ApiKey, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, name, key_prefix, key_hash, created_by, created_at, revoked_at, last_used_at
		FROM control_plane.api_keys
		WHERE project_id = $1
		ORDER BY created_at DESC`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ApiKey
	for rows.Next() {
		var k ApiKey
		if err := rows.Scan(
			&k.ID, &k.ProjectID, &k.Name, &k.KeyPrefix, &k.KeyHash,
			&k.CreatedBy, &k.CreatedAt, &k.RevokedAt, &k.LastUsedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, k)
	}
	return result, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*ApiKey, error) {
	var k ApiKey
	err := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, key_prefix, key_hash, created_by, created_at, revoked_at, last_used_at
		FROM control_plane.api_keys WHERE id = $1`, id,
	).Scan(
		&k.ID, &k.ProjectID, &k.Name, &k.KeyPrefix, &k.KeyHash,
		&k.CreatedBy, &k.CreatedAt, &k.RevokedAt, &k.LastUsedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *Repository) Revoke(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE control_plane.api_keys
		SET revoked_at = $2
		WHERE id = $1 AND revoked_at IS NULL`, id, time.Now().UTC())
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) FindActiveByHash(ctx context.Context, hash string) (*ApiKey, error) {
	var k ApiKey
	err := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, key_prefix, key_hash, created_by, created_at, revoked_at, last_used_at
		FROM control_plane.api_keys
		WHERE key_hash = $1 AND revoked_at IS NULL`, hash,
	).Scan(
		&k.ID, &k.ProjectID, &k.Name, &k.KeyPrefix, &k.KeyHash,
		&k.CreatedBy, &k.CreatedAt, &k.RevokedAt, &k.LastUsedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &k, nil
}

func (r *Repository) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE control_plane.api_keys
		SET last_used_at = $2
		WHERE id = $1 AND revoked_at IS NULL`, id, time.Now().UTC())
	return err
}
