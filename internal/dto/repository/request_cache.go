package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/alexbro4u/gotemplate/internal/entity"
)

type GetRequestCacheInput struct {
	UserID    uuid.UUID
	Path      string
	HTTPVerb  string
	RequestID string
}

type GetRequestCacheOutput struct {
	RequestCache *entity.RequestCache
}

type CreateRequestCacheInput struct {
	UserID      uuid.UUID
	Path        string
	HTTPVerb    string
	RequestID   string
	Response    []byte
	StatusCode  int
	ContentType string
	ExpiresAt   time.Time
}
