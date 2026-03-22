# GUE Fullstack Starter Kit

Production-quality fullstack starter kit with clean architecture, JWT auth, Redis/Memcached caching abstraction, Asynq jobs, Vue 3 frontend, and CI/CD-ready workflows.

## Tech Stack

### Backend
- Go + Gin
- MySQL
- Redis (refresh-token/session store + Asynq queue backend)
- Memcached (query cache + API response cache via abstraction)
- Asynq worker (separate process)
- Viper config loader with env precedence
- golang-migrate SQL migrations

### Frontend
- Vue 3 + Vite
- Tailwind CSS v4
- shadcn-vue style component system (`src/components/ui`)
- Pinia + Vue Router
- Vitest + Vue Test Utils

## Project Structure

```text
.
├── backend/
│   ├── cmd/
│   │   ├── server/
│   │   └── worker/
│   ├── internal/
│   ├── handler/
│   ├── service/
│   ├── repository/
│   ├── model/
│   ├── middleware/
│   ├── config/
│   ├── cache/
│   ├── queue/
│   ├── tests/
│   ├── pkg/
│   ├── migrations/
│   └── docs/
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   ├── views/
│   │   ├── stores/
│   │   ├── router/
│   │   └── services/
│   └── tests/
├── docker-compose.yml
└── .github/workflows/ci.yml
```

## Quick Start (Docker)

1. Copy environment template:
```bash
cp .env.example .env
```

2. Start stack:
```bash
docker compose up --build -d
```

`initdb` runs automatically before API/worker to:
- wait for MySQL readiness
- apply migrations
- seed payments
- ensure exactly one `dev` user from `BOOTSTRAP_DEV_*` env

Dummy dataset for local/demo:

```bash
cd backend
go run ./cmd/initdb --seed
```

Fresh reset + dummy seed:

```bash
cd backend
go run ./cmd/initdb --fresh --seed
```

Environment source of truth:
- use only root `.env`
- backend and frontend both load variables from root `.env`

3. App endpoints:
- Frontend: `http://localhost:5173`
- Backend API: `http://localhost:8080`
- OpenAPI: `http://localhost:8080/openapi.yaml`

## Local Development

### Prerequisites
- Go 1.24+
- Node.js 22+
- MySQL 8+
- Redis 7+
- Memcached 1.6+

### Backend

```bash
cd backend
go mod tidy
go run ./cmd/server
```

Run worker separately:
```bash
cd backend
go run ./cmd/server --worker
# or
go run ./cmd/worker
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

## Migrations (golang-migrate)

Create DB first (`gue`), then run:

```bash
migrate -path backend/migrations -database "mysql://root:secret@tcp(localhost:3306)/gue" up
```

Rollback one migration:

```bash
migrate -path backend/migrations -database "mysql://root:secret@tcp(localhost:3306)/gue" down 1
```

Fresh reset database (drop + recreate + migrate + seed + bootstrap dev):

```bash
cd backend
go run ./cmd/initdb --fresh
```

## API Endpoints

- `POST /api/v1/auth/register`
- `GET /api/v1/auth/csrf`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/user/me`
- `GET /api/v1/tokos` (auth)
- `POST /api/v1/tokos` (auth, max 3 tokos/user)
- `GET /api/v1/tokos/balances` (auth)
- `PATCH /api/v1/tokos/:id/settlement` (auth, dev/superadmin)
- `GET /api/v1/dashboard/overview` (auth)
- `GET /api/v1/transactions/history` (auth)
- `POST /api/v1/users` (auth, role must not be `user`)
- `PATCH /api/v1/users/:id/role` (auth, role must be `dev` or `superadmin`)
- `POST /api/v1/payments/gateway/generate` (Bearer toko token)
- `POST /api/v1/payments/gateway/check-status/:trx_id` (Bearer toko token)
- `POST /api/v1/payments/gateway/inquiry` (Bearer toko token)
- `POST /api/v1/payments/gateway/transfer` (Bearer toko token)
- `POST /api/v1/payments/gateway/transfer/check-status/:partner_ref_no` (Bearer toko token)
- `POST /api/v1/payments/gateway/balance` (Bearer toko token)
- `POST /api/v1/payments/gateway/callback/qris` (public callback)
- `POST /api/v1/payments/gateway/callback/transfer` (public callback)

## Postman and HTTP Examples

Internal API examples are available so the team can use project endpoints directly (not vendor endpoints):

- Postman collection: `backend/docs/postman/GUE-Internal-API.postman_collection.json`
- Postman environment: `backend/docs/postman/GUE-Local.postman_environment.json`
- HTTP examples (VS Code REST Client): `backend/docs/postman/internal-api-examples.http`

Recommended flow:

1. Import collection + environment into Postman.
2. Fill variables: `toko_token`, `merchant_uuid`, `gateway_client`, `callback_secret`, and user management vars.
3. Run in order: `Login` -> `Create Toko` -> payment bridge endpoints.

## Auth + Authorization Rules

### Guest users
- Allowed: `/login`
- Redirected from `/dashboard`, `/histori-transaksi`, `/toko` to `/login`

### Authenticated users
- Active users can access `/dashboard`
- Active users can access `/histori-transaksi`
- Active users can access `/toko`
- Inactive users are rejected by backend middleware and redirected to `/login`
- Only roles `dev` and `superadmin` can change user role via API
- Only roles `dev` and `superadmin` can apply manual settlement toko

## Testing

### Backend

```bash
cd backend
go test ./...
```

Coverage includes:
- service unit tests
- repository tests (sqlmock)
- handler HTTP tests

### Frontend

```bash
cd frontend
npm run test:run
```

Coverage includes:
- component tests
- Pinia stores (`auth`, `user`)
- route guards

## Linting

Backend:
```bash
cd backend
go vet ./...
```

Frontend:
```bash
cd frontend
npm run lint
```

## Architecture Explanation

### 1) Request Flow
`client -> router -> middleware -> handler -> service -> repository -> cache -> database`

- Router handles route matching (`/api/v1/...`).
- Middleware handles request ID, logging, CORS, recovery, and auth.
- Handlers decode/validate HTTP payload and map responses.
- Services contain business logic and orchestrate repositories/cache/queue.
- Repositories isolate persistence details.
- Cache layer is abstracted via interface and can use Redis or Memcached.
- Database (MySQL) remains the source of truth.

### 2) JWT Flow
`csrf bootstrap -> login -> access/refresh cookies -> access expires -> refresh rotates cookies`

- Access token and refresh token are set as cookies (`HttpOnly`, `SameSite`, optional `Secure`).
- CSRF token is issued by `GET /api/v1/auth/csrf` and must be sent in `X-CSRF-Token` for mutating requests.
- Refresh token ID (`jti`) is persisted in Redis.
- Refresh flow validates JWT signature + Redis presence, rotates refresh token, and sets new cookies.
- Logout revokes refresh token by deleting its token ID from Redis and clears cookies.

### 3) Cache Strategy (Read-through)

- Cache is queried first for read-heavy data (`/api/v1/user/me`).
- On cache miss, data is loaded from DB and stored in cache.
- External gateway balance on dashboard is cached for `5 minutes`.
- Business logic depends only on cache interface (`Get`, `Set`, `Delete`).
- Cache backend can be switched between Memcached or Noop.
- Redis is reserved only for refresh token/session storage, rate limiting, and Asynq queue.

### 4) Queue Flow
`API -> Asynq producer -> Redis queue -> worker`

- User creation enqueues `email:send_welcome` task.
- Payment gateway callbacks enqueue tasks to `callbacks` queue, then worker updates local transaction settlement.
- Worker runs as separate process/binary and consumes queued tasks.
- Worker and HTTP server both support graceful shutdown with SIGINT/SIGTERM.

## CI/CD

GitHub Actions workflow (`.github/workflows/ci.yml`) runs:
- backend lint + tests
- frontend lint + tests + build

## Security Defaults

- bcrypt password hashing
- JWT secrets from environment variables
- no hardcoded production secrets
- cookie-based auth (`withCredentials` frontend)
- CSRF protection for mutating endpoints
- middleware-based auth
- request validation
- centralized error response format:

```json
{
  "status": "error",
  "message": "string",
  "details": "optional"
}
```

## CSRF + Cookie Troubleshooting

If login returns `403 missing csrf cookie`:

1. Ensure frontend calls `GET /api/v1/auth/csrf` before login/refresh/logout.
2. Ensure frontend sends credentials (`withCredentials: true`).
3. Ensure CORS origin matches exactly and credentials are enabled.
4. For local HTTP dev:
   - `SECURITY_COOKIE_SECURE=false`
   - `SECURITY_COOKIE_SAME_SITE=lax`
   - Open frontend and API with the same loopback host (`localhost` with `localhost`, or `127.0.0.1` with `127.0.0.1`).
5. For cross-origin HTTPS (`https://gue.test` -> `https://api.test`):
   - `SECURITY_COOKIE_SECURE=true`
   - `SECURITY_COOKIE_SAME_SITE=none`
   - `VITE_API_BASE_URL=https://api.test`
