# Sicherheit – Galaxis v1.0

**Datum:** 2026-03-21
**GDD-Referenz:** v1.24

---

## 1. Threat Model (Überblick)

Galaxis ist ein MMO mit bis zu 100 gleichzeitigen Spielern pro Instanz. Der Server ist
**alleinige Autorität** über den Spielzustand — kein Client-seitiger Spielstand, keine
Client-seitigen Berechnungen die der Server nicht verifiziert.

Primäre Angriffsvektoren:
- **Cheating:** Manipulierte Befehle (falsche Ressourcen, illegale Aktionen)
- **Übernahme:** Fremde Spieler-Accounts oder Admin-Konten
- **DoS:** Tick-Engine durch Befehlsflut blockieren
- **Datenleak:** Fog of War Bypass (Spieler sieht mehr als erlaubt)

---

## 2. Authentifizierung (AP3 — ausstehend)

**Abhängigkeit:** creaminds IAM Platform (`blodilo/creaminds-platform`)

### Flow
```
Spieler → Keycloak (keycloak.creaminds.de) → JWT (RS256)
       → API Gateway (Go) → JWTMiddleware → UserContext in ctx
       → WebSocket-Upgrade mit JWT im Authorization-Header
```

### JWT-Middleware (Go/chi, Skizze)
```go
// galaxis/internal/auth/middleware.go
func JWTMiddleware(jwksURL string) func(http.Handler) http.Handler {
    kc := keyfunc.NewRemote(jwksURL, keyfunc.Options{RefreshInterval: 1 * time.Hour})
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            token, err := jwt.ParseFromRequest(r, jwt.TokenFromHeader, kc.Keyfunc)
            if err != nil || !token.Valid {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), userContextKey, extractClaims(token))
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}
```

### WebSocket-Auth
Token im `Authorization: Bearer <jwt>`-Header beim WebSocket-Upgrade — **nicht** als
Query-Parameter (würde in Access-Logs landen).

### Keycloak Realm-Rollen (galaxis)
| Rolle | Vergabe | Berechtigung |
|---|---|---|
| `player` | Automatisch bei Registrierung | Kann Partien beitreten |
| `game-admin` | Manuell durch `platform-admin` | Partien erstellen, Spieler verwalten |
| `spectator` | Selbst aktivierbar | Nur beobachten |

---

## 3. Autorisierung

### Spielbefehle
Alle eingehenden Aktionen werden server-seitig gegen den aktuellen Spielzustand validiert:
- Ressourcen vorhanden?
- Einheit gehört dem Spieler?
- Aktion in diesem Tick erlaubt?
- Rate Limit nicht überschritten?

Invalide Befehle werden stillschweigend verworfen (kein Stack-Trace an Client).

### Fog of War
Der Server sendet pro Tick **nur Daten, die der Spieler sehen darf** (Sensor-Reichweite,
eigene Flotten, scouted Systeme). Kein vollständiges Galaxy-State an alle Clients.

Referenz: `sensor-fow_v1.0.md`

---

## 4. Secrets-Management

| Secret | Dev | Prod |
|---|---|---|
| PostgreSQL-Passwort | `.env` (nicht eingecheckt) | Docker `secrets:` / K8s Secret |
| Redis-Passwort | `.env` | Docker `secrets:` / K8s Secret |
| Keycloak JWKS URL | Konfigurierbar per Env-Var | Env-Var |
| API Keys (Gemini für Scraper) | `tools/.env` (nicht eingecheckt) | — |

`.env` und `tools/.env` sind in `.gitignore` — **kein Secret in Git**.

---

## 5. Rate Limiting

Im API Gateway (Go):
- **Befehlsrate:** Max. `max_action_queue_depth` (game-params) ausstehende Befehle pro Spieler
- **HTTP:** Rate Limiting per IP für REST-Endpunkte (Generator-API, Admin)
- **WebSocket:** Befehlsflut durch Queue-Tiefenbegrenzung abgefangen

---

## 6. Datenbankzugriff

- PostgreSQL nur intern erreichbar (kein Port nach außen im Prod-Setup)
- Redis nur intern (kein öffentlicher Port)
- DB-Zugangsdaten per Env-Var, nie hardcoded
- Prepared Statements / pgx — kein SQL-Injection-Risiko durch String-Konkatenation

---

## 7. Frontend (Vite/React)

- Kein Spielzustand im LocalStorage (nur visuelle Präferenzen via `VisualParamsContext`)
- JWTs werden nach Login im RAM gehalten (kein `localStorage`)
- HTTPS in Prod (TLS via nginx)
- CSP-Header über nginx

---

## 8. Offene Punkte (AP3)

- [ ] JWKS-Middleware implementieren (`galaxis/internal/auth/`)
- [ ] Rate-Limiting-Middleware im API Gateway
- [ ] Fog-of-War serverseitige Filterung in der Tick-Engine
- [ ] Input-Validierung für alle Befehlstypen (Fleet, Production, Trade)
