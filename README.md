# Go Template

Go проект-шаблон с HTTP handlers, PostgreSQL, JWT аутентификацией, метриками Prometheus и трейсингом Jaeger.

## Что менять для нового проекта

- **go.mod** — module path и импорты во всех `.go` (см. раздел ниже).
- **docker-compose.yml** — имена контейнеров.
- **.env** — из `.env.example`

## Смена module path

Текущий module: `github.com/alexbro4u/gotemplate`

Чтобы сменить module path под свой проект, используйте команду:

```bash
make rename-module OLD=github.com/alexbro4u/gotemplate NEW=github.com/your-org/your-project
```

Команда заменит module path в `go.mod` и во всех `.go` файлах.

После переименования выполните:

```bash
go mod tidy
```

## TODO:
- отладить идемпотентность
- добавить проверку прав доступа

## Структура проекта

```
.
├── cmd/app/              # Точка входа приложения
├── internal/
│   ├── config/           # Конфиги
│   ├── core/             # Ядро - HTTP, метрики, трейсинг, JWT
│   ├── entity/           # Сущности
│   ├── dto/               
│   └── layers/            # Слои 
│       ├── controllers/   # HTTP handlers
│       ├── services/      # Бизнес-логика
│       ├── repositories/  # Доступ к данным
│       └── middlewares/   # auth, idempotency
├── migrations/            # миграции
├── pkg/                  
└── requests/             # HTTP запросы для тестирования
```

## User

`User` - пользователь с полями: UUID, email, name, password_hash, role (user/admin), created_at, updated_at.

## JWT

JWT-аутентификация: регистрация и логин возвращают токен, защищенные эндпоинты требуют токен в заголовке `Authorization: Bearer <token>`, middleware проверяет токен и устанавливает `user_uuid` в контекст.

## База данных

PostgreSQL с `sqlx` для работы с БД, миграции через `golang-migrate`, таблицы: `users` и `request_cache`

## Docker

`Dockerfile` для сборки приложения, `docker-compose.yml` для локальной разработки, команда `make docker-restart` для пересборки и запуска всех сервисов.

## Быстрый старт

```bash
# Запустить все сервисы (PostgreSQL + Jaeger + App + миграции)
make docker-restart

# Или через docker-compose
docker-compose up -d --build
```

Приложение доступно на `http://localhost:8080`

## Переменные окружения

`.env.example` - скопировать в `.env` и заполнить
