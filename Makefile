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
        api worker tidy vet lint \
        test test-race test-cover test-crisis test-services test-handlers test-workers \
        mobile-install mobile-start mobile-tunnel mobile-android mobile-ios mobile-web mobile-lint mobile-typecheck \
        mobile-build-dev mobile-build-dev-local mobile-build-preview mobile-build-preview-local mobile-build-prod \
        mobile-build-dev-ios mobile-build-preview-ios mobile-build-prod-ios \
        mobile-submit-android mobile-submit-ios mobile-device-ios mobile-versions \
        apk apk-debug apk-ci apk-download \
        portal-install portal-dev portal-build portal-start portal-lint

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
	@echo "DreamLog - available make targets"
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
	@echo "    make db-reset         Drop DB + re-run all migrations (destructive)"
	@echo "    make db-psql          Open psql session"
	@echo ""
	@echo "  Scaling"
	@echo "    make scale-worker N=2 Scale worker to N replicas"
	@echo ""
	@echo "  Backend - local Go (no Docker)"
	@echo "    make api              go run ./cmd/api"
	@echo "    make worker           go run ./cmd/worker"
	@echo "    make tidy             go mod tidy"
	@echo "    make vet              go vet ./..."
	@echo "    make lint             golangci-lint run (requires golangci-lint)"
	@echo ""
	@echo "  Backend - tests"
	@echo "    make test             go test ./..."
	@echo "    make test-race        go test -race ./...  (always use in CI)"
	@echo "    make test-cover       go test -coverprofile=coverage.out + open HTML"
	@echo "    make test-crisis      Run only crisis detection tests (blocking)"
	@echo "    make test-services    Run only internal/services tests"
	@echo "    make test-handlers    Run only internal/handlers tests"
	@echo "    make test-workers     Run only internal/workers tests"
	@echo ""
	@echo "  Mobile - dev server"
	@echo "    make mobile-install        npm install"
	@echo "    make mobile-start          expo start"
	@echo "    make mobile-tunnel         expo start --tunnel"
	@echo "    make mobile-android        expo start --android"
	@echo "    make mobile-ios            expo start --ios"
	@echo "    make mobile-web            expo start --web"
	@echo "    make mobile-lint           eslint check"
	@echo "    make mobile-typecheck      tsc --noEmit"
	@echo ""
	@echo "  Mobile - EAS cloud builds (free tier = long queue)"
	@echo "    make mobile-build-dev          EAS cloud, development profile (Android)"
	@echo "    make mobile-build-preview      EAS cloud, preview profile (Android)"
	@echo "    make mobile-build-prod         EAS cloud, production profile (Android)"
	@echo "    make mobile-build-dev-ios      EAS cloud, development profile (iOS)"
	@echo "    make mobile-build-preview-ios  EAS cloud, preview profile (iOS)"
	@echo "    make mobile-build-prod-ios     EAS cloud, production profile (iOS)"
	@echo ""
	@echo "  Mobile - store release (see docs/LAUNCH_CHECKLIST.md)"
	@echo "    make mobile-submit-android  Upload latest prod build to Play Console"
	@echo "    make mobile-submit-ios      Upload latest prod build to TestFlight"
	@echo "    make mobile-device-ios      Register an iPhone UDID for dev builds"
	@echo "    make mobile-versions        Show remote versionCode / buildNumber"
	@echo ""
	@echo "  Mobile - APK builds (Windows-native, no EAS/cloud)"
	@echo "    make apk              Build release APK via Gradle (needs Android Studio)"
	@echo "    make apk-debug        Build debug APK - faster, no signing"
	@echo "    make apk-ci           Trigger build on GitHub Actions (fallback)"
	@echo "    make apk-download     Download latest APK from GitHub Actions"
	@echo ""
	@echo "  Therapist Portal (Next.js)"
	@echo "    make portal-install   npm install"
	@echo "    make portal-dev       next dev  (http://localhost:3000)"
	@echo "    make portal-build     next build"
	@echo "    make portal-start     next start"
	@echo "    make portal-lint      next lint"
	@echo ""

# ── Dev lifecycle ─────────────────────────────────────────────────────────────
dev:
	@$(COPY_ENV)
	docker compose up --build -d
	@echo ""
	@echo "DreamLog running"
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

# Remove containers AND volumes (full wipe - loses all data).
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

# Wipe and re-apply all migrations (DEV ONLY - destroys all data).
db-reset:
	@echo "WARNING: This will drop and recreate the database. Ctrl-C to abort."
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
	@echo "Worker scaled to $(N) replicas"

# ── MinIO ─────────────────────────────────────────────────────────────────────
minio-console:
	@echo "MinIO console → http://localhost:9001"
	@echo "  user:     minioadmin"
	@echo "  password: minioadmin_secret"

# ── Backend - local Go (no Docker) ───────────────────────────────────────────
api:
	cd backend && go run ./cmd/api

worker:
	cd backend && go run ./cmd/worker

tidy:
	cd backend && go mod tidy

vet:
	cd backend && go vet ./...

lint:
	cd backend && golangci-lint run ./...

# ── Backend - tests ───────────────────────────────────────────────────────────

# Run all tests. Use test-race in CI.
test:
	cd backend && go test ./...

# Run all tests with the race detector (required before any merge).
test-race:
	cd backend && go test -race ./...

# Run tests with coverage, open the HTML report.
test-cover:
	cd backend && go test -coverprofile=coverage.out ./... && \
	  go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: backend/coverage.html"

# Safety-critical: crisis detection only. Blocks merges if these fail.
test-crisis:
	cd backend && go test -v -race ./internal/services/ -run "Crisis|crisis"

# All service-layer unit tests.
test-services:
	cd backend && go test -race ./internal/services/...

# All HTTP handler integration tests.
test-handlers:
	cd backend && go test -race ./internal/handlers/...

# All worker tests (transcription pipeline, schedulers, nudge).
test-workers:
	cd backend && go test -race ./internal/workers/...

# ── Mobile ────────────────────────────────────────────────────────────────────
# --legacy-peer-deps: the tree has a pre-existing react/react-dom peer conflict
# that npm's strict resolver rejects; all installs in mobile/ need this flag.
mobile-install:
	cd mobile && npm install --legacy-peer-deps

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

mobile-lint:
	cd mobile && npx eslint . --ext .ts,.tsx

mobile-typecheck:
	cd mobile && npx tsc --noEmit

# ── Mobile - EAS builds ───────────────────────────────────────────────────────
# Cloud builds (EAS servers - free tier queues can take hours).
mobile-build-dev:
	cd mobile && npx eas build --profile development --platform android

mobile-build-preview:
	cd mobile && npx eas build --profile preview --platform android

mobile-build-prod:
	cd mobile && npx eas build --profile production --platform android

# iOS builds run on EAS macOS workers - no Mac needed locally.
# Prerequisites: Apple Developer account + mobile/GoogleService-Info.plist
# (see docs/LAUNCH_CHECKLIST.md section 2b - iOS builds fail without the plist).
mobile-build-dev-ios:
	cd mobile && npx eas build --profile development --platform ios

mobile-build-preview-ios:
	cd mobile && npx eas build --profile preview --platform ios

mobile-build-prod-ios:
	cd mobile && npx eas build --profile production --platform ios

# ── Mobile - store release ────────────────────────────────────────────────────
# Versioning: bump "version" in mobile/app.json + git tag; versionCode /
# buildNumber are auto-incremented remotely by EAS (appVersionSource: remote).
mobile-submit-android:
	cd mobile && npx eas submit --platform android --latest

mobile-submit-ios:
	cd mobile && npx eas submit --platform ios --latest

# Register a physical iPhone's UDID so development builds can install on it.
mobile-device-ios:
	cd mobile && npx eas device:create

# Show the remote build counters EAS will use for the next build.
mobile-versions:
	cd mobile && npx eas build:version:get --platform android && \
	  npx eas build:version:get --platform ios

# Local builds (runs on your machine - no queue, no wait).
# Requires: Android SDK + JDK for Android; Xcode for iOS (macOS only).
# Install prerequisite: npm install -g eas-cli
mobile-build-dev-local:
	cd mobile && npx eas build --profile development-local --platform android --local --output ./build/dreamlog-dev.apk

mobile-build-preview-local:
	cd mobile && npx eas build --profile preview-local --platform android --local --output ./build/dreamlog-preview.apk

JAVA_HOME = C:\Program Files\Android\Android Studio\jbr

# Build APK directly using Gradle - works on Windows, no EAS/cloud needed.
# Requires: Android Studio installed (includes JDK + Android SDK).
# First run takes ~3 min (Gradle download); subsequent runs ~1 min.
apk:
	cd mobile && npx expo prebuild --platform android --clean
	powershell -ExecutionPolicy Bypass -File scripts\write-local-props.ps1
	powershell -ExecutionPolicy Bypass -File scripts\run-gradle.ps1 -Task assembleRelease
	@echo ""
	@echo "APK ready: mobile\android\app\build\outputs\apk\release\app-release.apk"

# Debug APK (faster build, no signing needed - good for quick device testing).
apk-debug:
	cd mobile && npx expo prebuild --platform android --clean
	powershell -ExecutionPolicy Bypass -File scripts\write-local-props.ps1
	powershell -ExecutionPolicy Bypass -File scripts\run-gradle.ps1 -Task assembleDebug
	@echo ""
	@echo "APK ready: mobile\android\app\build\outputs\apk\debug\app-debug.apk"

# Trigger build on GitHub Actions (fallback if Android Studio is not installed).
apk-ci:
	gh workflow run build-apk.yml --field profile=preview-local
	@echo "Build triggered - download from the Actions tab when done."

# Download the latest APK artifact from GitHub Actions.
apk-download:
	gh run download $$(gh run list --workflow=build-apk.yml --limit 1 --json databaseId -q '.[0].databaseId') --dir ./mobile/build
	@echo "APK saved to: mobile/build/"

# ── Therapist Portal (Next.js) ────────────────────────────────────────────────
portal-install:
	cd therapist-portal && npm install

portal-dev:
	cd therapist-portal && npm run dev

portal-build:
	cd therapist-portal && npm run build

portal-start:
	cd therapist-portal && npm run start

portal-lint:
	cd therapist-portal && npm run lint
