package tokenblacklist

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

func (r *Repository) Add(ctx context.Context, in repository.AddToBlacklistInput) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO token_blacklist (jti, expires_at) VALUES ($1, $2) ON CONFLICT (jti) DO NOTHING`,
		in.JTI, in.ExpiresAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "add to blacklist")
	}
	return nil
}

func (r *Repository) DeleteExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM token_blacklist WHERE expires_at < NOW()`)
	if err != nil {
		return apperrors.Wrap(err, "delete expired blacklist entries")
	}
	return nil
}

var _ repositories.BlacklistRepository = (*Repository)(nil)
