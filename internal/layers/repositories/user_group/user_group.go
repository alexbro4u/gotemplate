package usergroup

import (
	"context"
	"database/sql"
	"errors"

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
	return &Repository{
		db:   deps.DB,
		exec: deps.DB,
	}, nil
}

func (r *Repository) WithExecutor(exec transaction.SqlxTx) repositories.UserGroupRepository {
	if exec == nil {
		exec = r.db
	}
	return &Repository{
		db:   r.db,
		exec: exec,
	}
}

func (r *Repository) GetGroupNamesByUserID(ctx context.Context, userID int64) ([]string, error) {
	query := `
		SELECT g.name
		FROM groups g
		INNER JOIN user_groups ug ON ug.group_id = g.id
		WHERE ug.user_id = $1
	`
	rows, err := r.exec.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, apperrors.Wrap(err, "get group names by user id")
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if scanErr := rows.Scan(&name); scanErr != nil {
			return nil, apperrors.Wrap(scanErr, "scan group name")
		}
		names = append(names, name)
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, apperrors.Wrap(rowsErr, "rows")
	}
	return names, nil
}

func (r *Repository) AddUserToGroup(ctx context.Context, userID int64, groupName string) error {
	var groupID int64
	err := r.exec.QueryRowContext(ctx, `SELECT id FROM groups WHERE name = $1`, groupName).Scan(&groupID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.ErrGroupNotFound
		}
		return apperrors.Wrap(err, "get group by name")
	}

	_, err = r.exec.ExecContext(ctx, `
		INSERT INTO user_groups (user_id, group_id) VALUES ($1, $2)
		ON CONFLICT (user_id, group_id) DO NOTHING
	`, userID, groupID)
	if err != nil {
		return apperrors.Wrap(err, "add user to group")
	}
	return nil
}
