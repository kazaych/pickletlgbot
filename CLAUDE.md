# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run locally (requires .env file)
go run cmd/main.go

# Build binary
go build -o kitchenbot ./cmd/main.go

# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/domain/event/...

# Start full stack locally (bot + postgres)
docker compose up

# Build Docker image
docker build -t kitchenbot .

# Tidy dependencies
go mod tidy
```

## Environment Variables

Required in `.env`:
- `TELEGRAM_BOT_TOKEN` — bot token from BotFather
- `DATABASE_URL` — `postgres://user:pass@host:5432/dbname`
- `ADMIN_IDS` — comma-separated Telegram user IDs with admin access

## Architecture

This is a Telegram bot (KitchenBot) for managing pickleball training events. Users register for events via inline keyboards; admins create events and approve/reject registrations.

**Layered clean architecture:**

```
cmd/main.go                    → entry point, wires dependencies, runs auto-migration, graceful shutdown
api/telegram/                  → Telegram API layer
  handlers.go                  → main update router, owns state maps for multi-step flows
  admin_handlers.go            → admin commands (create event, manage locations, approve registrations)
  user_handlers.go             → user commands (list events, register, unregister)
  client.go                    → thin Telegram API wrapper
  formatter.go                 → builds inline keyboard messages
internal/domain/{event,location,user}/
  entity.go                    → pure Go domain structs (no GORM tags)
  repository.go                → repository interface
  service.go                   → business logic (use cases)
repositories/postgres/         → GORM implementations of domain repository interfaces
internal/models/               → GORM structs with DB tags (separate from domain entities)
```

**Key design patterns:**
- Domain entities and GORM models are kept separate; repositories translate between them
- Context is propagated through all layers for graceful shutdown
- Admin access is checked per-update against `ADMIN_IDS` env var

## State Management

Multi-step Telegram flows are managed via in-memory maps in `handlers.go`:
- `creatingEvents` — event creation wizard (type → max_players → name → date → trainer → payment_phone → price)
- `creatingLocations` — location creation wizard (name → address → map_url)
- `registeringUsers` — user registration (name → surname)

Callback button routing uses prefixes: `loc:`, `event:`, `admin:`.

## Database

PostgreSQL via GORM. Auto-migration runs on startup in dependency order:
1. `UserGORM`, `LocationGORM`, `EventGORM`
2. `EventRegistrationGORM` (FK → User + Event, CASCADE delete)

Soft deletes on all tables. Price stored in kopecks (integer). Event `Remaining` field is maintained by service logic when approving/rejecting registrations.

## CI/CD

GitHub Actions (`.github/workflows/docker-image.yml`) triggers on push to `main`:
1. Builds and pushes Docker image to Docker Hub
2. SSHes into production server, updates `.env`, restarts container
