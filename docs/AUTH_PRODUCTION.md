# Аутентификация: Simple Mode vs Production Mode

## Simple Mode (встроенный auth)

Шаблон включает встроенную JWT-аутентификацию, которая подходит для:
- Internal/B2B сервисов
- Прототипов и MVP
- Сервисов с небольшим количеством пользователей

### Возможности
- Регистрация и логин (`/auth/register`, `/auth/login`)
- JWT токены (HS256, 24h TTL)
- Role-based access control (user/admin)
- Group-based access control
- Self-service (`GET /me`, `PATCH /me`, `POST /me/password`)
- Управление через конфиг (`HTTP_REGISTRATION_ENABLED`)

### Ограничения
- Нет refresh-токенов
- Нет 2FA/MFA
- Нет email verification / password reset
- Нет SSO/OIDC
- Нет token revocation

---

## Production Mode: Ory Kratos

[Ory Kratos](https://www.ory.sh/kratos/) — headless identity management, написан на Go, self-hosted.

### Почему Kratos
- **Headless** — только API, без встроенного UI (вы контролируете UX)
- **Go-native** — single binary, минимальные ресурсы
- **PostgreSQL** — использует тот же инстанс БД
- **Self-service flows** — registration, login, recovery, verification, settings
- **MFA/2FA** — TOTP, WebAuthn из коробки
- **Webhooks** — интеграция с вашим сервисом

### Интеграция с шаблоном

1. **Убрать** встроенный auth: удалить `/auth/*` маршруты, `internal/core/jwt`, auth middleware
2. **Добавить** Kratos в `docker-compose.yml`:
```yaml
kratos:
  image: oryd/kratos:v1.3.0
  environment:
    - DSN=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
  volumes:
    - ./kratos:/etc/config/kratos
  ports:
    - "4433:4433"  # public API
    - "4434:4434"  # admin API
```

3. **Auth middleware** — заменить JWT-валидацию на проверку сессии Kratos:
```go
func KratosMiddleware(kratosPublicURL string) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            cookie := c.Request().Header.Get("Cookie")
            resp, err := http.Get(kratosPublicURL + "/sessions/whoami")
            // проверить сессию, извлечь identity
            // установить user_uuid, email, role в контекст
            return next(c)
        }
    }
}
```

4. **Self-service** — `/me` эндпоинты заменяются на Kratos self-service flows

### Конфигурация Kratos (`kratos/kratos.yml`)
```yaml
identity:
  default_schema_id: default
  schemas:
    - id: default
      url: file:///etc/config/kratos/identity.schema.json

selfservice:
  default_browser_return_url: http://localhost:3000/
  flows:
    registration:
      enabled: true
    login:
      lifespan: 24h
    verification:
      enabled: true
    recovery:
      enabled: true
```

---

## Production Mode: Zitadel

[Zitadel](https://zitadel.com/) — полноценный IAM с UI, OIDC/OAuth2, multi-tenancy.

### Почему Zitadel
- **Полный IAM** — SSO, OIDC, OAuth2, SAML
- **Multi-tenant** — организации, проекты, роли
- **Встроенный UI** — login, registration, account management
- **Go-native** — написан на Go
- **Audit log** — полная история действий

### Когда выбрать Zitadel вместо Kratos
- Нужен SSO/OIDC провайдер
- Multi-tenant SaaS
- Enterprise требования (audit, compliance)
- Нужен готовый UI без разработки

### Интеграция с шаблоном

1. **Добавить** Zitadel в `docker-compose.yml`:
```yaml
zitadel:
  image: ghcr.io/zitadel/zitadel:latest
  command: start-from-init --masterkey "MasterkeyNeedsToHave32Characters" --tlsMode disabled
  environment:
    - ZITADEL_DATABASE_POSTGRES_HOST=postgres
    - ZITADEL_DATABASE_POSTGRES_PORT=5432
    - ZITADEL_DATABASE_POSTGRES_DATABASE=${POSTGRES_DB}
    - ZITADEL_DATABASE_POSTGRES_USER_USERNAME=${POSTGRES_USER}
    - ZITADEL_DATABASE_POSTGRES_USER_PASSWORD=${POSTGRES_PASSWORD}
    - ZITADEL_DATABASE_POSTGRES_USER_SSL_MODE=disable
    - ZITADEL_EXTERNALSECURE=false
  ports:
    - "8081:8080"
  depends_on:
    postgres:
      condition: service_healthy
```

2. **Auth middleware** — валидация OIDC токенов:
```go
func ZitadelMiddleware(issuerURL, clientID string) echo.MiddlewareFunc {
    // Использовать go-oidc для валидации JWT от Zitadel
    // provider, _ := oidc.NewProvider(ctx, issuerURL)
    // verifier := provider.Verifier(&oidc.Config{ClientID: clientID})
}
```

3. **Роли и группы** — управляются через Zitadel Console UI, передаются в JWT claims

---

## Сравнение

| Возможность              | Simple Mode | Ory Kratos | Zitadel    |
|--------------------------|:-----------:|:----------:|:----------:|
| Простота деплоя          | ★★★         | ★★☆        | ★☆☆        |
| Self-service flows       | Базовый     | Полный     | Полный     |
| MFA/2FA                  | ✗           | ✓          | ✓          |
| SSO/OIDC                 | ✗           | ✗          | ✓          |
| Email verification       | ✗           | ✓          | ✓          |
| Password reset           | ✗           | ✓          | ✓          |
| Встроенный UI            | ✗           | ✗          | ✓          |
| Multi-tenancy            | ✗           | ✗          | ✓          |
| Ресурсы (RAM)            | ~50MB       | ~100MB     | ~500MB     |
| Зависимости              | PostgreSQL  | PostgreSQL | PostgreSQL |
