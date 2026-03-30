package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	DefaultMaxHeaderBytes    = 1 << 20
	DefaultReadTimeout       = 30 * time.Second
	DefaultReadHeaderTimeout = 30 * time.Second
	DefaultWriteTimeout      = 30 * time.Second
	DefaultIdleTimeout       = 30 * time.Second
)

type Config struct {
	Logger            *slog.Logger
	Host              string
	Port              string
	Handler           http.Handler
	MaxHeaderBytes    int
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

func (c *Config) SetDefault(defaults Config) {
	if c.Handler == nil {
		c.Handler = defaults.Handler
	}
	if c.MaxHeaderBytes == 0 {
		c.MaxHeaderBytes = defaults.MaxHeaderBytes
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = defaults.ReadTimeout
	}
	if c.ReadHeaderTimeout == 0 {
		c.ReadHeaderTimeout = defaults.ReadHeaderTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = defaults.WriteTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = defaults.IdleTimeout
	}
}

type Option func(*echo.Echo) error

type Server struct {
	logger *slog.Logger
	server *http.Server
	echo   *echo.Echo
}

func New(config Config, options ...Option) (*Server, error) {
	var echoInstance *echo.Echo

	if config.Handler == nil {
		echoInstance = echo.New()
		config.Handler = echoInstance
	} else {
		if e, ok := config.Handler.(*echo.Echo); ok {
			echoInstance = e
		} else {
			return nil, fmt.Errorf("handler must be *echo.Echo")
		}
	}

	for _, option := range options {
		if err := option(echoInstance); err != nil {
			return nil, fmt.Errorf("apply option: %w", err)
		}
	}

	config.SetDefault(Config{
		Handler:           echoInstance,
		MaxHeaderBytes:    DefaultMaxHeaderBytes,
		ReadTimeout:       DefaultReadTimeout,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
		WriteTimeout:      DefaultWriteTimeout,
		IdleTimeout:       DefaultIdleTimeout,
	})

	return &Server{
		echo:   echoInstance,
		logger: config.Logger,
		server: &http.Server{
			Addr:              fmt.Sprintf("%s:%s", config.Host, config.Port),
			Handler:           config.Handler,
			MaxHeaderBytes:    config.MaxHeaderBytes,
			ReadTimeout:       config.ReadTimeout,
			ReadHeaderTimeout: config.ReadHeaderTimeout,
			WriteTimeout:      config.WriteTimeout,
			IdleTimeout:       config.IdleTimeout,
		},
	}, nil
}

func (s *Server) Run() error {
	s.logger.Info("starting http server", slog.String("address", s.server.Addr))

	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error("http server error", slog.Any("error", err))
		return err
	}
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	s.logger.Info("shutting down http server")

	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}

	return nil
}

func (s *Server) Echo() *echo.Echo {
	return s.echo
}
