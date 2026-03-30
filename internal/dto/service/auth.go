package service

import (
	"github.com/google/uuid"
)

type RegisterInput struct {
	Email    string `validate:"required,email"`
	Name     string `validate:"required,min=1,max=100"`
	Password string `validate:"required,min=6"`
}

type RegisterOutput struct {
	User         UserDTO
	AccessToken  string
	RefreshToken string
}

type LoginInput struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required"`
}

type LoginOutput struct {
	User         UserDTO
	AccessToken  string
	RefreshToken string
}

type RefreshInput struct {
	RefreshToken string `validate:"required"`
}

type RefreshOutput struct {
	AccessToken  string
	RefreshToken string
}

type GetMeInput struct {
	UserUUID uuid.UUID
}

type GetMeOutput struct {
	User UserDTO
}

type UpdateMeInput struct {
	UserUUID uuid.UUID `validate:"required"`
	Email    *string   `validate:"omitempty,email"`
	Name     *string   `validate:"omitempty,min=1,max=100"`
}

type ChangePasswordInput struct {
	UserUUID    uuid.UUID `validate:"required"`
	OldPassword string    `validate:"required"`
	NewPassword string    `validate:"required,min=6"`
}
