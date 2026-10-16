package project

import (
	"context"
	"errors"
	"fmt"
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

func (r *Repository) Create(ctx context.Context, orgID uuid.UUID, name, slug string) (*Project, error) {
	p := &Project{
		ID:        uuid.New(),
		OrgID:     orgID,
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO control_plane.projects (id, org_id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		p.ID, p.OrgID, p.Name, p.Slug, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}
	return p, nil
}

func (r *Repository) ExistsByOrgAndSlug(ctx context.Context, orgID uuid.UUID, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM control_plane.projects WHERE org_id = $1 AND slug = $2
		)`, orgID, slug,
	).Scan(&exists)
	return exists, err
}

func (r *Repository) ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]Project, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, org_id, name, slug, created_at, updated_at
		FROM control_plane.projects
		WHERE org_id = $1
		ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.OrgID, &p.Name, &p.Slug, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Project, error) {
	var p Project
	err := r.pool.QueryRow(ctx, `
		SELECT id, org_id, name, slug, created_at, updated_at
		FROM control_plane.projects WHERE id = $1`, id,
	).Scan(&p.ID, &p.OrgID, &p.Name, &p.Slug, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}
