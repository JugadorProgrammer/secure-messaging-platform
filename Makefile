# Makefile координационного сервиса (COORD).
# make help — список целей.

IMAGE    ?= smp-coord
# Тег образа: $VERSION из окружения (или .env), иначе local.
TAG      ?= $(if $(VERSION),$(VERSION),local)
COMPOSE  ?= docker compose

# Документация: swagger — спецификация HTTP API, docs — сайт с документацией по коду.
SWAG_VERSION      ?= v1.16.6
GOMARKDOC_VERSION ?= v1.1.0
API_DIR           ?= api
SITE_DIR          ?= site
SITE_TITLE        ?= COORD
SWAG              = go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION)
GOMARKDOC         = go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@$(GOMARKDOC_VERSION)

# Переменные из .env (если файл есть) доступны run и test.
ifneq (,$(wildcard .env))
include .env
export
endif

.DEFAULT_GOAL := help
.PHONY: golangci-lint help build run test cover lint fmt fmt-check vet check swagger docs docker up infra down tidy

help: ## Список целей
	@grep -E '^[a-z0-9-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  %-12s %s\n", $$1, $$2}'

## --- Разработка ---

build: ## Собрать бинари bin/coord и bin/healthcheck
	go build -o bin/coord ./cmd/coord
	go build -o bin/healthcheck ./cmd/healthcheck

run: ## Запустить сервис локально
	go run ./cmd/coord

tidy: ## go mod tidy
	go mod tidy

## --- Проверки ---

test: ## Тесты с -race и покрытием
	go test -race -cover ./...

cover: ## Отчёт о покрытии → coverage.out, coverage.html
	go test -race -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html

GOLANGCI_LINT_VERSION ?= v2.15.0
# Минорная версия локального Go, например go1.27.
GO_MINOR = $(shell go env GOVERSION | sed -E 's/^(go[0-9]+\.[0-9]+).*/\1/')

# Ставит golangci-lint, если его нет, версия не совпадает с GOLANGCI_LINT_VERSION
# или он собран другой минорной версией Go (go1.27 поддерживается с v2.13.0).
golangci-lint:
	@golangci-lint version 2>/dev/null | grep -q "version $(GOLANGCI_LINT_VERSION:v%=%) built with $(GO_MINOR)" || \
		GOTOOLCHAIN=local go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

lint: golangci-lint ## golangci-lint
	golangci-lint run

fmt: ## Форматирование gofmt
	gofmt -w .

fmt-check: ## Проверить форматирование без записи
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "файлы выше не отформатированы: make fmt"; exit 1; }

vet: ## go vet
	go vet ./...

check: fmt-check vet lint test ## fmt-check + vet + lint + test

## --- Документация ---

swagger: ## OpenAPI (Swagger 2.0) из аннотаций cmd/ и internal/ → api/
	$(SWAG) init -d . -g cmd/coord/main.go --parseInternal -o $(API_DIR) --outputTypes json,yaml

docs: ## Документация по всем пакетам (cmd/, internal/) для Jekyll → site/
	mkdir -p $(SITE_DIR)
	$(GOMARKDOC) --include-unexported --output $(SITE_DIR)/api.raw.md ./...
	{ printf -- '---\ntitle: API\n---\n'; cat $(SITE_DIR)/api.raw.md; } > $(SITE_DIR)/api.md
	rm $(SITE_DIR)/api.raw.md
	{ printf -- '---\ntitle: $(SITE_TITLE)\n---\n'; cat README.md; } > $(SITE_DIR)/index.md
	printf 'theme: jekyll-theme-cayman\ntitle: $(SITE_TITLE)\n' > $(SITE_DIR)/_config.yml

## --- Docker ---

docker: ## Собрать образ smp-coord:<VERSION>, без VERSION — :local
	docker build --build-arg VERSION=$(TAG) -t $(IMAGE):$(TAG) .

up: ## Поднять PostgreSQL и сервис (docker-compose.yml)
	$(COMPOSE) up -d --build

infra: ## Поднять только PostgreSQL
	$(COMPOSE) up -d postgres

down: ## Остановить стенд
	$(COMPOSE) down
