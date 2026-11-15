# Deploy to Vercel

The backend runs as a Go serverless function on Vercel. The existing Go + Gin API and PostgreSQL schema are retained.

## 1. Import the repository

1. Open Vercel and choose **Add New → Project**.
2. Import `zuukaisme/Tradejournal-Backend-` from GitHub.
3. Keep the project root directory as `./`.
4. Deploy after adding the environment variables below.

Vercel detects the Go function in `api/index.go`. The root rewrite forwards API requests to that function while preserving the requested route.

## 2. Add PostgreSQL from Vercel

1. Open the project in Vercel.
2. Open **Storage / Marketplace** and install the **Neon Postgres** integration.
3. Create a database and connect it to this Vercel project.
4. Ensure the integration provides `DATABASE_URL` to Production and Preview environments. Use the pooled connection string where the integration offers one.

Vercel Postgres is no longer a separate first-party product; use the Neon Marketplace integration.

## 3. Add the application secret

In **Project → Settings → Environment Variables**, add:

- `JWT_SECRET`: a unique cryptographically random value with at least 32 characters.

The database integration supplies `DATABASE_URL`. Do not put either secret in GitHub or source files. Redeploy after changing environment variables.

## 4. Verify

After deployment, open:

- `https://YOUR-PROJECT.vercel.app/health`
- `https://YOUR-PROJECT.vercel.app/ready`

The first checks that the API function is responding. The second also checks the PostgreSQL connection. The API applies pending migrations when its function instance initializes.

## Notes

- Vercel's Go runtime is currently Beta.
- This repository change prepares the application for Vercel; it does not create the Vercel project, provision Neon, add secrets, or deploy it.
- Email verification, password-reset email delivery, cloud backup endpoints, and file storage are not implemented yet.
- The current account API is available under `/v1/auth`.
