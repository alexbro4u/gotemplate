package requestid

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

type ctxKey struct{}

const Header = "X-Request-ID"

// New generates a new request ID.
func New() string {
	return uuid.New().String()
}

// WithContext returns a new context with the given request ID.
func WithContext(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, ctxKey{}, requestID)
}

// FromContext extracts the request ID from context. Returns empty string if not set.
func FromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKey{}).(string); ok {
		return v
	}
	return ""
}

// Logger returns a logger with the request_id attribute from context.
func Logger(ctx context.Context, logger *slog.Logger) *slog.Logger {
	if rid := FromContext(ctx); rid != "" {
		return logger.With(slog.String("request_id", rid))
	}
	return logger
}
