-- +goose Up
ALTER TABLE control_plane.organization_members
    DROP CONSTRAINT IF EXISTS organization_members_role_check;

ALTER TABLE control_plane.organization_members
    ADD CONSTRAINT organization_members_role_check
    CHECK (role IN ('OWNER', 'ADMIN', 'MEMBER'));

CREATE TABLE control_plane.api_keys (
    id           UUID PRIMARY KEY,
    project_id   UUID NOT NULL REFERENCES control_plane.projects(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    key_prefix   TEXT NOT NULL,
    key_hash     TEXT NOT NULL,
    created_by   UUID NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    CONSTRAINT uq_api_keys_hash UNIQUE (key_hash)
);

CREATE INDEX idx_api_keys_project_id ON control_plane.api_keys (project_id);
CREATE INDEX idx_api_keys_prefix ON control_plane.api_keys (key_prefix);

-- +goose Down
DROP TABLE IF EXISTS control_plane.api_keys;

ALTER TABLE control_plane.organization_members
    DROP CONSTRAINT IF EXISTS organization_members_role_check;

ALTER TABLE control_plane.organization_members
    ADD CONSTRAINT organization_members_role_check
    CHECK (role IN ('OWNER', 'MEMBER'));
