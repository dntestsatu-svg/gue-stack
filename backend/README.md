# Backend Starter

Run from `backend/`.

## New Domain Migrations

Added migration pair:
- `migrations/000002_create_toko_domain.up.sql`
- `migrations/000002_create_toko_domain.down.sql`

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

## Seed Payments

```bash
go run ./cmd/seed
```

Seeder is idempotent (safe to rerun without duplicating existing records).

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
PAYMENT_GATEWAY_CALLBACK_SECRET=
```

### Internal endpoints (JWT protected)

- `POST /api/v1/payments/gateway/generate`
- `POST /api/v1/payments/gateway/check-status/:trx_id`
- `POST /api/v1/payments/gateway/inquiry`
- `POST /api/v1/payments/gateway/transfer`
- `POST /api/v1/payments/gateway/transfer/check-status/:partner_ref_no`
- `POST /api/v1/payments/gateway/balance/:merchant_uuid`

### Callback endpoints (public)

- `POST /api/v1/payments/gateway/callback/qris`
- `POST /api/v1/payments/gateway/callback/transfer`

If `PAYMENT_GATEWAY_CALLBACK_SECRET` is set, callbacks must send header:
- `X-Callback-Secret: <secret>`
