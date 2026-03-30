package controller

import (
	"time"

	"github.com/alexbro4u/gotemplate/internal/dto/service"
)

func UserResponseFromDTO(user service.UserDTO) UserResponse {
	return UserResponse{
		UUID:      user.UUID,
		Email:     user.Email,
		Name:      user.Name,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
}

type CreateUserRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Name     string `json:"name" validate:"required,min=1,max=100"`
	Password string `json:"password" validate:"required,min=6"`
}

type UpdateUserRequest struct {
	Email *string `json:"email,omitempty" validate:"omitempty,email"`
	Name  *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
}

type UserResponse struct {
	UUID      string `json:"uuid"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ListUsersRequest struct {
	Limit  int `json:"limit" query:"limit" validate:"required,min=1,max=100"`
	Offset int `json:"offset" query:"offset" validate:"min=0"`
}

type ListUsersResponse struct {
	Users  []UserResponse `json:"users"`
	Total  int64          `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}
