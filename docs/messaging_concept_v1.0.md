# Galaxis Messaging — Konzept v1.0

**Datum:** 2026-03-28
**Status:** Entwurf — Basis für Broker-Evaluation und Worktree-Implementation
**Scope:** Intra-Server-Bus + Game-Client-Kommunikation (WebSocket)

---

## 1. Ziele

| Ziel | Beschreibung |
|---|---|
| Near-Realtime | Combat-Events, Schiffsbewegungen: <5ms end-to-end Server→Client |
| Zuverlässig | Economy-Ticks, Trade-Confirmations: garantierte Zustellung, kein Datenverlust |
| Broker-agnostisch | Wechsel MQTT↔NATS ohne Änderung an Services oder Frontend |
| Skalierbar | Einzelner Server heute, Cluster möglich ohne Architektur-Bruch |
| Lizenz-sauber | Alle Komponenten Apache-2.0 oder MIT ✅ |

---

## 2. Schichtenmodell

```
┌─────────────────────────────────────────────────────────────────┐
│  Game Client  (Browser)                                         │
│  spricht: Galaxis-WS-Protokoll (JSON)                           │
│  kennt: WEDER MQTT noch NATS                                    │
└──────────────────────┬──────────────────────────────────────────┘
                       │ WebSocket  /ws
┌──────────────────────▼──────────────────────────────────────────┐
│  WS Gateway  (Go, in galaxis-api)                               │
│  • Authentifizierung (PlayerID aus JWT/Session)                 │
│  • Übersetzt: WS-Action → bus.Request()                         │
│  • Übersetzt: bus.Subscribe() → WS-Event                        │
│  • Filtert: nur für diesen Spieler relevante Subjects           │
│  • Kennt: bus.Bus Interface, NICHT den konkreten Broker         │
└──────────────────────┬──────────────────────────────────────────┘
                       │ internal/bus.Bus
┌──────────────────────▼──────────────────────────────────────────┐
│  Broker Adapter  (austauschbar)                                 │
│  Implementierungen: NATSAdapter | MQTTAdapter | InProcAdapter   │
└──────────────────────┬──────────────────────────────────────────┘
                       │ Broker-Protokoll
┌──────────────────────▼──────────────────────────────────────────┐
│  Message Broker  (NATS / MQTT / embedded)                       │
└──────────────────────┬──────────────────────────────────────────┘
                       │ Subscriber
┌──────────────────────▼──────────────────────────────────────────┐
│  Game Services  (alle via bus.Bus)                              │
│  economy2, combat, ship, tick-engine, …                         │
└─────────────────────────────────────────────────────────────────┘
          │
          ▼
     PostgreSQL  (Source of Truth, persistenter Zustand)
```

**Invariante:** Kein Service und kein Frontend-Code referenziert je einen konkreten Broker. Nur die Adapter-Implementierungen kennen NATS/MQTT-spezifischen Code.

---

## 3. `internal/bus` Interface

### 3.1 Kern-Typen

```go
// Message ist die einheitliche Nachricht über alle Broker-Grenzen hinweg.
type Message struct {
    Subject string            // z.B. "galaxis.economy.{starID}.stock"
    Payload []byte            // JSON
    Headers map[string]string // optional: correlation_id, player_id, …
}

// MsgHandler wird für jeden eingehenden Message aufgerufen.
type MsgHandler func(msg Message)

// AckHandler wird für durable Messages aufgerufen; Ack muss explizit bestätigt werden.
type AckHandler func(msg Message, ack func() error)

// Subscription repräsentiert ein aktives Abonnement.
type Subscription interface {
    Unsubscribe() error
}

// StreamConfig beschreibt einen persistenten Kanal (JetStream-Stream / MQTT-Retained).
type StreamConfig struct {
    Name     string   // z.B. "ECONOMY"
    Subjects []string // Subjects die in diesen Stream fallen
    MaxAge   time.Duration
    MaxBytes int64
}
```

### 3.2 Bus Interface

```go
// Bus ist die einzige öffentliche Abstraktion für alle Messaging-Operationen.
type Bus interface {
    // --- Tier 1: At-most-once (fire-and-forget) ---
    // Für: Combat-Events, Schiffsbewegung, Live-Positions-Updates
    Publish(ctx context.Context, msg Message) error

    // Subscribe empfängt alle Nachrichten auf einem Subject.
    // subject darf Wildcards enthalten (* = ein Segment, > = alle folgenden).
    Subscribe(ctx context.Context, subject string, h MsgHandler) (Subscription, error)

    // QueueSubscribe: Nur ein Subscriber der Gruppe empfängt die Nachricht (Load-Balancing).
    // Für: horizontales Scaling von Game-Service-Instanzen.
    QueueSubscribe(ctx context.Context, subject, queue string, h MsgHandler) (Subscription, error)

    // Request sendet eine Nachricht und wartet auf eine Antwort (Request/Reply).
    // Für: Player-Actions mit sofortiger Rückmeldung.
    Request(ctx context.Context, msg Message, timeout time.Duration) (Message, error)

    // Reply antwortet auf ein Request. Wird nur in Handlern aufgerufen die über Request eingehen.
    Reply(ctx context.Context, to Message, reply Message) error

    // --- Tier 2: At-least-once (durable, persistent) ---
    // Für: Economy-Ticks, Trade-Confirmationen, kritische Spielereignisse

    // EnsureStream legt einen persistenten Stream an (idempotent).
    EnsureStream(ctx context.Context, cfg StreamConfig) error

    // PublishDurable veröffentlicht mit Persistenz-Garantie.
    PublishDurable(ctx context.Context, stream string, msg Message) error

    // SubscribeDurable abonniert dauerhaft; Nachrichten werden erst nach Ack() entfernt.
    // startSeq=0: ab nächster Nachricht; startSeq>0: Replay ab dieser Sequenz.
    SubscribeDurable(
        ctx context.Context,
        stream, consumer string,
        startSeq uint64,
        h AckHandler,
    ) (Subscription, error)

    // --- Lifecycle ---
    Close() error
}
```

### 3.3 Konstruktor-Konvention

```go
// Jede Adapter-Implementierung liegt in einem Sub-Package:
// internal/bus/natsbus   → func New(url string) (bus.Bus, error)
// internal/bus/mqttbus   → func New(url string, opts MQTTOptions) (bus.Bus, error)
// internal/bus/inprocbus → func New() bus.Bus  (Tests, devctl-Modus)
```

In `cmd/server/main.go` wird exakt eine Implementierung instantiiert und als `bus.Bus` an alle Services übergeben.

---

## 4. Subject-Schema

Format: `galaxis.<domain>.<scope>.<event>`

Wildcards: `*` = ein Segment, `>` = alle folgenden Segmente

```
galaxis.tick.advance                       Tier 2  — Tick-Nr. + Zeitstempel nach jedem Tick
galaxis.tick.result.<starID>               Tier 2  — Economy-Ergebnis nach Tick (Stocks, Orders)

galaxis.economy.<starID>.stock             Tier 2  — Lageränderung
galaxis.economy.<starID>.order.<orderID>   Tier 2  — Order-Status-Update
galaxis.economy.<starID>.facility.*        Tier 1  — Facility-Status-Live-Update

galaxis.combat.<systemID>.event            Tier 1  — Combat-Event (Schuss, Treffer, Tod)
galaxis.combat.<systemID>.state            Tier 2  — Combat-State-Snapshot (nach Runde)

galaxis.ship.<shipID>.move                 Tier 1  — Schiffbewegung (Position, Kurs)
galaxis.ship.<shipID>.status               Tier 2  — Schiff-Statusänderung (HP, Treibstoff)

galaxis.player.<playerID>.notify           Tier 2  — Spieler-spezifische Benachrichtigung
galaxis.player.<playerID>.session          Tier 1  — Online/Offline-Status

galaxis.action.<type>                      —       — Player-Actions via Request/Reply
                                                     z.B. galaxis.action.ship.move
                                                          galaxis.action.economy.create_order
```

**Stream-Definitionen (Tier 2):**

| Stream | Subjects | MaxAge | Zweck |
|---|---|---|---|
| `TICK` | `galaxis.tick.>` | 7 Tage | Tick-History, Replay |
| `ECONOMY` | `galaxis.economy.>` | 7 Tage | Economy-State-Trail |
| `COMBAT` | `galaxis.combat.*.state` | 1 Tag | Combat-State-Snapshots |
| `PLAYER` | `galaxis.player.>` | 30 Tage | Player-Notifications (auch offline) |

Tier-1-Subjects (`combat.event`, `ship.move`, `facility.*`) landen in keinem Stream — sie sind flüchtig.

---

## 5. WS Gateway

### 5.1 Verbindungsaufbau

```
GET /ws?token=<JWT>   HTTP 101 Upgrade
```

Nach Upgrade:
1. Gateway liest `playerID` aus JWT
2. Abonniert automatisch: `galaxis.player.<playerID}.notify` (durable)
3. Abonniert automatisch: `galaxis.tick.advance`
4. Alle weiteren Subscriptions kommen vom Client via `subscribe`-Message

### 5.2 Galaxis-WS-Protokoll (Client ↔ Gateway)

Alle Messages sind JSON. Richtung: `S→C` = Server zu Client, `C→S` = Client zu Server.

#### Event  `S→C`
```json
{
  "type":    "event",
  "subject": "galaxis.economy.{starID}.stock",
  "seq":     1042,
  "payload": { ... }
}
```

#### Subscribe  `C→S`
```json
{
  "type":    "subscribe",
  "subject": "galaxis.economy.{starID}.>",
  "id":      "sub_1"
}
```

#### Unsubscribe  `C→S`
```json
{ "type": "unsubscribe", "id": "sub_1" }
```

#### Action (Request/Reply)  `C→S`
```json
{
  "type":   "action",
  "action": "ship.move",
  "req_id": "abc123",
  "payload": { "ship_id": "...", "target_system": "..." }
}
```

#### Reply  `S→C`
```json
{
  "type":    "reply",
  "req_id":  "abc123",
  "ok":      true,
  "payload": { "eta_ticks": 4 }
}
```

#### Error  `S→C`
```json
{
  "type":    "error",
  "req_id":  "abc123",
  "code":    "INSUFFICIENT_FUEL",
  "message": "Treibstoff reicht nicht für diese Route"
}
```

#### Ping/Pong  `C↔S`
Standard WebSocket Ping/Pong (RFC 6455). Timeout: 30s ohne Pong → Disconnect.

### 5.3 Sicherheit im Gateway

- Jede eingehende `subscribe`-Message wird gegen eine Whitelist valider Subject-Patterns geprüft
- Ein Spieler darf nur Subjects abonnieren die seinen `playerID` enthalten oder öffentlich sind (z.B. `galaxis.combat.<systemID>.*`)
- Actions werden im Gateway nicht ausgeführt — nur weitergeleitet; Autorisierung liegt im jeweiligen Service-Handler
- Rate-Limit: max 100 Messages/s pro Verbindung

---

## 6. Delivery-Garantien im Überblick

```
Player schießt auf Schiff
  → galaxis.combat.<systemID>.event   Tier 1 (fire-and-forget)
  → galaxis.combat.<systemID>.state   Tier 2 (nach Kampfrunde, persistiert)

Spieler legt Order an
  → galaxis.action.economy.create_order   Request/Reply (sofortiges OK/Fehler)
  → galaxis.economy.<starID>.order.*      Tier 2 (Order-Update an alle Subscriber)

Tick feuert
  → galaxis.tick.advance              Tier 2
  → galaxis.tick.result.<starID>      Tier 2
  → galaxis.economy.<starID>.stock    Tier 2

Spieler offline, kommt zurück
  → SubscribeDurable auf galaxis.player.<playerID>.notify
  → Replay aller verpassten Notifications seit letztem Ack
```

---

## 7. InProc-Adapter (Test + Dev)

Für Unit-Tests und devctl-Betrieb ohne Broker existiert ein `inprocbus`-Adapter der das gesamte Interface in-memory implementiert. Er unterstützt:
- Publish/Subscribe (Fan-out via Go-Channels)
- QueueSubscribe (Round-Robin)
- Request/Reply (synchron, timeout-fähig)
- Durable-Simulation (In-Memory-Queue mit Ack)

Kein externer Prozess nötig. Integration-Tests laufen damit ohne Docker.

---

## 8. Offene Fragen für Broker-Evaluation

Die folgenden Punkte sind von der Interface-Definition noch offen und werden pro Broker beantwortet:

| # | Frage |
|---|---|
| B1 | Wie wird `QueueSubscribe` für Load-Balancing implementiert? |
| B2 | Wie werden Streams/Durable-Consumer konfiguriert? |
| B3 | Wie verhält sich der Adapter bei Broker-Disconnect? (Reconnect-Logik) |
| B4 | Wie wird die Subject-Wildcard-Syntax intern gemappt? |
| B5 | Wie groß ist der Payload-Overhead pro Message? |
| B6 | Wie wird der Broker in docker-compose + devctl integriert? |
| B7 | Welche Lizenz hat der Go-Client (nicht nur der Broker)? |
| B8 | Kann der InProc-Adapter für Tests vollständig eingesetzt werden (kein Broker nötig)? |

---

## 9. Nächste Schritte

1. **Broker-Evaluation** — NATS, MQTT (EMQX), mochi-mqtt (embedded) anhand der Fragen B1–B8
2. **ADR schreiben** — Broker-Entscheidung dokumentieren
3. **Worktree anlegen** — `feat/messaging` Branch
4. **Implementation** in dieser Reihenfolge:
   - `internal/bus` Interface + Typen
   - `internal/bus/inprocbus` (Tests lauffähig ohne Broker)
   - WS Gateway Grundgerüst
   - Broker-Adapter (gewählter Broker)
   - Economy2-Tick-Handler auf Bus umstellen
   - Frontend WS-Client (ersetzt HTTP-Polling)
