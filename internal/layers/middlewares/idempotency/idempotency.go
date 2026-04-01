package idempotency

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"log/slog"
	"net/http"
	"time"

	stderrors "errors"

	"github.com/alexbro4u/gotemplate/internal/dto/repository"
	apperrors "github.com/alexbro4u/gotemplate/internal/errors"
	"github.com/alexbro4u/gotemplate/internal/layers/repositories"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
)

const (
	IdempotencyKeyHeader = "Idempotency-Key"
)

type Middleware struct {
	repos   repositories.RequestCacheRepository
	db      *sqlx.DB
	ttlDays int
	cache   *InMemoryCache
	logger  *slog.Logger
}

func New(repos *repositories.Repositories, db *sqlx.DB, ttlDays int, maxCacheEntries int) *Middleware {
	return &Middleware{
		repos:   repos.RequestCache,
		db:      db,
		ttlDays: ttlDays,
		cache:   NewInMemoryCache(maxCacheEntries),
		logger:  slog.Default(),
	}
}

func (m *Middleware) SetLogger(logger *slog.Logger) {
	m.logger = logger
}

func (m *Middleware) Middleware() echo.MiddlewareFunc { //nolint:funlen,gocognit // idempotency middleware is inherently complex
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Получаем user_id из контекст
			userIDStr, ok := c.Get("user_uuid").(string)
			if !ok || userIDStr == "" {
				// Если нет user_id, пропускаем middleware (публичные endpoints)
				return next(c)
			}

			// Идемпотентность только для мутирующих методов
			method := c.Request().Method
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				return next(c)
			}

			userID, err := uuid.Parse(userIDStr)
			if err != nil {
				return echo.NewHTTPError(http.StatusBadRequest, "invalid user uuid")
			}

			idempotencyKey := c.Request().Header.Get(IdempotencyKeyHeader)
			if idempotencyKey == "" {
				// Без заголовка идемпотентность не применяем — не кэшируем и не создаём записи
				return next(c)
			}

			c.Response().Header().Set(IdempotencyKeyHeader, idempotencyKey)

			// Фактический путь запроса (не шаблон маршрута), чтобы /users/aaa и /users/bbb не коллидировали
			path := c.Request().URL.Path
			httpVerb := c.Request().Method

			cacheKey := CacheKey{
				UserID:    userID,
				Path:      path,
				HTTPVerb:  httpVerb,
				RequestID: idempotencyKey,
			}

			// Вся работа с кэшем в одной транзакции
			tx, err := m.db.BeginTxx(c.Request().Context(), nil)
			if err != nil {
				m.logger.Warn("failed to begin tx for idempotency", "idempotency_key", idempotencyKey, "error", err)
				return echo.NewHTTPError(http.StatusServiceUnavailable, "could not acquire lock")
			}
			defer func() {
				if rerr := tx.Rollback(); rerr != nil && !stderrors.Is(rerr, sql.ErrTxDone) {
					m.logger.Warn("idempotency tx rollback", "error", rerr)
				}
			}()

			lockKey := m.getAdvisoryLockKey(idempotencyKey)
			if lockErr := m.acquireXactLock(c.Request().Context(), tx, lockKey); lockErr != nil {
				m.logger.Warn("failed to acquire xact lock", "idempotency_key", idempotencyKey, "error", lockErr)
				return echo.NewHTTPError(http.StatusServiceUnavailable, "could not acquire lock")
			}

			// Проверяем кэш по полной комбинации (userID + path + verb + requestID)
			cacheValue, exists := m.getCachedByKey(c.Request().Context(), cacheKey)
			if exists {
				_ = tx.Commit()
				return m.writeCachedResponse(c, cacheValue)
			}

			// Кэш не найден - выполняем запрос и сохраняем ответ
			responseBody := &bytes.Buffer{}
			responseWriter := &responseWriter{
				ResponseWriter: c.Response().Writer,
				body:           responseBody,
				statusCode:     http.StatusOK,
			}
			c.Response().Writer = responseWriter

			err = next(c)
			if err != nil {
				// Не кэшируем при ошибке
				return err
			}

			statusCode := c.Response().Status
			if statusCode == 0 {
				statusCode = responseWriter.statusCode
			}
			if statusCode == 0 {
				statusCode = http.StatusOK
			}

			contentType := c.Response().Header().Get("Content-Type")
			if contentType == "" {
				contentType = "application/json"
			}

			// Сохраняем ответ в кэш для всех статус кодов (включая 4xx/5xx, но только если ответ уже записан)
			var responseBytes []byte
			if responseBody.Len() > 0 {
				responseBytes = make([]byte, responseBody.Len())
				copy(responseBytes, responseBody.Bytes())
			}

			cacheValue = &CacheValue{
				Response:    responseBytes,
				StatusCode:  statusCode,
				ContentType: contentType,
				CreatedAt:   time.Now(),
			}

			m.cache.Set(cacheKey, cacheValue)

			expiresAt := time.Now().AddDate(0, 0, m.ttlDays)
			if createErr := m.repos.Create(c.Request().Context(), repository.CreateRequestCacheInput{
				UserID:      userID,
				Path:        path,
				HTTPVerb:    httpVerb,
				RequestID:   idempotencyKey,
				Response:    responseBytes,
				StatusCode:  statusCode,
				ContentType: contentType,
				ExpiresAt:   expiresAt,
			}); createErr != nil {
				m.logger.Warn("failed to save request cache to DB",
					"error", createErr,
					"idempotency_key", idempotencyKey,
					"user_id", userID.String(),
					"path", path,
					"status_code", statusCode,
					"body_len", len(responseBytes),
				)
				return createErr
			}
			if commitErr := tx.Commit(); commitErr != nil {
				m.logger.Warn("failed to commit idempotency tx", "error", commitErr)
				return echo.NewHTTPError(http.StatusInternalServerError, "failed to commit")
			}

			return nil
		}
	}
}

type responseWriter struct {
	http.ResponseWriter
	body        *bytes.Buffer
	statusCode  int
	wroteHeader bool
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	rw.body.Write(b)
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.wroteHeader {
		return
	}
	rw.wroteHeader = true
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func advisoryLockKey(key string) int64 {
	h := sha256.Sum256([]byte(key))
	return int64(binary.LittleEndian.Uint64(h[:8])) //nolint:gosec // advisory lock key, overflow is acceptable
}

func (m *Middleware) getAdvisoryLockKey(idempotencyKey string) int64 {
	return advisoryLockKey(idempotencyKey)
}

func (m *Middleware) acquireXactLock(ctx context.Context, tx *sqlx.Tx, lockKey int64) error {
	_, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, lockKey)
	return err
}

func (m *Middleware) getCachedByKey(ctx context.Context, cacheKey CacheKey) (*CacheValue, bool) {
	value, exists := m.cache.Get(cacheKey)
	if exists {
		return value, true
	}

	output, err := m.repos.Get(ctx, repository.GetRequestCacheInput{
		UserID:    cacheKey.UserID,
		Path:      cacheKey.Path,
		HTTPVerb:  cacheKey.HTTPVerb,
		RequestID: cacheKey.RequestID,
	})
	if err != nil {
		if apperrors.CodeIs(err, apperrors.CodeUserNotFound) {
			return nil, false
		}
		m.logger.WarnContext(ctx, "failed to query request cache", "error", err, "cache_key", cacheKey)
		return nil, false
	}

	rc := output.RequestCache
	value = &CacheValue{
		Response:    rc.Response,
		StatusCode:  rc.StatusCode,
		ContentType: rc.ContentType,
		CreatedAt:   rc.CreatedAt,
	}
	m.cache.Set(cacheKey, value)
	return value, true
}

func (m *Middleware) writeCachedResponse(c echo.Context, value *CacheValue) error {
	c.Response().Status = value.StatusCode
	if value.StatusCode == http.StatusNoContent {
		return c.NoContent(value.StatusCode)
	}
	contentType := value.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	c.Response().Header().Set("Content-Type", contentType)
	return c.Blob(value.StatusCode, contentType, value.Response)
}

// CleanupOld удаляет старые записи из in-memory кэша.
func (m *Middleware) CleanupOld(ctx context.Context) error {
	before := time.Now().AddDate(0, 0, -m.ttlDays)
	m.cache.DeleteOld(before)

	if err := m.repos.DeleteExpired(ctx); err != nil {
		return err
	}
	return nil
}
