package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alexbro4u/gotemplate/pkg/env"
	"github.com/go-playground/validator/v10"
)

type APP struct {
	LogLevel           string `json:"LOG_LEVEL" env:"LOG_LEVEL,default=info"`
	AppEnv             string `json:"APP_ENV" env:"APP_ENV,required" validate:"oneof=test prod dev"`
	ShutdownTimeoutSec int    `json:"APP_SHUTDOWN_TIMEOUT_SEC" env:"APP_SHUTDOWN_TIMEOUT_SEC,default=5"`
}

type HTTP struct {
	Host               string `json:"HTTP_HOST" env:"HTTP_HOST,required"`
	Port               string `json:"HTTP_PORT" env:"HTTP_PORT,required"`
	SecretKey          string `json:"HTTP_SECRET_KEY" env:"HTTP_SECRET_KEY,required"`
	CorsAllowedOrigins string `json:"HTTP_CORS_ALLOWED_ORIGINS" env:"HTTP_CORS_ALLOWED_ORIGINS,required"`
	// Rate limiter для /auth
	AuthRateLimitRate       float64 `json:"HTTP_AUTH_RATE_LIMIT_RATE" env:"HTTP_AUTH_RATE_LIMIT_RATE,default=10"`
	AuthRateLimitBurst      int     `json:"HTTP_AUTH_RATE_LIMIT_BURST" env:"HTTP_AUTH_RATE_LIMIT_BURST,default=20"`
	AuthRateLimitExpiresSec int     `json:"HTTP_AUTH_RATE_LIMIT_EXPIRES_SEC" env:"HTTP_AUTH_RATE_LIMIT_EXPIRES_SEC,default=60"`
	// Открытая регистрация (true = любой может зарегистрироваться, false = регистрация отключена)
	RegistrationEnabled bool `json:"HTTP_REGISTRATION_ENABLED" env:"HTTP_REGISTRATION_ENABLED,default=true"`
	RequestTimeoutSec   int  `json:"HTTP_REQUEST_TIMEOUT_SEC" env:"HTTP_REQUEST_TIMEOUT_SEC,default=30"`
}

type Postgres struct {
	Host     string `json:"POSTGRES_HOST" env:"POSTGRES_HOST,required"`
	Port     string `json:"POSTGRES_PORT" env:"POSTGRES_PORT,required"`
	User     string `json:"POSTGRES_USER" env:"POSTGRES_USER,required"`
	Password string `json:"POSTGRES_PASSWORD" env:"POSTGRES_PASSWORD,required"`
	DB       string `json:"POSTGRES_DB" env:"POSTGRES_DB,required"`
	SSLMode  string `json:"POSTGRES_SSL_MODE" env:"POSTGRES_SSL_MODE,default=disable"`

	PoolMaxConns int `json:"POSTGRES_POOL_MAX_CONNS" env:"POSTGRES_POOL_MAX_CONNS,default=20"`
	PoolMinConns int `json:"POSTGRES_POOL_MIN_CONNS" env:"POSTGRES_POOL_MIN_CONNS,default=4"`
}

type Metrics struct {
	Namespace   string `json:"METRICS_NAMESPACE" env:"METRICS_NAMESPACE"`
	Subsystem   string `json:"METRICS_SUBSYSTEM" env:"METRICS_SUBSYSTEM"`
	ConstLabels string `json:"METRICS_CONST_LABELS" env:"METRICS_CONST_LABELS"`
}

type Idempotency struct {
	TTLDays            int `json:"IDEMPOTENCY_TTL_DAYS" env:"IDEMPOTENCY_TTL_DAYS"`
	MaxCacheEntries    int `json:"IDEMPOTENCY_MAX_CACHE_ENTRIES" env:"IDEMPOTENCY_MAX_CACHE_ENTRIES,default=10000"`
	CleanupIntervalMin int `json:"IDEMPOTENCY_CLEANUP_INTERVAL_MIN" env:"IDEMPOTENCY_CLEANUP_INTERVAL_MIN,default=60"`
}

type Jaeger struct {
	URL      string `json:"JAEGER_URL" env:"JAEGER_URL"`
	AppName  string `json:"JAEGER_APP_NAME" env:"JAEGER_APP_NAME"`
	Insecure bool   `json:"JAEGER_INSECURE" env:"JAEGER_INSECURE"`
}

type Config struct {
	APP         APP         `validate:"required"`
	HTTP        HTTP        `validate:"required"`
	Postgres    Postgres    `validate:"required"`
	Metrics     Metrics     `validate:"required"`
	Jaeger      Jaeger      `validate:"required"`
	Idempotency Idempotency `validate:"required"`
}

func New() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(".env", cfg); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			msgs := make([]string, 0, len(ve))
			for _, fe := range ve {
				msgs = append(msgs, fmt.Sprintf("  %s: failed '%s' validation", fe.Field(), fe.Tag()))
			}
			return nil, fmt.Errorf("config validation failed:\n%s", strings.Join(msgs, "\n"))
		}
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return cfg, nil
}
