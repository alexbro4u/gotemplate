package services

import (
	"log/slog"

	"github.com/alexbro4u/uowkit/uow"
	"github.com/go-playground/validator/v10"
	"github.com/alexbro4u/gotemplate/internal/core/jwt"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"
	"github.com/alexbro4u/gotemplate/internal/layers/services/auth"
	"github.com/alexbro4u/gotemplate/internal/layers/services/user"
)

type Deps struct {
	Logger       *slog.Logger               `validate:"required"`
	Repositories *repositories.Repositories `validate:"required"`
	UoW          uow.UnitOfWork             `validate:"required"`
	JWTService   *jwt.Service               `validate:"required"`
	Validator    *validator.Validate        `validate:"required"`
}

type Services struct {
	User *user.Service
	Auth *auth.Service
}

func New(deps Deps) (*Services, error) {
	if err := deps.Validator.Struct(deps); err != nil {
		return nil, err
	}

	userService, err := user.New(user.Deps{
		Logger:        deps.Logger,
		UoW:           deps.UoW,
		UserRepo:      deps.Repositories.User,
		UserGroupRepo: deps.Repositories.UserGroup,
		Validator:     deps.Validator,
	})
	if err != nil {
		return nil, err
	}

	authService, err := auth.New(auth.Deps{
		Logger:        deps.Logger,
		UoW:           deps.UoW,
		UserRepo:      deps.Repositories.User,
		UserGroupRepo: deps.Repositories.UserGroup,
		JWTService:    deps.JWTService,
		Validator:     deps.Validator,
	})
	if err != nil {
		return nil, err
	}

	return &Services{
		User: userService,
		Auth: authService,
	}, nil
}
