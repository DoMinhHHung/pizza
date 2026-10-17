package project

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID        uuid.UUID
	OrgID     uuid.UUID
	Name      string
	Slug      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
