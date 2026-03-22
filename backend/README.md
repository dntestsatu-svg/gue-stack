# Backend Starter

Run from `backend/`.

Configuration source of truth:
- use root `.env` only (`../.env` from backend directory)
- backend local env files are removed to avoid duplicated keys

## New Domain Migrations

Added migration pair:
- `migrations/000002_create_toko_domain.up.sql`
- `migrations/000002_create_toko_domain.down.sql`
- `migrations/000003_add_user_role_and_active.up.sql`
- `migrations/000003_add_user_role_and_active.down.sql`
- `migrations/000005_add_platform_fee_and_update_toko_charge.up.sql`
- `migrations/000005_add_platform_fee_and_update_toko_charge.down.sql`

Tables:
- `tokos`
- `balances`
- `transactions`
- `toko_users`
- `payments`

## Apply Migration

```bash
migrate -path ./migrations -database "mysql://root:password@tcp(localhost:3306)/gue" up
```

## Migrate Fresh (Reset Total)

Mode ini akan:
1. drop database target
2. create ulang database kosong
3. apply semua migration
4. seed payments
5. bootstrap single dev user

```bash
go run ./cmd/initdb --fresh
```

Jika `APP_ENV=production`, `--fresh` diblokir by default. Gunakan explicit override:

```bash
go run ./cmd/initdb --fresh --allow-production-fresh
```

Alternatif via Makefile:

```bash
make migrate-fresh
make migrate-fresh-force
```

## Seed Dummy Dataset

Untuk kebutuhan local QA / demo data, `initdb` sekarang juga mendukung dummy seeding yang repeatable:

```bash
go run ./cmd/initdb --seed
```

Fitur ini akan:
- memastikan ada 1 `superadmin` dummy
- membuat `50` admin dummy
- membuat jumlah `user/karyawan` acak di bawah setiap admin
- membuat `1-3` toko per admin
- selalu membuat `balance` untuk setiap toko
- mengisi transaksi dummy agar dashboard dan histori transaksi langsung terisi

Untuk reset total lalu seed ulang:

```bash
go run ./cmd/initdb --fresh --seed
```

Optional env override:

```env
DUMMY_SEED_PASSWORD=SeedPassword123!
DUMMY_SEED_ADMIN_COUNT=50
DUMMY_SEED_MAX_EMPLOYEES_PER_ADMIN=5
DUMMY_SEED_MAX_TOKOS_PER_ADMIN=3
DUMMY_SEED_TRANSACTIONS_PER_TOKO_MIN=12
DUMMY_SEED_TRANSACTIONS_PER_TOKO_MAX=32
DUMMY_SEED_RANDOM_SEED=20260321
DUMMY_SEED_DOMAIN=seed.gue.local
```

## Seed Payments

```bash
go run ./cmd/seed
```

Seeder is idempotent (safe to rerun without duplicating existing records).

## Build Binaries (server + worker)

Recommended output directory: `backend/bin/`

### PowerShell (Windows)

Build native binaries:

```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/build.ps1 -Target current
```

Build Linux binaries:

```powershell
powershell -ExecutionPolicy Bypass -File ./scripts/build.ps1 -Target linux
```

### Makefile (Linux/macOS/Git-Bash)

```bash
make build        # native target -> ./bin/server ./bin/worker ./bin/initdb
make build-linux  # linux target -> ./bin/server ./bin/worker ./bin/initdb
```

## GORM Models

Model files:
- `model/toko.go`
- `model/balance.go`
- `model/transaction.go`
- `model/toko_user.go`
- `model/payment.go`
- `model/user.go` (updated relationship)

## Payment Gateway Integration

Integrated against the `API Qris & VA-V3` collection.

### Environment variables

```env
PAYMENT_GATEWAY_BASE_URL=https://rest.otomatis.vip
PAYMENT_GATEWAY_TIMEOUT=15s
PAYMENT_GATEWAY_DEFAULT_CLIENT=
PAYMENT_GATEWAY_DEFAULT_KEY=
PAYMENT_GATEWAY_MERCHANT_UUID=
PAYMENT_GATEWAY_CALLBACK_SECRET= # backward-compatible fallback for merchant UUID
PAYMENT_GATEWAY_WEBHOOK_SECRET=
PAYMENT_GATEWAY_PLATFORM_FEE_PERCENT=3
```

Cache/Redis separation:
- Memcached: query/API response cache (`CACHE_DRIVER=memcached`)
- Redis: refresh token/session + Asynq queue backend only

### Internal endpoints (JWT protected)

- `GET /api/v1/user/me`
- `GET /api/v1/dashboard/overview`
- `GET /api/v1/transactions/history`
- `GET /api/v1/tokos`
- `POST /api/v1/tokos` (max 3 tokos/user)
- `GET /api/v1/tokos/balances` (available + settlement balance toko)
- `PATCH /api/v1/tokos/:id/settlement` (manual settlement, role: dev/superadmin)
- `POST /api/v1/users` (role must not be `user`)
- `PATCH /api/v1/users/:id/role` (role must be `dev` or `superadmin`)

### Internal payment bridge endpoints (Bearer = toko token)

- `POST /api/v1/payments/gateway/generate`
- `POST /api/v1/payments/gateway/check-status/:trx_id`
- `POST /api/v1/payments/gateway/inquiry`
- `POST /api/v1/payments/gateway/transfer`
- `POST /api/v1/payments/gateway/transfer/check-status/:partner_ref_no`
- `POST /api/v1/payments/gateway/balance`

### Callback endpoints (public)

- `POST /api/v1/payments/gateway/callback/qris`
- `POST /api/v1/payments/gateway/callback/transfer`

If `PAYMENT_GATEWAY_WEBHOOK_SECRET` is set, callbacks must send header:
- `X-Callback-Secret: <secret>`

Callbacks are processed asynchronously via Asynq queue (`callbacks` queue) by worker process.

## Dashboard Realtime Payload

`GET /api/v1/dashboard/overview` now includes:
- `status_series` for chart `success` vs `failed/expired`
- `latest_success_orders` for table latest success orders
- `metrics.project_profit` from `platform_fee` success transactions
- `external_balance.pending_balance` and `external_balance.available_balance` from external gateway

## Internal API Examples (Postman/HTTP)

Use these files to test only project internal endpoints:

- Postman collection: `docs/postman/GUE-Internal-API.postman_collection.json`
- Postman environment: `docs/postman/GUE-Local.postman_environment.json`
- HTTP examples: `docs/postman/internal-api-examples.http`

Setup:

1. Import collection and environment into Postman.
2. Fill env values: `toko_token`, `merchant_uuid`, `gateway_client`, `callback_secret`, and user-management vars.
3. Execute `Login` first to auto-store `access_token` and `refresh_token`.
