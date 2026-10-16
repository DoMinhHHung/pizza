package org

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

func (r *Repository) Create(ctx context.Context, name, slug string, ownerID uuid.UUID) (*Organization, error) {
	org := &Organization{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO control_plane.organizations (id, name, slug, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)`,
		org.ID, org.Name, org.Slug, org.CreatedAt, org.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert organization: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO control_plane.organization_members (org_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4)`,
		org.ID, ownerID, RoleOwner, org.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("insert owner member: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return org, nil
}

func (r *Repository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM control_plane.organizations WHERE slug = $1)`, slug,
	).Scan(&exists)
	return exists, err
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]OrganizationWithRole, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT o.id, o.name, o.slug, o.created_at, o.updated_at, m.role
		FROM control_plane.organizations o
		INNER JOIN control_plane.organization_members m ON m.org_id = o.id
		WHERE m.user_id = $1
		ORDER BY o.created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	var result []OrganizationWithRole
	for rows.Next() {
		var item OrganizationWithRole
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Slug,
			&item.CreatedAt, &item.UpdatedAt, &item.Role,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repository) HasMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM control_plane.organization_members
			WHERE org_id = $1 AND user_id = $2
		)`, orgID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *Repository) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*Member, error) {
	var m Member
	err := r.pool.QueryRow(ctx, `
		SELECT org_id, user_id, role, created_at
		FROM control_plane.organization_members
		WHERE org_id = $1 AND user_id = $2`, orgID, userID,
	).Scan(&m.OrgID, &m.UserID, &m.Role, &m.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (r *Repository) ListMembers(ctx context.Context, orgID uuid.UUID) ([]Member, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT org_id, user_id, role, created_at
		FROM control_plane.organization_members
		WHERE org_id = $1
		ORDER BY created_at ASC`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Member
	for rows.Next() {
		var m Member
		if err := rows.Scan(&m.OrgID, &m.UserID, &m.Role, &m.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *Repository) AddMember(ctx context.Context, orgID, userID uuid.UUID, role Role) (*Member, error) {
	m := &Member{
		OrgID:     orgID,
		UserID:    userID,
		Role:      role,
		CreatedAt: time.Now().UTC(),
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO control_plane.organization_members (org_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4)`,
		m.OrgID, m.UserID, m.Role, m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *Repository) UpdateMemberRole(ctx context.Context, orgID, userID uuid.UUID, role Role) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE control_plane.organization_members
		SET role = $3
		WHERE org_id = $1 AND user_id = $2`, orgID, userID, role)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) RemoveMember(ctx context.Context, orgID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM control_plane.organization_members
		WHERE org_id = $1 AND user_id = $2`, orgID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) CountOwners(ctx context.Context, orgID uuid.UUID) (int, error) {
	var n int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM control_plane.organization_members
		WHERE org_id = $1 AND role = $2`, orgID, RoleOwner,
	).Scan(&n)
	return n, err
}
