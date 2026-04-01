package blacklist

import (
	"context"
	"time"

	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories/transaction"

	"github.com/go-playground/validator/v10"
	"github.com/jmoiron/sqlx"
)

type repo struct {
	db        *sqlx.DB
	exec      transaction.SqlxTx
	validator *validator.Validate
}

type Deps struct {
	DB        *sqlx.DB
	Validator *validator.Validate
}

func New(deps Deps) (repositories.BlacklistRepository, error) {
	if err := deps.Validator.Struct(deps); err != nil {
		return nil, err
	}
	return &repo{
		db:        deps.DB,
		exec:      deps.DB,
		validator: deps.Validator,
	}, nil
}

func (r *repo) Add(ctx context.Context, in repository.AddToBlacklistInput) error {
	if err := r.validator.Struct(in); err != nil {
		return err
	}

	query := `INSERT INTO token_blacklist (jti, expires_at) VALUES ($1, $2)`

	_, err := r.exec.ExecContext(ctx, query, in.JTI, in.ExpiresAt)
	if err != nil {
		return err
	}

	return nil
}

func (r *repo) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM token_blacklist WHERE expires_at < $1`

	_, err := r.exec.ExecContext(ctx, query, time.Now())
	if err != nil {
		return err
	}

	return nil
}

func (r *repo) WithExecutor(exec transaction.SqlxTx) repositories.BlacklistRepository {
	if exec == nil {
		exec = r.db
	}
	return &repo{
		db:        r.db,
		exec:      exec,
		validator: r.validator,
	}
}
