.PHONY: build run-server run-gen db-up db-down migrate lint test

# ── Build ──────────────────────────────────────────────────────────────────────
build:
	go build -o bin/server   ./cmd/server
	go build -o bin/galaxy-gen ./cmd/galaxy-gen

# ── Run ────────────────────────────────────────────────────────────────────────
run-server:
	go run ./cmd/server --config game-params_v1.1.yaml

run-gen:
	go run ./cmd/galaxy-gen --config game-params_v1.1.yaml

# ── Database ───────────────────────────────────────────────────────────────────
db-up:
	docker compose up -d postgres redis

db-down:
	docker compose down

migrate:
	go run ./cmd/server --config game-params_v1.1.yaml --migrate-only

# ── Dev ────────────────────────────────────────────────────────────────────────
lint:
	golangci-lint run ./...

test:
	go test ./... -race -count=1

# ── Frontend ───────────────────────────────────────────────────────────────────
NPM := $(HOME)/.nvm/versions/node/v22.22.0/bin/npm

frontend-install:
	cd frontend && $(NPM) install

frontend-dev:
	cd frontend && $(NPM) run dev

frontend-build:
	cd frontend && $(NPM) run build
