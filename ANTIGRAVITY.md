# ANTIGRAVITY.md

This file provides system guidelines, rules, and repository links for the **Antigravity** AI coding assistant when working with the DreamLog repository.

---

## 🚨 Required Reading & Execution Flow

Before answering any questions, writing code, or taking any execution actions, the agent **MUST** check the relevant markdown documents listed below to align with architectural contracts, design decisions, and guidelines.

### Documentation Index
* 🏗️ **[docs/ARCHITECTURE.md](file:///C:/Users/bharat.jain/Desktop/per/dream/docs/ARCHITECTURE.md)**
  * *Purpose:* Details the full system architecture, database schema, data flow, directory layouts, and end-to-end happy path request flow.
  * *Read when:* Proposing major structure changes, database schema updates, or adding new routes.
* 🔌 **[docs/API_CONTRACT.md](file:///C:/Users/bharat.jain/Desktop/per/dream/docs/API_CONTRACT.md)**
  * *Purpose:* The single source of truth for request/response shapes across all endpoints.
  * *Read when:* Modifying HTTP handlers in the backend or API services in the mobile app. Do not break these contracts!
* 🧠 **[docs/DECISIONS.md](file:///C:/Users/bharat.jain/Desktop/per/dream/docs/DECISIONS.md)**
  * *Purpose:* Architecture Decision Records (ADRs) explaining *why* things are built the way they are.
  * *Read when:* Proposing changes to core patterns (e.g. auth, worker statelessness, crisis detection fail-safes).
* 📝 **[docs/DreamLog_5Phase_Development_Plan.md](file:///C:/Users/bharat.jain/Desktop/per/dream/docs/DreamLog_5Phase_Development_Plan.md)**
  * *Purpose:* Strategic blueprint detailing product philosophy, longitudinal emotional intelligence scope, and specific requirements phase-by-phase.
  * *Read when:* Assessing overall product logic, phase features, or business rules.
* 🗺️ **[docs/ROADMAP.md](file:///C:/Users/bharat.jain/Desktop/per/dream/docs/ROADMAP.md)**
  * *Purpose:* Detailed status and checklist of completed, in-progress, and upcoming features across each phase.
  * *Read when:* Figuring out what needs implementation next or updating status.
* 🧪 **[docs/TESTING.md](file:///C:/Users/bharat.jain/Desktop/per/dream/docs/TESTING.md)**
  * *Purpose:* Testing priority matrix, mock definitions, and exact commands for verifying changes.
  * *Read when:* Preparing verification plans, writing tests, or validating backend/mobile changes.

---

## 🏛️ Core Project Architecture & Rules

1. **Backend (`backend/`):** Written in Go.
   * `cmd/api` runs the Gin HTTP API server. Runs DB migrations automatically on startup.
   * `cmd/worker` runs the asynchronous queue processor.
   * **In-memory state is forbidden:** Workers must remain stateless. All coordination goes through Redis and PostgreSQL.
2. **Mobile (`mobile/`):** React Native / Expo.
   * Follows file-based routing (`app/`).
   * Theme configuration is central in `src/theme.ts`.
3. **Safety-Critical Crisis Detection:**
   * Two-stage verification: keyword match → LLM confirmation.
   * **Must fail safe:** If LLM is unreachable or times out, the system *must* default to treating the entry as a crisis.
4. **LLM Prompts:**
   * Must live exclusively in [prompts.go](file:///C:/Users/bharat.jain/Desktop/per/dream/backend/internal/services/prompts.go). Never inline prompts in other services.
5. **No Placeholders / Incomplete Logic:**
   * Build complete, production-ready code with comprehensive error handling and logging.

---

## 💻 Developer Command Cheat Sheet

### Backend Commands (run from root or `backend/`)
```bash
go run ./cmd/api        # Start API server
go run ./cmd/worker     # Start queue worker
go test ./...           # Run tests
go mod tidy             # Clean up dependencies
```

### Mobile Commands (run from `mobile/`)
```bash
npm install             # Install package dependencies
npx expo start          # Start Expo developer server
npx expo start --android # Start on Android emulator
npx expo start --ios     # Start on iOS simulator
```

### Docker-Compose / Make Commands
```bash
make dev                # Boot up Postgres, Redis, MinIO, Whisper-mock, API, Worker
make dev-stop           # Stop containers
make dev-logs           # Tail container logs
make db-migrate         # Run migrations
```
