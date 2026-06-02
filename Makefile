.PHONY: help \
        dev dev-stop dev-restart dev-status dev-logs \
        logs-api logs-worker logs-whisper logs-postgres logs-redis \
        build build-api build-worker \
        up down restart \
        ps health \
        shell-api shell-postgres shell-redis \
        db-migrate db-migrate-down db-reset db-psql \
        scale-worker \
        minio-console \
        api worker tidy \
        mobile-install mobile-start mobile-tunnel mobile-android mobile-ios mobile-web

ifeq ($(OS),Windows_NT)
    COPY_ENV = if not exist .env copy .env.example .env
    DEV_NULL = NUL
else
    COPY_ENV = cp -n .env.example .env 2>/dev/null || true
    DEV_NULL = /dev/null
endif

# ── Default: print help ───────────────────────────────────────────────────────
help:
	@echo ""
	@echo "DreamLog — available make targets"
	@echo ""
	@echo "  Dev lifecycle"
	@echo "    make dev              Build + start all services (detached)"
	@echo "    make dev-stop         Stop + remove containers (keep volumes)"
	@echo "    make dev-restart      Rebuild changed images + restart"
	@echo "    make dev-status       Show container status"
	@echo "    make ps               Alias for dev-status"
	@echo "    make health           Curl the API health endpoint"
	@echo "    make down             Stop + remove containers AND volumes"
	@echo ""
	@echo "  Logs"
	@echo "    make dev-logs         Tail API + worker logs"
	@echo "    make logs-api         Tail API only"
	@echo "    make logs-worker      Tail worker only"
	@echo "    make logs-whisper     Tail Whisper server"
	@echo "    make logs-postgres    Tail PostgreSQL"
	@echo "    make logs-redis       Tail Redis"
	@echo ""
	@echo "  Build"
	@echo "    make build            Rebuild all images (no cache)"
	@echo "    make build-api        Rebuild API image only"
	@echo "    make build-worker     Rebuild worker image only"
	@echo ""
	@echo "  Shells"
	@echo "    make shell-api        Exec into running API container"
	@echo "    make shell-postgres   Exec into PostgreSQL container"
	@echo "    make shell-redis      Exec redis-cli"
	@echo ""
	@echo "  Database"
	@echo "    make db-migrate       Apply pending migrations"
	@echo "    make db-migrate-down  Roll back last migration"
	@echo "    make db-reset         Drop DB + re-run all migrations"
	@echo "    make db-psql          Open psql session"
	@echo ""
	@echo "  Scaling"
	@echo "    make scale-worker N=2 Scale worker to N replicas"
	@echo ""
	@echo "  Local Go (no Docker)"
	@echo "    make api              go run ./cmd/api"
	@echo "    make worker           go run ./cmd/worker"
	@echo "    make tidy             go mod tidy"
	@echo ""
	@echo "  Mobile"
	@echo "    make mobile-install   npm install"
	@echo "    make mobile-start     expo start"
	@echo "    make mobile-tunnel    expo start --tunnel"
	@echo "    make mobile-android   expo start --android"
	@echo "    make mobile-ios       expo start --ios"
	@echo "    make mobile-web       expo start --web"
	@echo ""

# ── Dev lifecycle ─────────────────────────────────────────────────────────────
dev:
	@$(COPY_ENV)
	docker compose up --build -d
	@echo ""
	@echo "✓ DreamLog running"
	@echo "  API            http://localhost:8080/health"
	@echo "  MinIO console  http://localhost:9001  (minioadmin / minioadmin_secret)"
	@echo "  PostgreSQL     localhost:5432"
	@echo "  Redis          localhost:6379"
	@echo "  Whisper        localhost:9002"
	@echo ""

dev-stop:
	docker compose down

# Rebuild only images that have changed source files, then restart.
dev-restart:
	docker compose up --build -d --remove-orphans

# Remove containers AND volumes (full wipe — loses all data).
down:
	docker compose down -v --remove-orphans

dev-status:
	docker compose ps

ps: dev-status

health:
	@curl -sf http://localhost:8080/health | python3 -m json.tool 2>$(DEV_NULL) || \
	  curl -s http://localhost:8080/health

# ── Logs ─────────────────────────────────────────────────────────────────────
dev-logs:
	docker compose logs -f api worker

logs-api:
	docker compose logs -f api

logs-worker:
	docker compose logs -f worker

logs-whisper:
	docker compose logs -f whisper

logs-postgres:
	docker compose logs -f postgres

logs-redis:
	docker compose logs -f redis

# ── Build ─────────────────────────────────────────────────────────────────────
build:
	docker compose build --no-cache

build-api:
	docker compose build --no-cache api

build-worker:
	docker compose build --no-cache worker

# Rebuild + force-recreate a single service (picks up .env changes too).
restart:
	@read -p "Service to restart (api/worker/whisper): " svc; \
	  docker compose up -d --build --force-recreate $$svc

# ── Shells ────────────────────────────────────────────────────────────────────
shell-api:
	docker compose exec api sh

shell-postgres:
	docker compose exec postgres sh

shell-redis:
	docker compose exec redis redis-cli -a $$(grep REDIS_PASSWORD .env | cut -d= -f2)

# ── Database ──────────────────────────────────────────────────────────────────
# Run migrations inside the API container (has the binary + migration files).
db-migrate:
	docker compose exec api sh -c \
	  'migrate -path /app/migrations -database "$$DATABASE_URL" up'

db-migrate-down:
	docker compose exec api sh -c \
	  'migrate -path /app/migrations -database "$$DATABASE_URL" down 1'

# Wipe and re-apply all migrations (DEV ONLY — destroys all data).
db-reset:
	@echo "⚠  This will drop and recreate the database. Ctrl-C to abort."
	@sleep 3
	docker compose exec api sh -c \
	  'migrate -path /app/migrations -database "$$DATABASE_URL" drop -f && \
	   migrate -path /app/migrations -database "$$DATABASE_URL" up'

db-psql:
	docker compose exec postgres psql \
	  -U $$(grep POSTGRES_USER .env | cut -d= -f2) \
	  -d $$(grep POSTGRES_DB   .env | cut -d= -f2)

# ── Scaling ───────────────────────────────────────────────────────────────────
N ?= 2
scale-worker:
	docker compose up -d --scale worker=$(N) --no-recreate
	@echo "✓ Worker scaled to $(N) replicas"

# ── MinIO ─────────────────────────────────────────────────────────────────────
minio-console:
	@echo "MinIO console → http://localhost:9001"
	@echo "  user:     minioadmin"
	@echo "  password: minioadmin_secret"

# ── Local Go (without Docker) ─────────────────────────────────────────────────
api:
	cd backend && go run ./cmd/api

worker:
	cd backend && go run ./cmd/worker

tidy:
	cd backend && go mod tidy

# ── Mobile ────────────────────────────────────────────────────────────────────
mobile-install:
	cd mobile && npm install

mobile-start:
	cd mobile && npx expo start

mobile-tunnel:
	cd mobile && npx expo start --tunnel

mobile-android:
	cd mobile && npx expo start --android

mobile-ios:
	cd mobile && npx expo start --ios

mobile-web:
	cd mobile && npx expo start --web
