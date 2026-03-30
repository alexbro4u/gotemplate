package user

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/alexbro4u/gotemplate/internal/dto/controller"
	"github.com/alexbro4u/gotemplate/internal/dto/service"
	usersvc "github.com/alexbro4u/gotemplate/internal/layers/services/user"

	"github.com/go-playground/validator/v10"
)

type Deps struct {
	UserService *usersvc.Service    `validate:"required"`
	Validator   *validator.Validate `validate:"required"`
}

type Controller struct {
	userService *usersvc.Service
	validator   *validator.Validate
}

func New(deps Deps) (*Controller, error) {
	if err := deps.Validator.Struct(deps); err != nil {
		return nil, err
	}

	return &Controller{
		userService: deps.UserService,
		validator:   deps.Validator,
	}, nil
}

func (c *Controller) Create(ctx echo.Context) error {
	var req controller.CreateUserRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.validator.Struct(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation failed")
	}

	output, err := c.userService.Create(ctx.Request().Context(), service.CreateUserInput{
		Email:    req.Email,
		Name:     req.Name,
		Password: req.Password,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusCreated, controller.UserResponseFromDTO(output.User))
}

func (c *Controller) Get(ctx echo.Context) error {
	targetUUID, err := getPathUUID(ctx)
	if err != nil {
		return err
	}

	output, err := c.userService.Get(ctx.Request().Context(), service.GetUserInput{
		UUID: targetUUID,
	})
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, controller.UserResponseFromDTO(output.User))
}

func (c *Controller) Update(ctx echo.Context) error {
	targetUUID, err := getPathUUID(ctx)
	if err != nil {
		return err
	}

	var req controller.UpdateUserRequest
	if err := ctx.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
	}

	if err := c.validator.Struct(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation failed")
	}

	err = c.userService.Update(ctx.Request().Context(), service.UpdateUserInput{
		UUID:  targetUUID,
		Email: req.Email,
		Name:  req.Name,
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (c *Controller) Delete(ctx echo.Context) error {
	targetUUID, err := getPathUUID(ctx)
	if err != nil {
		return err
	}

	err = c.userService.Delete(ctx.Request().Context(), service.DeleteUserInput{
		UUID: targetUUID,
	})
	if err != nil {
		return err
	}

	return ctx.NoContent(http.StatusNoContent)
}

func (c *Controller) List(ctx echo.Context) error {
	var req controller.ListUsersRequest

	// Try to bind from query parameters first, fallback to JSON body
	if err := (&echo.DefaultBinder{}).BindQueryParams(ctx, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid query parameters")
	}

	// If no query params, try JSON body
	if req.Limit == 0 && req.Offset == 0 {
		if err := ctx.Bind(&req); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid request body")
		}
	}

	// Set defaults if still empty
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Offset == 0 {
		req.Offset = 0
	}

	if err := c.validator.Struct(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "validation failed")
	}

	output, err := c.userService.List(ctx.Request().Context(), service.ListUsersInput{
		Limit:  req.Limit,
		Offset: req.Offset,
	})
	if err != nil {
		return err
	}

	users := make([]controller.UserResponse, len(output.Users))
	for i, user := range output.Users {
		users[i] = controller.UserResponseFromDTO(user)
	}

	return ctx.JSON(http.StatusOK, controller.ListUsersResponse{
		Users:  users,
		Total:  output.Total,
		Limit:  output.Limit,
		Offset: output.Offset,
	})
}

func getPathUUID(ctx echo.Context) (uuid.UUID, error) {
	uuidStr := ctx.Param("uuid")
	parsed, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil, echo.NewHTTPError(http.StatusBadRequest, "invalid uuid")
	}
	return parsed, nil
}
