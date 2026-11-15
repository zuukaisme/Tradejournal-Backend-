# Tradejournal Backend

Go + Gin API for Tradejournal accounts and future cloud backup.

## Stack

- Go 1.25
- Gin HTTP router
- PostgreSQL with pgx
- SQL migrations
- Docker and Docker Compose
- GitHub Actions CI

## Current scope

This repository contains the backend foundation: configuration, PostgreSQL connectivity, health endpoints, the initial account/session schema, local development setup, and automated checks.

Authentication endpoints, email delivery, cloud backup uploads, and object storage are intentionally not enabled yet. They will be added in separate stages and tested before the Android app depends on them.

## Run locally

1. Copy `.env.example` to `.env`.
2. Start PostgreSQL:

```sh
docker compose up -d db
```

3. Apply the initial schema:

```sh
psql "$DATABASE_URL" -f migrations/000001_initial_schema.up.sql
```

4. Start the API:

```sh
go run ./cmd/api
```

The API listens on `:8080` by default.

## Endpoints

- `GET /health` — process health.
- `GET /ready` — checks PostgreSQL connectivity.

## Configuration

See `.env.example`. Never commit real credentials or production secrets.
