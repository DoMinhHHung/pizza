-- +goose Up
CREATE SCHEMA IF NOT EXISTS control_plane;

CREATE TABLE control_plane.organizations (
    id          UUID PRIMARY KEY,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_organizations_slug UNIQUE (slug)
);

CREATE TABLE control_plane.organization_members (
    org_id      UUID NOT NULL REFERENCES control_plane.organizations(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL,
    role        TEXT NOT NULL CHECK (role IN ('OWNER', 'MEMBER')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX idx_org_members_user_id ON control_plane.organization_members (user_id);

CREATE TABLE control_plane.projects (
    id          UUID PRIMARY KEY,
    org_id      UUID NOT NULL REFERENCES control_plane.organizations(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_projects_org_slug UNIQUE (org_id, slug)
);

CREATE INDEX idx_projects_org_id ON control_plane.projects (org_id);

-- +goose Down
DROP TABLE IF EXISTS control_plane.projects;
DROP TABLE IF EXISTS control_plane.organization_members;
DROP TABLE IF EXISTS control_plane.organizations;
DROP SCHEMA IF EXISTS control_plane;
