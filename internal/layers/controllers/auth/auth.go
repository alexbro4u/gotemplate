package auth

import (
	"net/http"
	"time"

	"github.com/alexbro4u/gotemplate/internal/dto/controller"
	"github.com/alexbro4u/gotemplate/internal/dto/service"
	authsvc "github.com/alexbro4u/gotemplate/internal/layers/services/auth"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Deps struct {
	AuthService *authsvc.Service    `validate:"required"`
	Validator   *validator.Validate `validate:"required"`
}

type Controller struct {
	authService *authsvc.Service
	validator   *validator.Validate
}

func New(deps Deps) (*Controller, error) {
	if err := deps.Validator.Struct(deps); err != nil {
		return nil, err
	}

	return &Controller{
		authService: deps.AuthService,
		validator:   deps.Validator,
	}, nil
}

func (c *Controller) Register(ctx echo.Context) error {
	var req controller.RegisterRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.validator.Struct(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation failed")
	}

	output, err := c.authService.Register(ctx.Request().Context(), service.RegisterInput{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, controller.AuthResponse{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
		User:         controller.UserResponseFromDTO(output.User),
	})
}

func (c *Controller) Login(ctx echo.Context) error {
	var req controller.LoginRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.validator.Struct(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation failed")
	}

	output, err := c.authService.Login(ctx.Request().Context(), service.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, controller.AuthResponse{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
		User:         controller.UserResponseFromDTO(output.User),
	})
}

func (c *Controller) Refresh(ctx echo.Context) error {
	var req controller.RefreshRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.validator.Struct(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation failed")
	}

	output, err := c.authService.Refresh(ctx.Request().Context(), service.RefreshInput{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, controller.TokenResponse{
		AccessToken:  output.AccessToken,
		RefreshToken: output.RefreshToken,
	})
}

func (c *Controller) GetMe(ctx echo.Context) error {
	userUUID, err := getUserUUID(ctx)
	if err != nil {
		return err
	}

	output, err := c.authService.GetMe(ctx.Request().Context(), service.GetMeInput{
		UserUUID: userUUID,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, controller.UserResponseFromDTO(output.User))
}

func (c *Controller) UpdateMe(ctx echo.Context) error {
	userUUID, err := getUserUUID(ctx)
	if err != nil {
		return err
	}

	var req controller.UpdateMeRequest
	if bindErr := ctx.Bind(&req); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if validErr := c.validator.Struct(req); validErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation failed")
	}

	err = c.authService.UpdateMe(ctx.Request().Context(), service.UpdateMeInput{
		UserUUID: userUUID,
		Email:    req.Email,
		Name:     req.Name,
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (c *Controller) ChangePassword(ctx echo.Context) error {
	userUUID, err := getUserUUID(ctx)
	if err != nil {
		return err
	}

	var req controller.ChangePasswordRequest
	if bindErr := ctx.Bind(&req); bindErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if validErr := c.validator.Struct(req); validErr != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation failed")
	}

	err = c.authService.ChangePassword(ctx.Request().Context(), service.ChangePasswordInput{
		UserUUID:    userUUID,
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (c *Controller) Logout(ctx echo.Context) error {
	jti, _ := ctx.Get("token_jti").(string)
	expiresAt, _ := ctx.Get("token_expires_at").(time.Time)
	if jti == "" {
		return ctx.NoContent(http.StatusNoContent)
	}
	if err := c.authService.Logout(ctx.Request().Context(), service.LogoutInput{
		JTI:       jti,
		ExpiresAt: expiresAt,
	}); err != nil {
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}

func (c *Controller) RequestPasswordReset(ctx echo.Context) error {
	var req controller.RequestPasswordResetRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.validator.Struct(req); err != nil {
		return err
	}
	out, err := c.authService.RequestPasswordReset(ctx.Request().Context(), service.RequestPasswordResetInput{
		Email: req.Email,
	})
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, controller.PasswordResetTokenResponse{Token: out.Token})
}

func (c *Controller) ConfirmPasswordReset(ctx echo.Context) error {
	var req controller.ConfirmPasswordResetRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}
	if err := c.validator.Struct(req); err != nil {
		return err
	}
	if err := c.authService.ConfirmPasswordReset(ctx.Request().Context(), service.ConfirmPasswordResetInput{
		Token:       req.Token,
		NewPassword: req.NewPassword,
	}); err != nil {
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}

func getUserUUID(ctx echo.Context) (uuid.UUID, error) {
	uuidStr, ok := ctx.Get("user_uuid").(string)
	if !ok || uuidStr == "" {
		return uuid.Nil, echo.NewHTTPError(http.StatusUnauthorized, "missing user context")
	}
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "invalid user uuid")
	}
	return parsed, nil
}
