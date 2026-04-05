# Architektur – Galaxis v1.2

**Datum:** 2026-04-05
**GDD-Referenz:** v1.24
**Änderungen gegenüber v1.1:** Economy2 v2 — 5-Stufen-Fabrik-Taxonomie, Factories-as-Items, Deploy-Mechanismus, Deposit-Modell v2

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
│  • Tick-Engine (Scheduler + Production + Ship)                     │
│  • Economy2 Services (MRP, Orders, Facilities, Stock, Deploy)      │
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

**Invariante:** Browser und Go-Services sprechen dieselbe NATS-Infrastruktur. Kein custom WS-Gateway-Code — NATS-over-WS ist nativ vom `nats-server` unterstützt.

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
| Economy2 | `internal/economy2` | MRP, Orders, Facilities, Stock, Extractor, Ship |
| Planet-Generator | `internal/planet` | Prozedurale Planetensysteme + Deposit-Generierung |
| Galaxy-Generator | `internal/generator` | Sterne, Nebel, FTLW-Grid |
| Bus | `internal/bus` | Broker-agnostisches Interface (→ NATS-Adapter) |
| Job-Store | `internal/jobs` | Async-Jobs für Galaxie-Generierung |

**Tick-Handler-Reihenfolge pro Tick:**
1. `economy2.SchedulerHandler` — MRP-Allokation + Zuweisung idle→running
2. `economy2.ProductionHandler` — Extraktion, Fertigung, Effizienz-Akkumulator
3. `economy2.ShipTickHandler` — Schiffsbewegung (Grundgerüst)

Nach jedem Tick publiziert die Engine `galaxis.tick.advance` (JetStream).

---

## Economy2-System (v2)

Vollständig implementiertes Wirtschaftssystem (Migration 010–014).

Konzeptdokument: `docs/economy_v2.0.md`

### 5-Stufen-Fabrik-Taxonomie

| Stufe | factory_type | Produziert |
|---|---|---|
| 1 | `extractor` | Rohstoffe aus Deposits (keine Input-Güter) |
| 2 | `refinery` | Verarbeitete Materialien (Stahl, Halbleiter, Fusionskraftstoff) |
| 3 | `plant` | Komponenten + Tier-1/2-Fabrik-Items |
| 4 | `assembly_plant` | Schwerbauteile (drive_unit, reactor_module, structural_frame) + Tier-3-Items |
| 5 | `construction_yard` | Schiffe + Tier-4/5-Fabrik-Items |

### Factories-as-Items

Fabriken existieren zunächst als **Inventar-Items** (z.B. `fac_extractor_iron_mk1`) und werden durch eine explizite Deploy-Aktion zu aktiven Anlagen. Das ermöglicht Transport und Handel von Fabriken als Waren.

**Item-Katalog:** `items_v1.0.json`
- Schlüssel: `item_id` (z.B. `fac_extractor_iron_mk1`)
- Wert: `DeployableItemDef` mit `factory_type`, `deposit_good_id`, `level`, `max_rate`
- Kein String-Parsing — reines Lookup

**Deploy-Flow:**
```
econ2_item_stock  (fac_extractor_iron_mk1: 1)
      ↓ POST /api/v2/econ2/items/deploy  {node_id, item_id}
      ↓ DeployItem(): konsumiert 1 Item, erstellt Facility
econ2_facilities  (factory_type=extractor, config.deposit_good_id=iron)
```

### Produktionspfad

```
planets.resource_deposits (JSONB)
      ↓ extractor (processExtractor)
econ2_item_stock  [Rohstoffe]
      ↓ refinery
econ2_item_stock  [Materialien: steel, semiconductor_wafer, fusion_fuel, …]
      ↓ plant
econ2_item_stock  [Komponenten: base_component, nav_computer, fac_*_mk1]
      ↓ assembly_plant
econ2_item_stock  [Schwerbauteile: drive_unit, reactor_module, structural_frame]
      ↓ construction_yard
econ2_item_stock  [Schiffe, fac_construction_yard_mk1, fac_assembly_plant_mk1]
      ↓ Transport-Route (econ2_routes)
econ2_item_stock  [Zielknoten]
```

### Deposit-Modell (Migration 014)

Deposits leben als JSONB direkt auf `planets.resource_deposits`:

```json
{
  "iron":   { "amount": 48500.0, "quality": 0.97, "max_mines": 5 },
  "helium_3": { "amount": 12000.0, "quality": 0.61, "max_mines": 14 }
}
```

- `max_mines` = maximale simultane Extractor-Anlagen auf diesem Deposit
- Gas-Riesen: Ø 8 Slots (orbital harvesting, bis 20)
- Asteroidengürtel: Ø 6 Slots (verteilte Körper, bis 15)
- Felsplaneten / Eisriesen: Ø 4 Slots (Oberflächenabbau, bis 10)

### Datenmodell

| Tabelle | Inhalt |
|---|---|
| `econ2_nodes` | Wirtschaftsknoten (Orbit, Planet, Mond) |
| `econ2_item_stock` | Lagerbestand pro Node + Good |
| `econ2_facilities` | Anlagen, Status, Config (JSONB mit level, max_rate, deposit_good_id, …) |
| `econ2_orders` | Produktionsaufträge (batch, continuous) |
| `econ2_routes` | Transport-Routen zwischen Nodes |
| `econ2_ships` | Schiffe (Grundgerüst) |

`econ2_orders.order_type` kennt nur noch `batch` und `continuous` — kein `build` mehr.

### Rezepte

Datei: `econ2_recipes_v2.0.yaml`

Jedes Rezept hat `recipe_id`, `product_id`, `factory_type`, `inputs[]`, `base_yield`, `ticks`, `efficiency`. Extractor-Rezepte haben zusätzlich `geological_input` und leere `inputs`.

Rezeptindex wird beim Server-Start in-memory geladen (`RecipeBook` = `map[RecipeKey]*Recipe`).

### API-Endpunkte (Economy2, /api/v2)

| Methode | Pfad | Beschreibung |
|---|---|---|
| `POST` | `/econ2/items/deploy` | Item aus Lager deployen → Facility erstellen |
| `GET` | `/econ2/facilities` | Alle Facilities des Spielers |
| `DELETE` | `/econ2/facilities/{id}` | Facility zerstören |
| `POST` | `/econ2/orders` | Produktionsauftrag anlegen |
| `GET` | `/econ2/orders` | Aufträge eines Nodes |
| `DELETE` | `/econ2/orders/{id}` | Auftrag stornieren |
| `POST` | `/econ2/routes` | Transport-Route anlegen |
| `GET` | `/econ2/routes` | Routen des Spielers |
| `GET` | `/econ2/stock` | Lagerbestand eines Nodes |
| `POST` | `/econ2/nodes` | Node anlegen/abrufen |
| `GET` | `/econ2/my-nodes` | Alle Nodes des Spielers |
| `POST` | `/econ2/bootstrap` | Startpaket für neues Sternsystem |
| `GET` | `/econ2/recipes` | Alle Rezepte |
| `GET` | `/econ2/deposits` | Deposit-Zustand des Heimatplaneten |

---

## Schlüsselprinzipien

### Planetengenerierung – Zwei-Modus-Strategie (ADR-009)
Unverändert gegenüber v1.0. Eager-Modus für Balancing, JIT für Produktion.

### Autoritativer Server
Clients senden nur Intent (Befehle/Aktionen), nie Zustand. Alle Berechnungen laufen auf dem Server.

### JSON over Columns (Datenbankstrategie)
Konfigurationen und sich häufig ändernde Daten in JSONB, relationale Spalten nur für FKs und WHERE-Felder. Neue Güter und Rezepte → YAML/JSON-Änderung, keine Migration.

### Broker-Abstraktion
Kein Service-Code kennt NATS direkt. Alle nutzen `bus.Bus`. Details: `docs/messaging_concept_v1.0.md`.

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
| `docs/economy_v2.0.md` | Wirtschaftsmodell v2 — Produktionskette, Deploy-Mechanismus, Tier-4/5 |
| `docs/messaging_concept_v1.0.md` | Bus-Interface-Design, Subject-Schema, Delivery-Tiers |
| `docs/messaging_nats_v1.0.md` | NATS-Implementierung, Browser-Client, Auth-Flow |
| `tech-decisions_v1.0.md` | ADRs 001–009 |
| `progress_v1.5.md` | Sprint-Log, offene Punkte |
