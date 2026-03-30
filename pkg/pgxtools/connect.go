package pgxtools

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Config struct {
	Host         string
	Port         string
	User         string
	Password     string
	Database     string
	SSLMode      string
	PoolMaxConns int
	PoolMinConns int
}

type ConnectOptions struct {
	Config  Config
	Logger  *slog.Logger
	Timeout time.Duration
}

type Connection struct {
	db *sqlx.DB
}

func Connect(ctx context.Context, options ConnectOptions) (*Connection, error) {
	logger := options.Logger

	postgresURI := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(options.Config.User, options.Config.Password),
		Host:     net.JoinHostPort(options.Config.Host, options.Config.Port),
		Path:     "/" + options.Config.Database,
		RawQuery: buildQuery(options.Config),
	}

	safeURI := fmt.Sprintf("postgres://%s@%s/%s?%s",
		options.Config.User,
		net.JoinHostPort(options.Config.Host, options.Config.Port),
		options.Config.Database,
		buildQuery(options.Config),
	)

	timeoutExceeded := time.After(options.Timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf(
				"db connection <%s> cancelled: %w",
				safeURI,
				ctx.Err(),
			)
		case <-timeoutExceeded:
			return nil, fmt.Errorf(
				"db connection <%s> failed after %v timeout: %w",
				safeURI,
				options.Timeout,
				context.DeadlineExceeded,
			)
		case <-time.After(1 * time.Second):
			db, err := sqlx.ConnectContext(ctx, "postgres", postgresURI.String())
			if err != nil {
				logger.Error(
					"failed to connect to db",
					slog.String("host", options.Config.Host),
					slog.String("port", options.Config.Port),
					slog.String("database", options.Config.Database),
					slog.String("user", options.Config.User),
					slog.String("error", err.Error()),
				)
				continue
			}

			db.SetMaxOpenConns(options.Config.PoolMaxConns)
			db.SetMaxIdleConns(options.Config.PoolMinConns)
			db.SetConnMaxLifetime(time.Hour)

			if err := db.PingContext(ctx); err != nil {
				logger.Error("failed to ping", slog.String("error", err.Error()))
				db.Close()
				continue
			}

			logger.Info("successfully connected to database",
				slog.String("host", options.Config.Host),
				slog.String("database", options.Config.Database),
			)

			return &Connection{
				db: db,
			}, nil
		}
	}
}

func (c *Connection) DB() *sqlx.DB {
	return c.db
}

func (c *Connection) Close() error {
	return c.db.Close()
}

func buildQuery(config Config) string {
	queries := make(url.Values, 1)

	if config.SSLMode != "" {
		queries["sslmode"] = []string{config.SSLMode}
	}

	return queries.Encode()
}
