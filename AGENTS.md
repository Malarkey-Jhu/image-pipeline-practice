# Repository Guidelines

## Project Structure & Module Organization
- `cmd/api` and `cmd/worker`: entrypoints for the HTTP API and background worker.
- `internal/`: application code (config, db, mq, storage, pipeline, worker logic).
- `deployments/migrations`: SQL migrations (schema for `media` and `processing_task`).
- `scripts/`: local helper scripts (curl-based tests will live here).
- `spec.md`: system design PRD and testing plan.

## Build, Test, and Development Commands
- `cp .env.example .env`: create local environment config.
- `docker compose up -d`: start Postgres, RabbitMQ, MinIO, API, and Worker.
- `go run ./cmd/api`: run the API locally (outside Docker).
- `go run ./cmd/worker`: run the worker locally (outside Docker).
- `go test ./...`: run all Go tests (when added).
- `docker compose exec -T postgres psql -U app -d app -f /app/deployments/migrations/001_init.sql`: apply initial schema.

## Coding Style & Naming Conventions
- Go code follows `gofmt` formatting (tabs for indentation).
- Package names are short, lowercase, and single-purpose (e.g., `db`, `mq`).
- Files should group related types and keep exported APIs minimal.

## Testing Guidelines
- Testing will use Go’s standard `testing` package.
- Test files should be named `*_test.go` and live alongside the code they test.
- Integration tests should prefer Docker Compose services to mirror local runtime.

## Commit & Pull Request Guidelines
- No commit message convention is established yet; use short, imperative summaries (e.g., `add worker claim loop`).
- PRs should include a brief description, the related spec section (if applicable), and how to verify (commands or curl steps).

## Security & Configuration Tips
- Local secrets are stored in `.env` (do not commit). Copy from `.env.example`.
- Object storage keys should be deterministic to ensure idempotency (see `spec.md`).

## Architecture Overview
- The system is an async pipeline: API writes metadata, MQ queues tasks, Worker processes images, and state lives in Postgres.
- Reliability relies on DB lease locks and "update DB before ACK" semantics.
