package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/alexbro4u/errkit"
	"github.com/labstack/echo/v4"
)

type errorResponse struct {
	Error  string         `json:"error"`
	Code   string         `json:"code,omitempty"`
	Fields map[string]any `json:"fields,omitempty"`
}

func newErrorHandler(logger *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		// Echo's own HTTP errors (from middleware, bind, etc.)
		var he *echo.HTTPError
		if errors.As(err, &he) {
			msg := "error"
			if m, msgOk := he.Message.(string); msgOk {
				msg = m
			}
			_ = c.JSON(he.Code, errorResponse{
				Error: msg,
			})
			return
		}

		// errkit-based errors — extract code, HTTP status, fields
		var ekErr *errkit.Error
		if errors.As(err, &ekErr) {
			status := errkit.HTTPStatus(err)
			code := errkit.GetCode(err)

			resp := errorResponse{
				Error: ekErr.Error(),
				Code:  code,
			}

			if status >= http.StatusInternalServerError {
				logger.Error("internal error",
					slog.Any("error", err),
					slog.Int("status", status),
					slog.String("code", code),
				)
			}

			_ = c.JSON(status, resp)
			return
		}

		// Fallback for unknown errors
		logger.Error("unhandled error", slog.Any("error", err))
		_ = c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
			Code:  "INTERNAL",
		})
	}
}
