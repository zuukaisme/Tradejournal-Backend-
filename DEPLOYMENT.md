# Production deployment

The API is container-ready and can be deployed through Dokploy or another Docker Compose host. PostgreSQL should be a managed PostgreSQL database for production. Do not use the local development database or commit production credentials.

## Required environment

Create a private `.env.production` file on the deployment host using `.env.production.example` as a template.

- `APP_ENV=production`
- `APP_PORT=8080`
- `DATABASE_URL`: managed PostgreSQL connection string. Use the provider's TLS-enabled connection string and retain `sslmode=require` (or the provider's stricter TLS setting).
- `JWT_SECRET`: a unique random secret of at least 32 characters. Generate it on the deployment host; never reuse the example value.

The API automatically applies pending database migrations during startup. Keep the database reachable before deploying a new application version. Back up the database before production upgrades.

## Dokploy / Docker Compose

1. Create an application from this repository.
2. Set the Compose file to `docker-compose.production.yml`.
3. Add the production environment variables in Dokploy's environment settings, or provide `.env.production` securely on the host.
4. Set the application service's exposed container port to `8080`.
5. Attach a domain and enable HTTPS through the Dokploy proxy.
6. Deploy, then verify `GET /health` and `GET /ready` on the deployed domain.

Do not publish port 8080 directly to the public internet when using a reverse proxy. Route traffic through the HTTPS proxy.

## Database

Use a managed PostgreSQL instance with automated backups and point-in-time recovery where available. Restrict database network access to the application host when the provider supports it. The application requires permission to create and update its migration history and application tables.

## Not deployed yet

This repository change only prepares the deployment configuration and instructions. It does not create a database, configure a domain, provision secrets, or deploy the API. Email delivery, cloud backup endpoints, and object storage remain separate future work.
