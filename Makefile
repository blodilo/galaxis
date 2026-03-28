.PHONY: build run-server run-gen db-up db-down migrate lint test dev install-hooks install-tools

# .env automatisch laden (falls vorhanden)
-include .env
export

# ── Build ──────────────────────────────────────────────────────────────────────
build:
	go build -o bin/galaxis-server  ./cmd/server
	go build -o bin/galaxis-devctl  ./cmd/devctl
	go build -o bin/galaxy-gen      ./cmd/galaxy-gen

# ── Run ────────────────────────────────────────────────────────────────────────
run-server:
	go run ./cmd/server --config game-params_v1.3.yaml --addr :8090

run-gen:
	go run ./cmd/galaxy-gen --config game-params_v1.3.yaml

# ── Database ───────────────────────────────────────────────────────────────────
db-up:
	docker compose up -d postgres redis

db-down:
	docker compose down

migrate:
	go run ./cmd/server --config game-params_v1.3.yaml --migrate-only

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

# ── Hooks ───────────────────────────────────────────────────────────────────
install-hooks:
	@ln -sf ../../scripts/hooks/pre-commit  .git/hooks/pre-commit
	@ln -sf ../../scripts/hooks/post-commit .git/hooks/post-commit
	@echo "Git-Hooks installiert (pre-commit, post-commit)"

install-tools:
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh \
	  | sh -s -- -b $$(go env GOPATH)/bin
	@echo "golangci-lint installiert → $$(go env GOPATH)/bin/golangci-lint"
