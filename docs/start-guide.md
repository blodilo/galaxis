# Start Guide — Galaxis (Lokal)

## Voraussetzungen

| Tool | Version | Prüfen |
|------|---------|--------|
| Go | ≥ 1.22 | `go version` |
| Node.js | ≥ 22 (via nvm) | `node --version` |
| npm | ≥ 10 | `npm --version` |
| Docker + Compose | aktuell | `docker compose version` |
| `secret-tool` | (GNOME Keyring) | `secret-tool --version` |
| `curl` | beliebig | `curl --version` |

## Erster Start (einmalig)

### 1. Go-Abhängigkeiten laden
```bash
go mod download
```

### 2. Node-Abhängigkeiten laden
```bash
cd frontend && npm install
```

### 3. (Optional) DB-Passwort im Keyring hinterlegen
Das Script verwendet als Fallback `galaxis_dev` (Docker-Default).
Für explizite Keyring-Speicherung:
```bash
secret-tool store --label="Galaxis Local DB" service galaxis-local account postgres
# Passwort eingeben: galaxis_dev
```

## Starten

### Via Devlaunchpad
Button **"Start Local"** im Devlaunchpad — öffnet `http://localhost:5173` automatisch.

### Manuell
```bash
bash scripts/start-local.sh
```

## Dienste & Ports

| Dienst | Port | URL |
|--------|------|-----|
| Frontend (Vite) | 5174 | http://localhost:5174 |
| Backend (Go) | 8090 | http://localhost:8090 |
| Postgres | 5432 | localhost:5432/galaxis |
| Redis | 6379 | localhost:6379 |

## Logs

```
.dev-logs/
  server.log    # Go-Backend
  frontend.log  # Vite Dev Server
  docker.log    # Docker Compose Output
  build.log     # go build Output
```

## Stoppen

- **Devlaunchpad:** Button "Stop"
- **Manuell:** `Ctrl+C` im Terminal — stoppt Backend, Frontend und Docker-Container

## Fehlersuche

| Symptom | Ursache | Lösung |
|---------|---------|--------|
| `Backend abgestürzt` | Migrations fehlgeschlagen oder DB nicht erreichbar | `cat .dev-logs/server.log` |
| `Frontend abgestürzt` | npm-Abhängigkeiten fehlen | `cd frontend && npm install` |
| `Postgres-Start Timeout` | Docker-Daemon nicht gestartet | `sudo systemctl start docker` |
| `npm nicht gefunden` | nvm-Default nicht gesetzt | `nvm alias default 22` |
| Backend-Compilefehler | Go-Code nicht kompilierbar | `cat .dev-logs/build.log` |
