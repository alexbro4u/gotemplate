package ping

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type Controller struct{}

func New() *Controller {
	return &Controller{}
}

func (c *Controller) Ping(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{
		"message": "pong",
	})
}

func (c *Controller) Health(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, map[string]string{
		"status": "ok",
	})
}
