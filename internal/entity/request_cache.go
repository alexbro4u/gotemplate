package entity

import (
	"time"

	"github.com/google/uuid"
)

type RequestCache struct {
	ID          int64
	UserID      uuid.UUID
	Path        string
	HTTPVerb    string
	RequestID   string
	Response    []byte
	StatusCode  int
	ContentType string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}
