SHELL := /bin/sh

.PHONY: run run-worker test lint migrate-up migrate-down

run:
	go run ./cmd/server

run-worker:
	go run ./cmd/server --worker

test:
	go test ./...

lint:
	go vet ./...

migrate-up:
	migrate -path migrations -database "mysql://$(DB_USER):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)" up

migrate-down:
	migrate -path migrations -database "mysql://$(DB_USER):$(DB_PASSWORD)@tcp($(DB_HOST):$(DB_PORT))/$(DB_NAME)" down 1
