#!/usr/bin/env bash
# scripts/start-local.sh — Galaxis lokaler Dev-Stack
# Devlaunchpad-Kontrakt:
#   Exitcode 0    = Stack bereit (oder bereits laufend)
#   Exitcode ≠ 0  = Fehler (Meldung auf stderr)
#   PROGRESS:<0-100>:<Meldung> auf stdout
#   Kein Browser-Start — übernimmt Devlaunchpad
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

LOGDIR="$PROJECT_DIR/.dev-logs"
mkdir -p "$LOGDIR"

# ── npm-Binary auflösen ───────────────────────────────────────────────────────
# nvm-Versionen bevorzugen, sonst system-npm
if command -v npm &>/dev/null; then
  NPM="$(command -v npm)"
elif [ -d "${HOME}/.nvm" ]; then
  # nvm nicht initialisiert — lädt default-Version
  export NVM_DIR="${HOME}/.nvm"
  # shellcheck source=/dev/null
  source "$NVM_DIR/nvm.sh" --no-use
  NPM="$(nvm which default 2>/dev/null)" || { echo "npm nicht gefunden" >&2; exit 1; }
else
  echo "npm nicht gefunden" >&2
  exit 1
fi

# ── Credentials aus Keyring ───────────────────────────────────────────────────
echo "PROGRESS:2:Lade Credentials …"
DB_PASS=$(secret-tool lookup service galaxis-local account postgres 2>/dev/null || true)
if [ -z "$DB_PASS" ]; then
  # Fallback: lokales Docker-Dev-Default
  DB_PASS="galaxis_dev"
fi
DB_URL="postgres://galaxis:${DB_PASS}@localhost:5432/galaxis?sslmode=disable"

# ── Bereits laufend? ──────────────────────────────────────────────────────────
if curl -s http://localhost:5174 &>/dev/null && curl -s http://localhost:8090/health &>/dev/null; then
  echo "PROGRESS:100:Stack bereits laufend"
  exit 0
fi

# ── 1. Docker: Postgres + Redis ───────────────────────────────────────────────
echo "PROGRESS:5:Starte Postgres + Redis …"
docker compose up -d postgres redis 2>&1 | tee -a "$LOGDIR/docker.log" >/dev/null

echo "PROGRESS:15:Warte auf Postgres …"
for i in $(seq 1 30); do
  docker compose exec -T postgres pg_isready -U galaxis -d galaxis &>/dev/null && break
  sleep 1
  [ "$i" -eq 30 ] && { echo "Postgres-Start Timeout" >&2; exit 1; }
done
echo "PROGRESS:25:Postgres bereit"

# ── 2. Backend ────────────────────────────────────────────────────────────────
echo "PROGRESS:30:Baue Backend …"
go build -o bin/server ./cmd/server 2>&1 | tee -a "$LOGDIR/build.log" >/dev/null

echo "PROGRESS:50:Starte Backend (Port 8090) …"
# DATABASE_URL wird NUR an diesen Prozess übergeben, nicht exportiert
DATABASE_URL="$DB_URL" bin/server \
  --config game-params_v1.8.yaml \
  --addr :8090 \
  >"$LOGDIR/server.log" 2>&1 &
SERVER_PID=$!

echo "PROGRESS:55:Warte auf Backend …"
for i in $(seq 1 30); do
  curl -s http://localhost:8090/health &>/dev/null && break
  sleep 1
  if ! kill -0 "$SERVER_PID" 2>/dev/null; then
    echo "Backend abgestürzt — siehe $LOGDIR/server.log" >&2
    exit 1
  fi
  [ "$i" -eq 30 ] && { echo "Backend-Start Timeout" >&2; exit 1; }
done
echo "PROGRESS:70:Backend bereit"

# ── 3. Frontend ───────────────────────────────────────────────────────────────
echo "PROGRESS:75:Starte Frontend (Port 5174) …"
(cd frontend && "$NPM" run dev) >"$LOGDIR/frontend.log" 2>&1 &
FRONTEND_PID=$!

echo "PROGRESS:85:Warte auf Vite …"
for i in $(seq 1 30); do
  curl -s http://localhost:5174 &>/dev/null && break
  sleep 1
  if ! kill -0 "$FRONTEND_PID" 2>/dev/null; then
    echo "Frontend abgestürzt — siehe $LOGDIR/frontend.log" >&2
    exit 1
  fi
  [ "$i" -eq 30 ] && { echo "Frontend-Start Timeout" >&2; exit 1; }
done
echo "PROGRESS:100:Stack bereit — http://localhost:5174"

# ── Cleanup bei Beenden ───────────────────────────────────────────────────────
cleanup() {
  echo ""
  kill "$SERVER_PID" "$FRONTEND_PID" 2>/dev/null || true
  docker compose stop 2>/dev/null || true
}
trap cleanup EXIT INT TERM

wait "$SERVER_PID" "$FRONTEND_PID"
