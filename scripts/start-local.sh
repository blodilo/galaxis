#!/usr/bin/env bash
# scripts/start-local.sh — Galaxis Dev-Stack via galaxis-devctl
# Devlaunchpad-Kontrakt:
#   Exitcode 0    = Stack bereit (oder bereits laufend)
#   Exitcode ≠ 0  = Fehler (Meldung auf stderr)
#   PROGRESS:<0-100>:<Meldung> auf stdout
#   Kein Browser-Start — übernimmt Devlaunchpad
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

DEVCTL_URL="http://localhost:9191"
LOGDIR="$PROJECT_DIR/.dev-logs"
mkdir -p "$LOGDIR"

tcp_alive() { nc -z localhost "$1" 2>/dev/null; }
http_alive() { curl -s --max-time 1 "$1" &>/dev/null; }

# ── Bereits laufend? ──────────────────────────────────────────────────────────
if http_alive "$DEVCTL_URL/api/status" \
   && http_alive "http://localhost:8081/health" \
   && tcp_alive 5175; then
  echo "PROGRESS:100:Stack bereits laufend"
  exit 0
fi

# ── 1. devctl bauen ───────────────────────────────────────────────────────────
echo "PROGRESS:5:Baue galaxis-devctl …"
mkdir -p bin
go build -o bin/galaxis-devctl ./cmd/devctl 2>>"$LOGDIR/devctl-build.log" \
  || { echo "devctl-Build fehlgeschlagen — siehe $LOGDIR/devctl-build.log" >&2; exit 1; }

# ── 2. devctl starten (falls noch nicht laufend) ─────────────────────────────
echo "PROGRESS:10:Starte devctl …"
if ! http_alive "$DEVCTL_URL/api/status"; then
  nohup ./bin/galaxis-devctl >>"$LOGDIR/devctl.log" 2>&1 &
  for i in $(seq 1 20); do
    http_alive "$DEVCTL_URL/api/status" && break
    sleep 0.5
    [ "$i" -eq 20 ] && { echo "devctl antwortet nicht" >&2; exit 1; }
  done
fi
echo "PROGRESS:15:devctl bereit"

# ── 3. Alle Komponenten starten ───────────────────────────────────────────────
echo "PROGRESS:18:Starte Stack-Komponenten …"
for comp in postgres nats galaxis-api galaxis-frontend; do
  curl -s -X POST "$DEVCTL_URL/api/start/$comp" >/dev/null
done

# ── 4. Auf Postgres warten ────────────────────────────────────────────────────
echo "PROGRESS:20:Warte auf Postgres (:5432) …"
for i in $(seq 1 60); do
  tcp_alive 5432 && break
  sleep 1
  [ "$i" -eq 60 ] && { echo "Postgres-Start Timeout" >&2; exit 1; }
done
echo "PROGRESS:35:Postgres bereit"

# ── 5. Auf NATS warten ────────────────────────────────────────────────────────
echo "PROGRESS:37:Warte auf NATS (:4222) …"
for i in $(seq 1 30); do
  tcp_alive 4222 && break
  sleep 1
  [ "$i" -eq 30 ] && { echo "NATS-Start Timeout" >&2; exit 1; }
done
echo "PROGRESS:50:NATS bereit"

# ── 6. Auf Backend warten ────────────────────────────────────────────────────
echo "PROGRESS:52:Warte auf galaxis-api (:8081) …"
for i in $(seq 1 60); do
  http_alive "http://localhost:8081/health" && break
  sleep 1
  [ "$i" -eq 60 ] && { echo "galaxis-api Timeout" >&2; exit 1; }
done
echo "PROGRESS:75:galaxis-api bereit"

# ── 7. Auf Frontend warten ───────────────────────────────────────────────────
echo "PROGRESS:77:Warte auf Frontend (:5175) …"
for i in $(seq 1 60); do
  tcp_alive 5175 && break
  sleep 1
  [ "$i" -eq 60 ] && { echo "Frontend-Start Timeout" >&2; exit 1; }
done
echo "PROGRESS:100:Stack bereit — http://localhost:5175"
