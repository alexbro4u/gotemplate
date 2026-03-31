package http

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/alexbro4u/gotemplate/internal/core/requestid"
	"github.com/labstack/echo/v4"
)

// requestIDMiddleware generates or extracts a request ID and stores it in context.
func requestIDMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			rid := c.Request().Header.Get(requestid.Header)
			if rid == "" {
				rid = requestid.New()
			}

			c.Response().Header().Set(requestid.Header, rid)

			ctx := requestid.WithContext(c.Request().Context(), rid)
			c.SetRequest(c.Request().WithContext(ctx))

			return next(c)
		}
	}
}

// requestLoggingMiddleware logs every HTTP request with method, path, status, duration, request_id.
func requestLoggingMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			duration := time.Since(start)
			status := c.Response().Status
			rid := requestid.FromContext(c.Request().Context())

			attrs := []slog.Attr{
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.Int("status", status),
				slog.Duration("duration", duration),
				slog.String("remote_ip", c.RealIP()),
			}
			if rid != "" {
				attrs = append(attrs, slog.String("request_id", rid))
			}

			level := slog.LevelInfo
			if status >= http.StatusInternalServerError {
				level = slog.LevelError
			} else if status >= http.StatusBadRequest {
				level = slog.LevelWarn
			}

			logger.LogAttrs(c.Request().Context(), level, "http request", attrs...)

			return nil
		}
	}
}
