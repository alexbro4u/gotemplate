package user

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	"github.com/alexbro4u/gotemplate/internal/entity"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories/transaction"

	"github.com/go-playground/validator/v10"
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

	return &Repository{
		db:   deps.DB,
		exec: deps.DB,
	}, nil
}

func (r *Repository) WithExecutor(exec transaction.SqlxTx) repositories.UserRepository {
	if exec == nil {
		exec = r.db
	}
	return &Repository{
		db:   r.db,
		exec: exec,
	}
}

func (r *Repository) Create(ctx context.Context, in repository.CreateUserInput) (*repository.CreateUserOutput, error) {
	userUUID := uuid.New()
	now := time.Now()

	query := `
		INSERT INTO users (uuid, email, name, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, uuid, email, name, password_hash, role, created_at, updated_at
	`

	var user entity.User
	role := string(entity.RoleUser)
	if in.Role != nil {
		role = *in.Role
	}

	err := r.exec.QueryRowContext(ctx, query, userUUID, in.Email, in.Name, in.PasswordHash, role, now, now).Scan(
		&user.ID,
		&user.UUID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return nil, apperrors.Wrap(apperrors.ErrUniqueViolation, "create user")
		}
		return nil, apperrors.Wrap(err, "create user")
	}

	return &repository.CreateUserOutput{
		User: &user,
	}, nil
}

func (r *Repository) Get(ctx context.Context, in repository.GetUserInput) (*repository.GetUserOutput, error) {
	query := `
		SELECT id, uuid, email, name, password_hash, role, created_at, updated_at
		FROM users
		WHERE uuid = $1
	`

	var user entity.User
	err := r.exec.QueryRowContext(ctx, query, in.UUID).Scan(
		&user.ID,
		&user.UUID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, apperrors.Wrap(err, "get user")
	}

	return &repository.GetUserOutput{
		User: &user,
	}, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*repository.GetUserOutput, error) {
	query := `
		SELECT id, uuid, email, name, password_hash, role, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	var user entity.User
	err := r.exec.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.UUID,
		&user.Email,
		&user.Name,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.ErrUserNotFound
		}
		return nil, apperrors.Wrap(err, "get user by email")
	}

	return &repository.GetUserOutput{
		User: &user,
	}, nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) (*repository.ListUsersOutput, error) {
	// Get total count
	var total int64
	countQuery := `SELECT COUNT(*) FROM users`
	err := r.exec.GetContext(ctx, &total, countQuery)
	if err != nil {
		return nil, apperrors.Wrap(err, "count users")
	}

	// Get paginated results
	query := `
		SELECT id, uuid, email, name, password_hash, role, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	var users []*entity.User
	err = r.exec.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(err, "list users")
	}

	return &repository.ListUsersOutput{
		Users: users,
		Total: total,
	}, nil
}

func (r *Repository) Update(ctx context.Context, in repository.UpdateUserInput) error {
	query := `
		UPDATE users
		SET email = COALESCE($1, email),
		    name = COALESCE($2, name),
		    updated_at = $3
		WHERE uuid = $4
	`

	result, err := r.exec.ExecContext(ctx, query, in.Email, in.Name, time.Now(), in.UUID)
	if err != nil {
		return apperrors.Wrap(err, "update user")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err, "get rows affected")
	}
	if rowsAffected == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

func (r *Repository) UpdatePassword(ctx context.Context, in repository.UpdatePasswordInput) error {
	query := `
		UPDATE users
		SET password_hash = $1, updated_at = $2
		WHERE uuid = $3
	`

	result, err := r.exec.ExecContext(ctx, query, in.PasswordHash, time.Now(), in.UUID)
	if err != nil {
		return apperrors.Wrap(err, "update password")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err, "get rows affected")
	}
	if rowsAffected == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}

func (r *Repository) Delete(ctx context.Context, in repository.DeleteUserInput) error {
	query := `DELETE FROM users WHERE uuid = $1`

	result, err := r.exec.ExecContext(ctx, query, in.UUID)
	if err != nil {
		return apperrors.Wrap(err, "delete user")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return apperrors.Wrap(err, "get rows affected")
	}
	if rowsAffected == 0 {
		return apperrors.ErrUserNotFound
	}

	return nil
}
