package org

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner  Role = "OWNER"
	RoleMember Role = "MEMBER"
)

type Organization struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Member struct {
	OrgID  uuid.UUID
	UserID uuid.UUID
	Role   Role
}

type OrganizationWithRole struct {
	Organization
	Role Role
}
