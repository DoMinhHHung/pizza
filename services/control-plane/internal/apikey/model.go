package apikey

import (
	"time"

	"github.com/google/uuid"
)

type ApiKey struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	Name       string
	KeyPrefix  string
	KeyHash    string
	CreatedBy  uuid.UUID
	CreatedAt  time.Time
	RevokedAt  *time.Time
	LastUsedAt *time.Time
}
