package repositories

import (
	"context"

	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories/transaction"
)

type UserRepository interface {
	Create(ctx context.Context, in repository.CreateUserInput) (*repository.CreateUserOutput, error)
	Get(ctx context.Context, in repository.GetUserInput) (*repository.GetUserOutput, error)
	GetByEmail(ctx context.Context, email string) (*repository.GetUserOutput, error)
	Update(ctx context.Context, in repository.UpdateUserInput) error
	UpdatePassword(ctx context.Context, in repository.UpdatePasswordInput) error
	UpdatePasswordByID(ctx context.Context, in repository.UpdatePasswordByIDInput) error
	Delete(ctx context.Context, in repository.DeleteUserInput) error
	List(ctx context.Context, limit, offset int) (*repository.ListUsersOutput, error)
	WithExecutor(exec transaction.SqlxTx) UserRepository
}

type UserGroupRepository interface {
	GetGroupNamesByUserID(ctx context.Context, userID int64) ([]string, error)
	AddUserToGroup(ctx context.Context, userID int64, groupName string) error
	WithExecutor(exec transaction.SqlxTx) UserGroupRepository
}

type RequestCacheRepository interface {
	Get(ctx context.Context, in repository.GetRequestCacheInput) (*repository.GetRequestCacheOutput, error)
	Create(ctx context.Context, in repository.CreateRequestCacheInput) error
	DeleteExpired(ctx context.Context) error
}

type BlacklistRepository interface {
	Add(ctx context.Context, in repository.AddToBlacklistInput) error
	DeleteExpired(ctx context.Context) error
}

type AuditRepository interface {
	Log(ctx context.Context, in repository.LogAuditInput) error
}

type PasswordResetRepository interface {
	Create(ctx context.Context, in repository.CreatePasswordResetInput) error
	Get(ctx context.Context, in repository.GetPasswordResetInput) (*repository.GetPasswordResetOutput, error)
	MarkUsed(ctx context.Context, in repository.MarkPasswordResetUsedInput) error
	DeleteExpired(ctx context.Context) error
}
