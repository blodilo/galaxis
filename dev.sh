#!/usr/bin/env bash
# dev.sh — Galaxis Dev-Stack starten
# Startet DB, Backend und Frontend; öffnet den Browser automatisch.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# .env laden
if [[ -f .env ]]; then
  set -a; source .env; set +a
fi

NPM="${HOME}/.nvm/versions/node/v22.22.0/bin/npm"
LOGDIR="$SCRIPT_DIR/.dev-logs"
mkdir -p "$LOGDIR"

# ── 1. Docker-Dienste ────────────────────────────────────────────────────────
echo "[1/4] Starte Postgres + Redis …"
docker compose up -d postgres redis

echo "      Warte auf Postgres …"
until docker compose exec -T postgres pg_isready -U "${POSTGRES_USER:-galaxis}" -d "${POSTGRES_DB:-galaxis}" &>/dev/null; do
  sleep 1
done
echo "      Postgres: bereit"

# ── 2. Backend ───────────────────────────────────────────────────────────────
echo "[2/4] Starte Backend (Port 8080) …"
go run ./cmd/server --config game-params_v1.2.yaml \
  >"$LOGDIR/server.log" 2>&1 &
SERVER_PID=$!

echo "      Warte auf Backend …"
for i in $(seq 1 30); do
  curl -s http://localhost:8080/health &>/dev/null && break
  sleep 1
done
echo "      Backend: bereit (PID $SERVER_PID)"

# ── 3. Frontend ──────────────────────────────────────────────────────────────
echo "[3/4] Starte Frontend (Port 5173) …"
(cd frontend && "$NPM" run dev) >"$LOGDIR/frontend.log" 2>&1 &
FRONTEND_PID=$!

echo "      Warte auf Vite …"
for i in $(seq 1 20); do
  curl -s http://localhost:5173 &>/dev/null && break
  sleep 1
done
echo "      Frontend: bereit (PID $FRONTEND_PID)"

# ── 4. Browser öffnen ────────────────────────────────────────────────────────
echo "[4/4] Öffne Browser …"
xdg-open http://localhost:5173 2>/dev/null || true

echo ""
echo "  Stack läuft. Logs:"
echo "    Backend:  $LOGDIR/server.log"
echo "    Frontend: $LOGDIR/frontend.log"
echo ""
echo "  Stoppen: Ctrl+C"

# Beim Beenden beide Prozesse sauber beenden
trap "echo ''; echo 'Stoppe Stack …'; kill $SERVER_PID $FRONTEND_PID 2>/dev/null; docker compose stop; echo 'Fertig.'" EXIT INT TERM

wait $SERVER_PID $FRONTEND_PID
