# NATS Implementierung – Galaxis v1.0

**Datum:** 2026-03-28
**Basis:** `docs/messaging_concept_v1.0.md`, `architecture_v1.1.md`
**Scope:** NATS-Server-Konfiguration, Go-Adapter, Browser-Client (nats.ws), Auth-Flow, JetStream-Streams

---

## 1. Warum NATS — Entscheidungszusammenfassung

NATS ist der einzige evaluierte Broker der alle drei Tier-Anforderungen nativ erfüllt:
- **Tier 1** (at-most-once): NATS Core, <1ms, kein Overhead
- **Tier 2** (at-least-once, persistent): JetStream, vollwertige Streams, Replay, Durable-Consumer
- **Request/Reply** (Player-Aktionen): nativer Einzeiler `nc.Request()`, kein Eigencode

Entscheidend für den Browser: **NATS unterstützt WebSocket nativ** (`nats-server -ws`). Der Browser verbindet sich über `nats.ws` (npm, Apache-2.0 ✅) direkt mit dem NATS-Server — kein custom WS-Gateway nötig.

Lizenzen:
- `nats-server`: Apache-2.0 ✅
- `nats.go` (Go-Client): Apache-2.0 ✅
- `nats.ws` (Browser-Client): Apache-2.0 ✅

---

## 2. nats-server Konfiguration

### 2.1 Ports

```
TCP  :4222  — Go-Services, KI-Clients
WS   :4223  — Browser (nats.ws, HTTP Upgrade)
MON  :8222  — HTTP Monitoring (dev only)
```

### 2.2 nats-server.conf (Entwicklung)

```hcl
# nats-server.conf
port: 4222
http_port: 8222

jetstream {
  store_dir: "/data/nats"
  max_memory_store: 256MB
  max_file_store: 1GB
}

websocket {
  port: 4223
  no_tls: true   # Dev only — TLS in Produktion pflicht
  compression: true
}

authorization {
  # Dev: kein Auth. Produktion: NKeys / JWT-Auth (Abschnitt 5)
  allow_user_password_based_connections: true
}
```

### 2.3 docker-compose.yml Ergänzung

```yaml
nats:
  image: nats:2.10-alpine          # Apache-2.0
  command: ["-c", "/etc/nats/nats-server.conf", "-js"]
  ports:
    - "4222:4222"   # Go-Services
    - "4223:4223"   # Browser WS
    - "8222:8222"   # Monitoring
  volumes:
    - ./nats-server.conf:/etc/nats/nats-server.conf
    - nats_data:/data/nats
  healthcheck:
    test: ["CMD", "nats-server", "--help"]
    interval: 5s

volumes:
  nats_data:
```

---

## 3. JetStream Streams

Streams werden beim Server-Start via `bus.EnsureStream()` angelegt (idempotent).

| Stream | Subjects | Retention | MaxAge | Zweck |
|---|---|---|---|---|
| `TICK` | `galaxis.tick.>` | Limits | 7 Tage | Tick-History, Replay nach Reconnect |
| `ECONOMY` | `galaxis.economy.>` | Limits | 7 Tage | Stock/Order-Updates, Audit-Trail |
| `COMBAT` | `galaxis.combat.*.state` | Limits | 1 Tag | Combat-State-Snapshots (nicht Events) |
| `PLAYER` | `galaxis.player.>` | Limits | 30 Tage | Notifications auch für Offline-Spieler |

Tier-1-Subjects (`galaxis.combat.*.event`, `galaxis.ship.*.move`) landen in keinem Stream — sie sind flüchtig und latenzoptimiert.

---

## 4. Go-Adapter: `internal/bus/natsbus`

### 4.1 Struktur

```
internal/bus/
  bus.go           — Interface + Typen (Message, Subscription, StreamConfig, …)
  natsbus/
    adapter.go     — NATSAdapter implements bus.Bus
    streams.go     — EnsureStream, EnsureConsumer
    auth.go        — ScopedCredential-Generierung für Browser
  inprocbus/
    adapter.go     — InProcAdapter für Tests (kein externer Prozess)
```

### 4.2 Adapter-Kerncode (Skizze)

```go
// internal/bus/natsbus/adapter.go
package natsbus

import (
    "context"
    "time"

    "github.com/nats-io/nats.go"
    "github.com/nats-io/nats.go/jetstream"
    "galaxis/internal/bus"
)

type NATSAdapter struct {
    nc *nats.Conn
    js jetstream.JetStream
}

func New(url string) (*NATSAdapter, error) {
    nc, err := nats.Connect(url,
        nats.MaxReconnects(-1),           // unbegrenzt
        nats.ReconnectWait(2*time.Second),
        nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
            log.Printf("nats: disconnect: %v", err)
        }),
        nats.ReconnectHandler(func(_ *nats.Conn) {
            log.Printf("nats: reconnected")
        }),
    )
    if err != nil {
        return nil, err
    }
    js, err := jetstream.New(nc)
    if err != nil {
        return nil, err
    }
    return &NATSAdapter{nc: nc, js: js}, nil
}

// Tier 1: fire-and-forget
func (a *NATSAdapter) Publish(_ context.Context, msg bus.Message) error {
    nm := nats.NewMsg(msg.Subject)
    nm.Data = msg.Payload
    for k, v := range msg.Headers {
        nm.Header.Set(k, v)
    }
    return a.nc.PublishMsg(nm)
}

// Subscribe mit Wildcard-Unterstützung
func (a *NATSAdapter) Subscribe(_ context.Context, subject string, h bus.MsgHandler) (bus.Subscription, error) {
    sub, err := a.nc.Subscribe(subject, func(m *nats.Msg) {
        h(natsToMsg(m))
    })
    return &subscription{sub}, err
}

// Request/Reply — nativer NATS-Einzeiler
func (a *NATSAdapter) Request(ctx context.Context, msg bus.Message, timeout time.Duration) (bus.Message, error) {
    nm := nats.NewMsg(msg.Subject)
    nm.Data = msg.Payload
    reply, err := a.nc.RequestMsgWithContext(ctx, nm)
    if err != nil {
        return bus.Message{}, err
    }
    return natsToMsg(reply), nil
}

// Tier 2: at-least-once via JetStream
func (a *NATSAdapter) PublishDurable(_ context.Context, stream string, msg bus.Message) error {
    nm := nats.NewMsg(msg.Subject)
    nm.Data = msg.Payload
    _, err := a.js.PublishMsg(context.Background(), nm)
    return err
}

func (a *NATSAdapter) SubscribeDurable(
    ctx context.Context,
    stream, consumer string,
    startSeq uint64,
    h bus.AckHandler,
) (bus.Subscription, error) {
    cons, err := a.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
        Name:          consumer,
        DeliverPolicy: seqPolicy(startSeq),
        OptStartSeq:   startSeq,
        AckPolicy:     jetstream.AckExplicitPolicy,
    })
    if err != nil {
        return nil, err
    }
    mc, err := cons.Messages()
    if err != nil {
        return nil, err
    }
    go func() {
        for {
            m, err := mc.Next()
            if err != nil {
                return
            }
            h(natsToMsg(m.(*nats.Msg)), m.Ack)
        }
    }()
    return &jsSubscription{mc}, nil
}
```

### 4.3 Verwendung im Tick-Handler (Beispiel)

```go
// internal/economy2/production.go — nach Tick-Abschluss
func ProductionHandler(db *pgxpool.Pool, bus bus.Bus, recipes RecipeBook, mineParams MineParams) tick.Handler {
    return func(ctx context.Context, tickN int64) {
        result, err := runProductionTick(ctx, db, recipes, mineParams)
        if err != nil {
            log.Printf("economy2: production tick %d: %v", tickN, err)
            return
        }
        // Durable: Economy-Update
        for _, update := range result.StockUpdates {
            payload, _ := json.Marshal(update)
            _ = bus.PublishDurable(ctx, "ECONOMY", bus.Message{
                Subject: "galaxis.economy." + update.StarID + ".stock",
                Payload: payload,
            })
        }
        // Core (Tier 1): Tick-Signal an alle live Clients
        _ = bus.Publish(ctx, bus.Message{
            Subject: "galaxis.tick.advance",
            Payload: []byte(fmt.Sprintf(`{"tick":%d}`, tickN)),
        })
    }
}
```

---

## 5. Auth-Flow: Browser erhält scoped NATS-Credential

### 5.1 Konzept

Der Browser bekommt nach Login kein direktes NATS-Passwort. Stattdessen:

1. Browser authentifiziert sich gegen die REST-API (Session-Cookie / JWT)
2. Browser ruft `POST /api/v1/auth/nats-token` auf
3. Server generiert ein **kurzlebiges, scoped NATS-Credential** (NKey User Token)
4. Browser verbindet sich damit direkt mit dem NATS-WS-Port

Das Credential ist durch NATS-Permissions auf die Subjects dieses Spielers beschränkt — der NATS-Server erzwingt das, kein Gateway-Code nötig.

### 5.2 NATS Permissions pro Spieler

```json
{
  "publish": [
    "galaxis.action.>"
  ],
  "subscribe": [
    "galaxis.tick.advance",
    "galaxis.economy.{starID_1}.>",
    "galaxis.combat.{systemID}.event",
    "galaxis.player.{playerID}.>"
  ],
  "deny_publish": [
    "galaxis.tick.>",
    "galaxis.player.>"
  ]
}
```

Der Spieler kann:
- **Publizieren** auf `galaxis.action.*` (eigene Aktionen)
- **Abonnieren** auf seine Economy-Nodes, öffentliche Combat-Events, eigene Notifications

Der Spieler kann **nicht**:
- Auf andere Spieler-Subjects schreiben
- Auf interne Service-Subjects zugreifen

### 5.3 Token-Generierung (Server-seitig, Go)

```go
// internal/bus/natsbus/auth.go
import "github.com/nats-io/nkeys"  // Apache-2.0

func ScopedUserToken(playerID string, allowedStars []string) (string, error) {
    // Operator- und Account-Keys liegen als Secrets vor
    // User-Key wird per-request frisch generiert (ephemeral)
    userKP, err := nkeys.CreateUser()
    if err != nil {
        return "", err
    }
    userPub, _ := userKP.PublicKey()

    // Subjects auf Basis der Spieler-Assets berechnen
    subscribePerms := buildSubscribePerms(playerID, allowedStars)

    // JWT mit Permissions und TTL (15 Minuten)
    claims := jwt.NewUserClaims(userPub)
    claims.Expires = time.Now().Add(15 * time.Minute).Unix()
    claims.Permissions.Pub.Allow = nats.StringList{"galaxis.action.>"}
    claims.Permissions.Sub.Allow = subscribePerms
    claims.Permissions.Sub.Deny  = nats.StringList{"galaxis.player.>"}
    // (playerID-spezifisches Subject wird explizit erlaubt)
    claims.Permissions.Sub.Allow = append(
        subscribePerms,
        "galaxis.player."+playerID+".>",
    )

    // Signieren mit Account-Signing-Key (liegt in GNOME Keyring / Vault)
    return claims.Encode(accountSigningKey)
}
```

Der Token hat TTL 15 Minuten. Der Browser-Client erneuert ihn via REST kurz vor Ablauf (silent refresh).

---

## 6. Browser-Client: nats.ws

### 6.1 Installation

```bash
npm install nats.ws    # Apache-2.0 ✅ — ~45KB gzipped
```

`nats.ws` ist das offizielle NATS-TypeScript-SDK für Browser. Es nutzt die native WebSocket-API und unterstützt JetStream, Request/Reply, Wildcards — identisches API wie `nats.go`.

### 6.2 Verbindungsaufbau (TypeScript)

```typescript
// src/lib/nats.ts
import { connect, NatsConnection, StringCodec, JSONCodec } from 'nats.ws'

const jc = JSONCodec()

let nc: NatsConnection | null = null

export async function connectNATS(token: string): Promise<NatsConnection> {
  nc = await connect({
    servers: 'ws://localhost:4223',   // Produktion: wss://nats.galaxis.game:4223
    authenticator: tokenAuthenticator(token),
    reconnect: true,
    maxReconnectAttempts: -1,         // unbegrenzt
    reconnectTimeWait: 2000,
  })

  // Automatisch neu verbinden nach Token-Refresh
  nc.closed().then(() => console.warn('nats: connection closed'))

  return nc
}

export function getNATS(): NatsConnection {
  if (!nc) throw new Error('NATS not connected')
  return nc
}
```

### 6.3 Token holen und verbinden

```typescript
// src/lib/auth.ts
export async function initNATSConnection(): Promise<void> {
  const res = await fetch('/api/v1/auth/nats-token', {
    method: 'POST',
    credentials: 'include',         // Session-Cookie
  })
  const { token, expires_in } = await res.json()

  await connectNATS(token)

  // Silent refresh 60s vor Ablauf
  setTimeout(initNATSConnection, (expires_in - 60) * 1000)
}
```

### 6.4 Events empfangen (JetStream Push Consumer)

```typescript
// src/hooks/useEconomyUpdates.ts
import { useEffect } from 'react'
import { getNATS, jc } from '../lib/nats'

export function useEconomyUpdates(starID: string, onUpdate: (data: StockUpdate) => void) {
  useEffect(() => {
    const nc = getNATS()
    const sub = nc.subscribe(`galaxis.economy.${starID}.stock`)

    ;(async () => {
      for await (const msg of sub) {
        onUpdate(jc.decode(msg.data) as StockUpdate)
      }
    })()

    return () => { sub.unsubscribe() }
  }, [starID])
}
```

### 6.5 Player-Aktion mit Request/Reply

```typescript
// src/api/actions.ts
import { getNATS, jc } from '../lib/nats'

export async function moveShip(shipID: string, targetSystemID: string): Promise<MoveReply> {
  const nc = getNATS()

  const reply = await nc.request(
    'galaxis.action.ship.move',
    jc.encode({ ship_id: shipID, target_system: targetSystemID }),
    { timeout: 5000 }
  )

  return jc.decode(reply.data) as MoveReply
}
// Kein Polling, kein REST-Roundtrip — direktes Request/Reply in <5ms
```

### 6.6 Verpasste Events nachholen (JetStream Replay)

```typescript
// src/lib/nats.ts — nach Reconnect: Economy-Updates seit letztem empfangenen Seq nachholen
export async function replayEconomySince(starID: string, lastSeq: number) {
  const nc = getNATS()
  const js = nc.jetstream()

  const consumer = await js.consumers.get('ECONOMY', `web-${playerID}-economy`)
  // Falls Consumer noch nicht existiert, wird er angelegt:
  // deliver_policy: by_start_sequence, opt_start_seq: lastSeq

  const messages = await consumer.fetch({ max_messages: 100 })
  for await (const msg of messages) {
    processEconomyUpdate(jc.decode(msg.data))
    msg.ack()
  }
}
```

---

## 7. Reconnect und Fehlerbehandlung

### Browser-Reconnect-Strategie

```
1. nats.ws reconnect automatisch (exponential backoff, unbegrenzt)
2. Bei erneutem Connect:
   a. Token prüfen (TTL) — ggf. silent refresh via REST
   b. Subscriptions sind automatisch restored (nats.ws merkt sie sich)
   c. JetStream: lastSeq aus localStorage, replayEconomySince() aufrufen
3. Während Offline: UI zeigt "Verbindung unterbrochen" Banner
```

### Server-seitige Garantie

- Alle Tier-2-Events landen in JetStream-Streams
- Streams haben MaxAge (1–30 Tage) — Replay nach Reconnect immer möglich
- Tick-Engine publiziert auch wenn kein Client verbunden ist — keine verlorenen Events

---

## 8. devctl Integration

```go
// cmd/devctl/main.go — NATS als 4. Komponente
func makeNATS() *component {
    c := &component{id: "nats", display: "NATS", port: 4222}
    c.fnHealth = func() bool { return tcpAlive(4222) }
    c.fnStart = func(c *component) error {
        return runShell(c, "docker", "compose", "up", "-d", "nats")
    }
    c.fnStop = func(c *component) {
        _ = runShell(c, "docker", "compose", "stop", "nats")
    }
    if tcpAlive(4222) {
        c.st = stRunning; c.startedAt = time.Now()
    }
    return c
}
```

NATS Monitoring-Dashboard via Browser: `http://localhost:8222` (Dev).

---

## 9. Implementierungs-Reihenfolge (Worktree `feat/messaging`)

| Schritt | Was | Abhängigkeit |
|---|---|---|
| 1 | `internal/bus` Interface + Typen | — |
| 2 | `internal/bus/inprocbus` | Schritt 1 |
| 3 | Tests für inprocbus | Schritt 2 |
| 4 | `nats-server.conf` + docker-compose | — |
| 5 | `internal/bus/natsbus` Adapter | Schritt 1 + 4 |
| 6 | `cmd/server/main.go`: Bus instantiieren + EnsureStreams | Schritt 5 |
| 7 | Economy2-Tick-Handler: bus.PublishDurable nach Tick | Schritt 6 |
| 8 | `POST /api/v1/auth/nats-token` Endpoint | Schritt 5 |
| 9 | devctl: NATS-Komponente | Schritt 4 |
| 10 | Frontend: `nats.ws` installieren, `src/lib/nats.ts` | Schritt 8 |
| 11 | `useEconomyUpdates` Hook, Economy2Page auf NATS umstellen | Schritt 10 |
| 12 | HTTP-Polling in Economy2Page entfernen | Schritt 11 |

Schritte 1–3 und 4 können parallel laufen. Jeder Schritt ist unabhängig commitbar.
