package request_cache

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	"github.com/alexbro4u/gotemplate/internal/entity"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"

	"github.com/go-playground/validator/v10"
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

	return &Repository{
		db: deps.DB,
	}, nil
}

func (r *Repository) Get(ctx context.Context, in repository.GetRequestCacheInput) (*repository.GetRequestCacheOutput, error) {
	query := `
		SELECT id, user_id, path, http_verb, request_id, response, status_code, content_type, created_at, expires_at
		FROM request_cache
		WHERE user_id = $1 AND path = $2 AND http_verb = $3 AND request_id = $4
			AND expires_at > NOW()
		LIMIT 1
	`

	var rc entity.RequestCache
	err := r.db.QueryRowContext(ctx, query, in.UserID, in.Path, in.HTTPVerb, in.RequestID).Scan(
		&rc.ID, &rc.UserID, &rc.Path, &rc.HTTPVerb, &rc.RequestID,
		&rc.Response, &rc.StatusCode, &rc.ContentType, &rc.CreatedAt, &rc.ExpiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrRequestCacheNotFound
		}
		return nil, apperrors.Wrap(err, "get request cache")
	}

	return &repository.GetRequestCacheOutput{
		RequestCache: &rc,
	}, nil
}

func (r *Repository) Create(ctx context.Context, in repository.CreateRequestCacheInput) error {
	query := `
		INSERT INTO request_cache (user_id, path, http_verb, request_id, response, status_code, content_type, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id, path, http_verb, request_id) DO NOTHING
	`

	var responseParam interface{} = in.Response
	if len(in.Response) == 0 {
		responseParam = nil
	}

	contentType := in.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	_, err := r.db.ExecContext(ctx, query,
		in.UserID, in.Path, in.HTTPVerb, in.RequestID,
		responseParam, in.StatusCode, contentType,
		time.Now(), in.ExpiresAt,
	)
	if err != nil {
		return apperrors.Wrap(err, "create request cache")
	}

	return nil
}

func (r *Repository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM request_cache WHERE expires_at < NOW()`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return apperrors.Wrap(err, "delete expired request cache")
	}

	return nil
}
