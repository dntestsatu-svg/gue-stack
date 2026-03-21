# GUE Fullstack Starter Kit

Production-quality fullstack starter kit with clean architecture, JWT auth, Redis/Memcached caching abstraction, Asynq jobs, Vue 3 frontend, and CI/CD-ready workflows.

## Tech Stack

### Backend
- Go + Gin
- MySQL
- Redis (refresh-token/session store + Asynq queue backend)
- Memcached (optional query cache via abstraction)
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
docker compose up --build
```

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

## API Endpoints

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/user/me`
- `POST /api/v1/payments/gateway/generate` (auth)
- `POST /api/v1/payments/gateway/check-status/:trx_id` (auth)
- `POST /api/v1/payments/gateway/inquiry` (auth)
- `POST /api/v1/payments/gateway/transfer` (auth)
- `POST /api/v1/payments/gateway/transfer/check-status/:partner_ref_no` (auth)
- `POST /api/v1/payments/gateway/balance/:merchant_uuid` (auth)
- `POST /api/v1/payments/gateway/callback/qris` (public callback)
- `POST /api/v1/payments/gateway/callback/transfer` (public callback)

## Postman and HTTP Examples

Internal API examples are available so the team can use project endpoints directly (not vendor endpoints):

- Postman collection: `backend/docs/postman/GUE-Internal-API.postman_collection.json`
- Postman environment: `backend/docs/postman/GUE-Local.postman_environment.json`
- HTTP examples (VS Code REST Client): `backend/docs/postman/internal-api-examples.http`

Recommended flow:

1. Import collection + environment into Postman.
2. Fill variables: `merchant_uuid`, `gateway_client`, `gateway_client_key`, `callback_secret`.
3. Run in order: `Register` or `Login` -> token-dependent endpoints.

## Auth + Authorization Rules

### Guest users
- Allowed: `/login`, `/register`
- Redirected from `/dashboard` to `/login`

### Authenticated users
- Allowed: `/dashboard`
- Redirected from `/login` and `/register` to `/dashboard`

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
`login/register -> access token + refresh token -> access expires -> refresh endpoint rotates tokens`

- Access token is short-lived and sent in `Authorization: Bearer`.
- Refresh token is long-lived JWT with unique token ID (`jti`).
- Refresh token ID is persisted in Redis.
- Refresh flow validates JWT signature + Redis presence, then rotates token IDs.
- Logout revokes refresh token by deleting its token ID from Redis.

### 3) Cache Strategy (Read-through)

- Cache is queried first for read-heavy data (`/api/v1/user/me`).
- On cache miss, data is loaded from DB and stored in cache.
- Business logic depends only on cache interface (`Get`, `Set`, `Delete`).
- Cache backend can be switched between Redis, Memcached, or Noop.

### 4) Queue Flow
`API -> Asynq producer -> Redis queue -> worker`

- Registration enqueues `email:send_welcome` task.
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
