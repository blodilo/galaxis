# Architektur – Galaxis v1.1

**Datum:** 2026-03-28
**GDD-Referenz:** v1.24
**Änderungen gegenüber v1.0:** Messaging-Architektur (NATS ersetzt Redis Pub/Sub), WebSocket-Strategie, Economy2 eingearbeitet

---

## Überblick

Galaxis ist ein Hard-Sci-Fi Grand Strategy MMO mit strikter Client-Server-Trennung. Der Server ist die alleinige Autorität über den Spielzustand. Clients kommunizieren über REST (Initialdaten) und WebSocket/NATS (Echtzeit-Events und Aktionen).

```
┌────────────────────────────────────────────────────────────────────┐
│  CLIENTS                                                           │
│  ┌───────────────────────┐   ┌──────────────────────────────────┐ │
│  │  Web-Frontend         │   │  KI-Headless-Client (Go)         │ │
│  │  React + Vite + TS    │   │  Identische API + Bus-Nutzung    │ │
│  │  nats.ws (Apache-2.0) │   │  nats.go (Apache-2.0)            │ │
│  └──────────┬────────────┘   └─────────────────┬────────────────┘ │
└─────────────┼───────────────────────────────────┼──────────────────┘
              │ REST (Initialdaten)                │ NATS / REST
              │ NATS-over-WS (Events, Aktionen)   │
┌─────────────▼───────────────────────────────────▼──────────────────┐
│  galaxis-api  (Go, chi)                                            │
│  • REST-Routen: Galaxy, Economy2, Admin                            │
│  • NATS Credential Endpoint: POST /api/v1/auth/nats-token          │
│  • Tick-Engine (Scheduler + Production + Build + Ship)             │
│  • Economy2 Services (MRP, Orders, Facilities, Stock)              │
└─────────────────────────────┬──────────────────────────────────────┘
                              │ nats.go  TCP :4222
┌─────────────────────────────▼──────────────────────────────────────┐
│  NATS Server  (Apache-2.0)                                         │
│  TCP  :4222  ← Go-Services                                        │
│  WS   :4223  ← Browser-Clients  (TLS in Produktion)               │
│  JetStream aktiviert                                               │
│                                                                    │
│  Subject-Space:                                                    │
│  galaxis.tick.*          galaxis.economy.<starID>.*               │
│  galaxis.combat.*        galaxis.ship.*                            │
│  galaxis.player.<id>.*   galaxis.action.*                         │
└──────────────────────────────────────────────────────────────────┬─┘
                                                                   │
┌──────────────────────────────────────────────────────────────────▼─┐
│  PostgreSQL 16                                                     │
│  Source of Truth für persistenten Spielzustand                    │
│  Galaxie, Sterne, Planeten, Economy2, Deposits, Facilities         │
└─────────────────────────────────────────────────────────────────────┘
```

**Invariante:** Browser und Go-Services sprechen dieselbe NATS-Infrastruktur. Kein custom WS-Gateway-Code — NATS-over-WS ist nativ vom `nats-server` unterstützt. Der Browser erhält nach Login ein scoped NATS-Credential das ihn auf seine eigenen Subjects beschränkt.

Detail: `docs/messaging_nats_v1.0.md`

---

## Komponenten

### galaxis-api (Go)

Monolithischer Prozess in der Entwicklungsphase, modular aufgebaut für spätere Aufteilung.

**Subsysteme:**

| Subsystem | Paket | Beschreibung |
|---|---|---|
| REST-Router | `internal/api` | chi v5, alle HTTP-Endpunkte |
| Tick-Engine | `internal/tick` | Strategietick, manuell auslösbar (Admin) und periodisch |
| Economy2 | `internal/economy2` | MRP, Orders, Facilities, Stock, Mine, Build, Ship |
| Planet-Generator | `internal/planet` | Prozedurale Planetensysteme |
| Galaxy-Generator | `internal/generator` | Sterne, Nebel, FTLW-Grid |
| Bus | `internal/bus` | Broker-agnostisches Interface (→ NATS-Adapter) |
| Job-Store | `internal/jobs` | Async-Jobs für Galaxie-Generierung |

**Tick-Handler-Reihenfolge pro Tick:**
1. `economy2.SchedulerHandler` — MRP-Allokation + Zuweisung idle→running
2. `economy2.BuildTickHandler` — Bau-Aufträge ticken
3. `economy2.ProductionHandler` — Mine-Abbau, Fertigung, Effizienz
4. `economy2.ShipTickHandler` — Schiffsbewegung (Grundgerüst)

Nach jedem Tick publiziert die Engine `galaxis.tick.advance` (JetStream).

### NATS

Zentrale Messaging-Infrastruktur. Kein Redis mehr.

- **Core (at-most-once):** Combat-Events, Live-Positions-Updates
- **JetStream (at-least-once):** Tick-Events, Economy-Updates, Player-Notifications
- **Request/Reply:** Player-Aktionen mit sofortiger Serverantwort

Streams: `TICK`, `ECONOMY`, `COMBAT`, `PLAYER` — Details in `docs/messaging_nats_v1.0.md`.

### PostgreSQL

Unveränderter Source of Truth. NATS ist ausschließlich für Events, nie für persistenten Zustand. Nach einem NATS-Restart sind alle Events weg — der Spielzustand liegt in Postgres.

### internal/bus Interface

Broker-agnostisches Interface das alle Services gegen Programming. Einzige Implementierung: `internal/bus/natsbus`. Für Tests: `internal/bus/inprocbus` (kein externer Prozess).

Details: `docs/messaging_concept_v1.0.md`

---

## Economy2-System

Vollständig implementiertes Wirtschaftssystem (Migration 010–013).

### Produktionsfluss

```
planet_deposits (JSONB)
      ↓ Mine-Facility (processMine)
econ2_item_stock
      ↓ Smelter / Refinery / Precision-Fab
econ2_item_stock (Fertigwaren)
      ↓ Transport-Route (econ2_routes)
econ2_item_stock (Zielknoten)
```

### Datenmodell

| Tabelle | Inhalt |
|---|---|
| `econ2_nodes` | Wirtschaftsknoten (Orbit, Planet, Mond) |
| `econ2_item_stock` | Lagerbestand pro Node + Good |
| `econ2_facilities` | Anlagen (Mine, Smelter, …), Status, Config |
| `econ2_orders` | Produktionsaufträge (batch, continuous, build) |
| `econ2_routes` | Transport-Routen zwischen Nodes |
| `econ2_ships` | Schiffe (Grundgerüst) |
| `planet_deposits` | Rohstoffvorkommen pro Planet (JSONB) |

### Ressourcen-IDs

IDs entsprechen exakt `internal/planet/resources.go` (`iron`, `silicon`, `titanium`, `rare_earth`, `helium_3`, `hydrogen`, …). Migration 013 hat historische Abweichungen (iron_ore etc.) bereinigt.

---

## Schlüsselprinzipien

### Planetengenerierung – Zwei-Modus-Strategie (ADR-009)
Unverändert gegenüber v1.0. Eager-Modus für Balancing, JIT für Produktion.

### Autoritativer Server
Clients senden nur Intent (Befehle/Aktionen), nie Zustand. Alle Berechnungen laufen auf dem Server. KI-Headless-Clients nutzen dieselbe API und denselben Bus.

### JSON over Columns (Datenbankstrategie)
Konfigurationen und sich häufig ändernde Daten in JSONB, relationale Spalten nur für FKs und WHERE-Felder. Neue Güter und Rezepte → YAML-Änderung, keine Migration.

### Broker-Abstraktion
Kein Service-Code kennt NATS direkt. Alle nutzen `bus.Bus`. Adapter-Wechsel kostet 1–3 Tage, kein Service-Code ändert sich. Dokumentiert: `docs/messaging_concept_v1.0.md`.

---

## Deployment

| Phase | Infrastruktur |
|---|---|
| Entwicklung | galaxis-devctl (Port 9191), Docker Compose (Postgres + NATS) |
| Produktion | AWS/GCP, containerisiert, TLS auf NATS-WS-Port |
| Combat Pods | Kubernetes on-demand (geplant, AP6) |

---

## Frontend

- **Stack:** React 19, Vite 5, TypeScript, Tailwind CSS, Three.js + R3F
- **Datenstrategie:** REST für Initialdaten beim Laden, NATS-WS für Live-Updates
- **NATS-Client:** `nats.ws` (npm, Apache-2.0 ✅)
- **Ansichten:** Galaktisches Holo-Deck → Sternensystem → Planet → Economy2Page → (CIC geplant)

---

## Referenzen

| Dokument | Inhalt |
|---|---|
| `docs/messaging_concept_v1.0.md` | Bus-Interface-Design, Subject-Schema, Delivery-Tiers |
| `docs/messaging_nats_v1.0.md` | NATS-Implementierung, Browser-Client, Auth-Flow |
| `tech-decisions_v1.0.md` | ADRs 001–009 |
| `progress_v1.5.md` | Sprint-Log, offene Punkte |
| `economy_v1.0.md` | Wirtschaftsmodell-Design |
