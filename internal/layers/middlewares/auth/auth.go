package auth

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/alexbro4u/gotemplate/internal/core/jwt"
	"github.com/labstack/echo/v4"
)

func Middleware(jwtService *jwt.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Получаем токен из заголовка Authorization
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "missing authorization header")
			}

			// Проверяем формат "Bearer <token>"
			parts := strings.Fields(authHeader)
			if len(parts) != 2 || !strings.EqualFold("bearer", parts[0]) {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid authorization header format")
			}

			tokenString := parts[1]

			// Валидируем токен
			claims, err := jwtService.ValidateToken(tokenString)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "invalid token")
			}

			// Сохраняем claims в контекст для использования в handlers
			c.Set("user_uuid", claims.UserUUID.String())
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)
			c.Set("user_groups", groupsFromClaims(claims.Groups))

			return next(c)
		}
	}
}

// RequireRole создает middleware для проверки роли пользователя.
func RequireRole(requiredRole string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole, ok := c.Get("user_role").(string)
			if !ok || userRole != requiredRole {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}
			return next(c)
		}
	}
}

// RequireGroup создает middleware: доступ только если пользователь в указанной группе.
func RequireGroup(requiredGroup string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			groups := getUserGroups(c)
			if !contains(groups, requiredGroup) {
				return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
			}
			return next(c)
		}
	}
}

// RequireAnyGroup создает middleware: доступ только если пользователь в одной из указанных групп.
func RequireAnyGroup(allowedGroups ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			groups := getUserGroups(c)
			for _, allowed := range allowedGroups {
				if contains(groups, allowed) {
					return next(c)
				}
			}
			return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
		}
	}
}

func groupsFromClaims(groups []string) []string {
	if groups == nil {
		return []string{}
	}
	return groups
}

func getUserGroups(c echo.Context) []string {
	val := c.Get("user_groups")
	if val == nil {
		return nil
	}
	groups, ok := val.([]string)
	if !ok {
		return nil
	}
	return groups
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// GetUserUUID извлекает UUID пользователя из контекста.
func GetUserUUID(c echo.Context) (string, bool) {
	uuidVal := c.Get("user_uuid")
	if uuidVal == nil {
		return "", false
	}
	uuid, ok := uuidVal.(string)
	if !ok {
		// Если это не string, попробуем преобразовать через fmt
		return fmt.Sprintf("%v", uuidVal), true
	}
	return uuid, true
}
