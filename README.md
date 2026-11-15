# Tradejournal Backend

Go + Gin API for Tradejournal accounts and future cloud backup.

## Stack

- Go 1.25
- Gin HTTP router
- PostgreSQL with pgx
- SQL migrations
- Docker and Docker Compose
- GitHub Actions CI

## Current endpoints

- `GET /health` — process health.
- `GET /ready` — checks PostgreSQL connectivity.
- `POST /v1/auth/register` — create an account using email and password.
- `POST /v1/auth/login` — sign in.
- `POST /v1/auth/refresh` — rotate a refresh token.
- `POST /v1/auth/logout` — revoke all active sessions for the account.
- `GET /v1/auth/me` — return the authenticated user and account IDs.

Registration generates a unique account ID automatically. Passwords are hashed with bcrypt. Access tokens expire after 15 minutes; refresh tokens are random, stored as hashes, rotated on refresh, and expire after 30 days.

## Run locally

1. Copy `.env.example` to `.env`.
2. Set `JWT_SECRET` to a private random value of at least 32 characters.
3. Start PostgreSQL with `docker compose up -d db`.
4. Export the environment values from `.env` and run `go run ./cmd/api`. The API applies pending SQL migrations automatically before accepting requests.

The API listens on `:8080` by default.

## Configuration

See `.env.example`. Never commit real credentials or production secrets.

Email verification, password-reset delivery, cloud backup endpoints, and object storage are not enabled yet.
