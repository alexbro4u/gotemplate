package controllers

import (
	"log/slog"

	"github.com/go-playground/validator/v10"
	"github.com/alexbro4u/gotemplate/internal/layers/controllers/auth"
	"github.com/alexbro4u/gotemplate/internal/layers/controllers/ping"
	"github.com/alexbro4u/gotemplate/internal/layers/controllers/user"
	authsvc "github.com/alexbro4u/gotemplate/internal/layers/services/auth"
	usersvc "github.com/alexbro4u/gotemplate/internal/layers/services/user"
)

type Deps struct {
	Logger      *slog.Logger        `validate:"required"`
	AuthService *authsvc.Service    `validate:"required"`
	UserService *usersvc.Service    `validate:"required"`
	Validator   *validator.Validate `validate:"required"`
}

type Controllers struct {
	Ping *ping.Controller
	Auth *auth.Controller
	User *user.Controller
}

func New(deps Deps) (*Controllers, error) {
	if err := deps.Validator.Struct(deps); err != nil {
		return nil, err
	}

	authController, err := auth.New(auth.Deps{
		AuthService: deps.AuthService,
		Validator:   deps.Validator,
	})
	if err != nil {
		return nil, err
	}

	userController, err := user.New(user.Deps{
		UserService: deps.UserService,
		Validator:   deps.Validator,
	})
	if err != nil {
		return nil, err
	}

	return &Controllers{
		Ping: ping.New(),
		Auth: authController,
		User: userController,
	}, nil
}
