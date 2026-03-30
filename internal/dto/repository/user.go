package repository

import (
	"github.com/google/uuid"
	"github.com/alexbro4u/gotemplate/internal/entity"
)

type CreateUserInput struct {
	Email        string
	Name         string
	PasswordHash string
	Role         *string
}

type CreateUserOutput struct {
	User *entity.User
}

type GetUserInput struct {
	UUID uuid.UUID
}

type GetUserOutput struct {
	User *entity.User
}

type UpdateUserInput struct {
	UUID  uuid.UUID
	Email *string
	Name  *string
}

type UpdatePasswordInput struct {
	UUID         uuid.UUID
	PasswordHash string
}

type DeleteUserInput struct {
	UUID uuid.UUID
}

type ListUsersOutput struct {
	Users []*entity.User
	Total int64
}
