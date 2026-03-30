package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/alexbro4u/gotemplate/internal/entity"
)

func UserToDTO(user *entity.User) UserDTO {
	if user == nil {
		return UserDTO{}
	}
	return UserDTO{
		ID:        user.ID,
		UUID:      user.UUID.String(),
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

type CreateUserInput struct {
	Email    string `validate:"required,email"`
	Name     string `validate:"required,min=1,max=100"`
	Password string `validate:"required,min=6"`
}

type UserDTO struct {
	ID        int64
	UUID      string
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateUserOutput struct {
	User UserDTO
}

type GetUserInput struct {
	UUID uuid.UUID
}

type GetUserOutput struct {
	User UserDTO
}

type UpdateUserInput struct {
	UUID  uuid.UUID `validate:"required"`
	Email *string   `validate:"omitempty,email"`
	Name  *string   `validate:"omitempty,min=1,max=100"`
}

type DeleteUserInput struct {
	UUID uuid.UUID
}

type ListUsersInput struct {
	Limit  int `validate:"required,min=1,max=100"`
	Offset int `validate:"min=0"`
}

type ListUsersOutput struct {
	Users  []UserDTO
	Total  int64
	Limit  int
	Offset int
}
