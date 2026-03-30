package http

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"git.ptb.bet/public-group/shared/v2/pkg/metrics"

	"github.com/jmoiron/sqlx"
	"github.com/alexbro4u/gotemplate/internal/config"
	"github.com/alexbro4u/gotemplate/internal/core/jwt"
	coremetrics "github.com/alexbro4u/gotemplate/internal/core/metrics"
	"github.com/alexbro4u/gotemplate/internal/layers/controllers"
	"github.com/alexbro4u/gotemplate/internal/layers/middlewares/auth"
	"github.com/alexbro4u/gotemplate/internal/layers/middlewares/idempotency"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"
	"github.com/alexbro4u/gotemplate/pkg/echotools/server"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

type Deps struct {
	Config       *config.Config             `validate:"required"`
	Logger       *slog.Logger               `validate:"required"`
	Controllers  *controllers.Controllers   `validate:"required"`
	Metrics      *coremetrics.Factory       `validate:"required"`
	JWTService   *jwt.Service               `validate:"required"`
	Repositories *repositories.Repositories `validate:"required"`
	DB           *sqlx.DB                   `validate:"required"`
	Validator    *validator.Validate        `validate:"required"`
}

type Server struct {
	*server.Server
	idempotencyMiddleware *idempotency.Middleware
	logger                *slog.Logger
}

func New(ctx context.Context, deps Deps) (*Server, error) {
	if err := deps.Validator.Struct(deps); err != nil {
		return nil, err
	}

	echoInstance := echo.New()
	echoInstance.HTTPErrorHandler = newErrorHandler(deps.Logger)

	idempotencyMiddleware := idempotency.New(deps.Repositories, deps.DB, deps.Config.Idempotency.TTLDays, deps.Config.Idempotency.MaxCacheEntries)
	idempotencyMiddleware.SetLogger(deps.Logger)

	// Порядок важен
	echoInstance.Use(
		middleware.Recover(),
		requestIDMiddleware(),
		requestLoggingMiddleware(deps.Logger),
		corsFromConfig(deps.Config.HTTP.CorsAllowedOrigins),
		deps.Metrics.HTTPMetrics.EchoMiddleware(),
	)

	registerRoutes(echoInstance, deps.Controllers, deps.Metrics, deps.JWTService, deps.Repositories, deps.Config, idempotencyMiddleware)

	httpServer, err := server.New(
		server.Config{
			Logger:  deps.Logger,
			Host:    deps.Config.HTTP.Host,
			Port:    deps.Config.HTTP.Port,
			Handler: echoInstance,
		},
	)
	if err != nil {
		return nil, err
	}

	return &Server{
		Server:                httpServer,
		idempotencyMiddleware: idempotencyMiddleware,
		logger:                deps.Logger,
	}, nil
}

func (s *Server) StartCleanup(ctx context.Context) {
	// Очищаем кэш каждый час
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.idempotencyMiddleware.CleanupOld(ctx); err != nil {
				s.logger.Warn("failed to cleanup idempotency cache", "error", err)
			} else {
				s.logger.Debug("idempotency cache cleanup completed")
			}
		}
	}
}

func corsFromConfig(allowedOriginsCSV string) echo.MiddlewareFunc {
	origins := strings.Split(allowedOriginsCSV, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	return middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: origins,
		AllowMethods: []string{"GET", "HEAD", "PUT", "PATCH", "POST", "DELETE", "OPTIONS"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	})
}

func registerRoutes(e *echo.Echo, controllers *controllers.Controllers, metricsFactory *coremetrics.Factory, jwtService *jwt.Service, repos *repositories.Repositories, cfg *config.Config, idempotencyMiddleware *idempotency.Middleware) {
	// Infra endpoints (без версии)
	e.GET("/ping", controllers.Ping.Ping)
	e.GET(metrics.HealthEndpointPath, metricsFactory.EchoHealthHandler())
	e.GET(metrics.MetricsEndpointPath, metricsFactory.EchoMetricsHandler())
	e.GET(metrics.UpEndpointPath, metricsFactory.EchoUpHandler())

	// API v1
	v1 := e.Group("/api/v1")

	authGroup := v1.Group("/auth")
	authRateLimitRate := cfg.HTTP.AuthRateLimitRate
	authRateLimitBurst := cfg.HTTP.AuthRateLimitBurst
	authRateLimitExpiresIn := time.Duration(cfg.HTTP.AuthRateLimitExpiresSec) * time.Second
	authGroup.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
			Rate:      rate.Limit(authRateLimitRate),
			Burst:     authRateLimitBurst,
			ExpiresIn: authRateLimitExpiresIn,
		}),
		DenyHandler: func(c echo.Context, identifier string, err error) error {
			return c.JSON(http.StatusTooManyRequests, map[string]string{"message": "too many requests"})
		},
	}))
	if cfg.HTTP.RegistrationEnabled {
		authGroup.POST("/register", controllers.Auth.Register)
	}
	authGroup.POST("/login", controllers.Auth.Login)
	authGroup.POST("/refresh", controllers.Auth.Refresh)

	protected := v1.Group("", auth.Middleware(jwtService))
	protected.Use(idempotencyMiddleware.Middleware())

	// Self-service: любой авторизованный пользователь
	protected.GET("/me", controllers.Auth.GetMe)
	protected.PATCH("/me", controllers.Auth.UpdateMe)
	protected.POST("/me/password", controllers.Auth.ChangePassword)

	// только admin
	users := protected.Group("/users", auth.RequireRole("admin"))
	users.POST("", controllers.User.Create)
	users.GET("", controllers.User.List)
	users.GET("/:uuid", controllers.User.Get)
	users.PATCH("/:uuid", controllers.User.Update)
	users.DELETE("/:uuid", controllers.User.Delete)

	// только admin (роль + группа)
	admin := protected.Group("/admin", auth.RequireRole("admin"), auth.RequireGroup("admin"))
	admin.GET("", controllers.Ping.Ping)
}
