FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/server ./cmd/server \
    && CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /out/worker ./cmd/worker

FROM gcr.io/distroless/base-debian12 AS runtime

WORKDIR /app

COPY --from=builder /out/server /usr/local/bin/server
COPY --from=builder /out/worker /usr/local/bin/worker
COPY --from=builder /app/docs /app/docs
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/.env.production /app/.env.production

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/server"]
