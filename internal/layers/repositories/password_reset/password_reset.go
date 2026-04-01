package passwordreset

import (
	"context"
	"database/sql"
	"errors"

	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories/transaction"

	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
)

type Deps struct {
	DB        *sqlx.DB            `validate:"required"`
	Validator *validator.Validate `validate:"required"`
}

type Repository struct {
	db   *sqlx.DB
	exec transaction.SqlxTx
}

func New(deps Deps) (*Repository, error) {
	if err := deps.Validator.Struct(deps); err != nil {
		return nil, apperrors.Wrap(err, "validate deps")
	}
	return &Repository{db: deps.DB, exec: deps.DB}, nil
}

func (r *Repository) Create(ctx context.Context, in repository.CreatePasswordResetInput) error {
	_, err := r.exec.ExecContext(ctx,
		`INSERT INTO password_reset_tokens (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		in.Token, in.UserID, in.ExpiresAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "create password reset token")
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, in repository.GetPasswordResetInput) (*repository.GetPasswordResetOutput, error) {
	var out repository.GetPasswordResetOutput
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id, expires_at, used FROM password_reset_tokens WHERE token = $1`,
		in.Token,
	).Scan(&out.UserID, &out.ExpiresAt, &out.Used)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrTokenNotFound
		}
		return nil, apperrors.Wrap(err, "get password reset token")
	}
	return &out, nil
}

func (r *Repository) MarkUsed(ctx context.Context, in repository.MarkPasswordResetUsedInput) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE password_reset_tokens SET used = TRUE WHERE token = $1`,
		in.Token,
	)
	if err != nil {
		return apperrors.Wrap(err, "mark password reset token used")
	}
	return nil
}

func (r *Repository) DeleteExpired(ctx context.Context) error {
	_, err := r.exec.ExecContext(ctx,
		`DELETE FROM password_reset_tokens WHERE expires_at < NOW() OR used = TRUE`,
	)
	if err != nil {
		return apperrors.Wrap(err, "delete expired password reset tokens")
	}
	return nil
}

func (r *Repository) WithExecutor(exec transaction.SqlxTx) repositories.PasswordResetRepository {
	if exec == nil {
		exec = r.db
	}
	return &Repository{db: r.db, exec: exec}
}

var _ repositories.PasswordResetRepository = (*Repository)(nil)
