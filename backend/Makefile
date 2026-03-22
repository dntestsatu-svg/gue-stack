SHELL := /bin/sh

BIN_DIR ?= ./bin

.PHONY: run run-worker test lint migrate-up migrate-down seed initdb migrate-fresh migrate-fresh-force build build-linux clean run-bin run-worker-bin

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

seed:
	go run ./cmd/seed

initdb:
	go run ./cmd/initdb

migrate-fresh:
	go run ./cmd/initdb --fresh

migrate-fresh-force:
	go run ./cmd/initdb --fresh --allow-production-fresh

build:
	mkdir -p $(BIN_DIR)
	go build -ldflags='-s -w' -o $(BIN_DIR)/server ./cmd/server
	go build -ldflags='-s -w' -o $(BIN_DIR)/worker ./cmd/worker
	go build -ldflags='-s -w' -o $(BIN_DIR)/initdb ./cmd/initdb

build-linux:
	mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o $(BIN_DIR)/server ./cmd/server
	CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o $(BIN_DIR)/worker ./cmd/worker
	CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o $(BIN_DIR)/initdb ./cmd/initdb

run-bin:
	$(BIN_DIR)/server

run-worker-bin:
	$(BIN_DIR)/worker

clean:
	rm -rf $(BIN_DIR)
