-include .env
export

.PHONY: help run build test test-integration test-all mocks deps docker-restart migrate-up migrate-down setup

run: ## Запустить приложение
	go run main.go

build: ## Собрать приложение
	go build -o bin/gotemplate .

test: ## Запустить unit тесты
	go test -race ./...

test-integration: ## Запустить интеграционные тесты (поднимает тестовый PostgreSQL в Docker)
	docker-compose -f docker-compose.test.yml up -d --wait
	TEST_POSTGRES_PORT=5433 go test -race -tags integration -count=1 -p 1 ./...; \
	  EXIT_CODE=$$?; \
	  docker-compose -f docker-compose.test.yml down; \
	  exit $$EXIT_CODE

test-all: test test-integration ## Запустить все тесты

mocks: ## Сгенерировать моки (mockery)
	go run github.com/vektra/mockery/v2@latest

deps: ## Загрузить зависимости
	go mod download
	go mod tidy

docker-restart: ## Остановить, пересобрать и запустить все сервисы с миграциями
	docker-compose down
	docker-compose up -d --build

migrate-up: ## Применить миграции
	migrate -path migrations -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)" up

migrate-down: ## Откатить миграции
	migrate -path migrations -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@$(POSTGRES_HOST):$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=$(POSTGRES_SSL_MODE)" down 1

rename-module: ## Сменить module path: make rename-module OLD=github.com/alexbro4u/gotemplate NEW=github.com/your-org/new-project
	@if [ -z "$(OLD)" ] || [ -z "$(NEW)" ]; then echo "Usage: make rename-module OLD=old/module NEW=new/module"; exit 1; fi
	find . -type f \( -name '*.go' -o -name 'go.mod' \) -exec sed -i '' 's|$(OLD)|$(NEW)|g' {} +
	@echo "Module renamed from $(OLD) to $(NEW)"

setup: docker-restart ## Полная настройка: запустить Docker (миграции запускаются в docker-compose)
