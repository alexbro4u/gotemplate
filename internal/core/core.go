//nolint:cyclop // bootstrap package has multiple init paths by design
package core

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexbro4u/gotemplate/internal/config"
	"github.com/alexbro4u/gotemplate/internal/core/http"
	"github.com/alexbro4u/gotemplate/internal/core/jaeger"
	"github.com/alexbro4u/gotemplate/internal/core/jwt"
	"github.com/alexbro4u/gotemplate/internal/core/metrics"
	"github.com/alexbro4u/gotemplate/internal/layers/controllers"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"
	auditrepo "github.com/alexbro4u/gotemplate/internal/layers/repositories/audit"
	passwordresetrepo "github.com/alexbro4u/gotemplate/internal/layers/repositories/password_reset"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories/request_cache"
	tokenblacklistrepo "github.com/alexbro4u/gotemplate/internal/layers/repositories/token_blacklist"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories/user"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories/user_group"
	"github.com/alexbro4u/gotemplate/internal/layers/services"
	"github.com/alexbro4u/gotemplate/pkg/cache"
	"github.com/alexbro4u/gotemplate/pkg/closer"
	"github.com/alexbro4u/gotemplate/pkg/pgxtools"
	"github.com/alexbro4u/gotemplate/pkg/sqlxadapter"

	"github.com/alexbro4u/uowkit/uow"
	"github.com/go-playground/validator/v10"
)

const (
	dbConnectionTimeout = 10 * time.Second
)

func Run(cfg *config.Config) error { //nolint:funlen,gocognit // application bootstrap
	validate := validator.New()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	logger := slog.New(
		slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: parseLogLevel(cfg.APP.LogLevel),
		}),
	)

	logger.Info("starting application",
		"http_host", cfg.HTTP.Host,
		"http_port", cfg.HTTP.Port,
		"log_level", cfg.APP.LogLevel,
	)

	dbConnection, err := pgxtools.Connect(ctx, pgxtools.ConnectOptions{
		Config: pgxtools.Config{
			Host:         cfg.Postgres.Host,
			Port:         cfg.Postgres.Port,
			User:         cfg.Postgres.User,
			Password:     cfg.Postgres.Password,
			Database:     cfg.Postgres.DB,
			SSLMode:      cfg.Postgres.SSLMode,
			PoolMaxConns: cfg.Postgres.PoolMaxConns,
			PoolMinConns: cfg.Postgres.PoolMinConns,
		},
		Logger:  logger,
		Timeout: dbConnectionTimeout,
	})
	if err != nil {
		return err
	}

	metricsFactory, err := metrics.New(cfg)
	if err != nil {
		return err
	}

	// up и health устанавливаются автоматически через Echo handlers

	jwtService := jwt.New(cfg.HTTP.SecretKey)

	db := dbConnection.DB()

	userRepo, err := user.New(user.Deps{
		DB:        db,
		Validator: validate,
	})
	if err != nil {
		return err
	}

	userGroupRepo, err := usergroup.New(usergroup.Deps{
		DB:        db,
		Validator: validate,
	})
	if err != nil {
		return err
	}

	requestCacheRepo, err := requestcache.New(requestcache.Deps{
		DB:        db,
		Validator: validate,
	})
	if err != nil {
		return err
	}

	blacklistRepo, err := tokenblacklistrepo.New(tokenblacklistrepo.Deps{
		DB:        db,
		Validator: validate,
	})
	if err != nil {
		return err
	}

	auditRepo, err := auditrepo.New(auditrepo.Deps{
		DB:        db,
		Validator: validate,
	})
	if err != nil {
		return err
	}

	passwordResetRepo, err := passwordresetrepo.New(passwordresetrepo.Deps{
		DB:        db,
		Validator: validate,
	})
	if err != nil {
		return err
	}

	blacklistCache := cache.NewBlacklist()

	repositoriesInstance := &repositories.Repositories{
		User:          userRepo,
		UserGroup:     userGroupRepo,
		RequestCache:  requestCacheRepo,
		Blacklist:     blacklistRepo,
		Audit:         auditRepo,
		PasswordReset: passwordResetRepo,
	}

	txFactory := sqlxadapter.NewTxFactory(db, nil)
	unitOfWork := uow.New(txFactory, uow.WithLogger(logger))

	servicesInstance, err := services.New(services.Deps{
		Logger:        logger,
		Repositories:  repositoriesInstance,
		UoW:           unitOfWork,
		JWTService:    jwtService,
		Validator:     validate,
		Blacklist:     blacklistRepo,
		AuditRepo:     auditRepo,
		PasswordReset: passwordResetRepo,
	})
	if err != nil {
		return err
	}

	controllersInstance, err := controllers.New(controllers.Deps{
		Logger:      logger,
		AuthService: servicesInstance.Auth,
		UserService: servicesInstance.User,
		Validator:   validate,
	})
	if err != nil {
		return err
	}

	httpServer, err := http.New(ctx, http.Deps{
		Config:       cfg,
		Logger:       logger,
		Controllers:  controllersInstance,
		Metrics:      metricsFactory,
		JWTService:   jwtService,
		Repositories: repositoriesInstance,
		DB:           db,
		Validator:    validate,
		Blacklist:    blacklistCache,
	})
	if err != nil {
		return err
	}

	tracerProvider, err := jaeger.New(cfg.Jaeger, httpServer)
	if err != nil {
		return err
	}

	closerInstance := closer.New()
	closerInstance.Add(func(ctx context.Context) error {
		return httpServer.Server.Stop(ctx)
	})
	if tracerProvider != nil {
		closerInstance.Add(func(ctx context.Context) error {
			return tracerProvider.Shutdown(ctx)
		})
	}
	closerInstance.Add(func(_ context.Context) error {
		return dbConnection.Close()
	})

	logger.Info("application started successfully")

	//  Очистка кэша идемпотентности
	go httpServer.StartCleanup(ctx, time.Duration(cfg.Idempotency.CleanupIntervalMin)*time.Minute)

	// Очистка просроченных токенов из блэклиста
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				blacklistCache.Cleanup()
				if cleanupErr := blacklistRepo.DeleteExpired(ctx); cleanupErr != nil {
					logger.WarnContext(ctx, "failed to cleanup token blacklist", "error", cleanupErr)
				}
				if cleanupErr := passwordResetRepo.DeleteExpired(ctx); cleanupErr != nil {
					logger.WarnContext(ctx, "failed to cleanup password reset tokens", "error", cleanupErr)
				}
			}
		}
	}()

	serverErr := make(chan error, 1)
	go func() {
		serverErr <- httpServer.Server.Run()
	}()

	select {
	case <-ctx.Done():
	case srvErr := <-serverErr:
		if srvErr != nil {
			logger.Error("http server exited", "error", srvErr)
		}
		cancel()
	}
	<-ctx.Done()

	logger.Info("shutting down application gracefully")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), time.Duration(cfg.APP.ShutdownTimeoutSec)*time.Second)
	defer shutdownCancel()

	if closeErr := closerInstance.Close(shutdownCtx); closeErr != nil {
		return closeErr
	}

	return nil
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
