package org

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner  Role = "OWNER"
	RoleAdmin  Role = "ADMIN"
	RoleMember Role = "MEMBER"
)

func ParseRole(s string) (Role, bool) {
	switch Role(s) {
	case RoleOwner, RoleAdmin, RoleMember:
		return Role(s), true
	default:
		return "", false
	}
}

func (r Role) Rank() int {
	switch r {
	case RoleOwner:
		return 3
	case RoleAdmin:
		return 2
	case RoleMember:
		return 1
	default:
		return 0
	}
}

func (r Role) CanManageMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}

type Organization struct {
	ID        uuid.UUID
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Member struct {
	OrgID     uuid.UUID
	UserID    uuid.UUID
	Role      Role
	CreatedAt time.Time
}

type OrganizationWithRole struct {
	Organization
	Role Role
}
