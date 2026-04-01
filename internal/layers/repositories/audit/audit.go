package audit

import (
	"context"

	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"
	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
)

type Deps struct {
	DB        *sqlx.DB            `validate:"required"`
	Validator *validator.Validate `validate:"required"`
}

type Repository struct {
	db *sqlx.DB
}

func New(deps Deps) (*Repository, error) {
	if err := deps.Validator.Struct(deps); err != nil {
		return nil, apperrors.Wrap(err, "validate deps")
	}
	return &Repository{db: deps.DB}, nil
}

func (r *Repository) Log(ctx context.Context, in repository.LogAuditInput) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_log (entity_type, entity_id, actor_uuid, action, old_value, new_value)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		in.EntityType, in.EntityID, in.ActorUUID, string(in.Action), in.OldValue, in.NewValue,
	)
	if err != nil {
		return apperrors.Wrap(err, "log audit entry")
	}
	return nil
}

var _ repositories.AuditRepository = (*Repository)(nil)
