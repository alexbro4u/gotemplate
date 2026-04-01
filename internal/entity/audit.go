package entity

import (
	"time"

	"github.com/google/uuid"
)

type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
)

type AuditEntry struct {
	ID         int64       `db:"id"`
	EntityType string      `db:"entity_type"`
	EntityID   string      `db:"entity_id"`
	ActorUUID  uuid.UUID   `db:"actor_uuid"`
	Action     AuditAction `db:"action"`
	OldValue   []byte      `db:"old_value"`
	NewValue   []byte      `db:"new_value"`
	CreatedAt  time.Time   `db:"created_at"`
}
