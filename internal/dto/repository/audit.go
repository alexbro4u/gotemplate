package repository

import (
	"github.com/alexbro4u/gotemplate/internal/entity"
	"github.com/google/uuid"
)

type LogAuditInput struct {
	EntityType string
	EntityID   string
	ActorUUID  uuid.UUID
	Action     entity.AuditAction
	OldValue   []byte
	NewValue   []byte
}
