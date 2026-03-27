# Progress – Galaxis v1.4

**Datum:** 2026-03-23

---

## Aktueller Status

**Phase:** AP4 weitgehend implementiert — Facility-Spezialisierung, 3-Ebenen-Lager, Fertigungsaufträge + Pool-Scheduler lauffähig · BOM-Resolver + Integrations-Tests konzipiert, noch nicht implementiert

---

## Erledigte Meilensteine

| Datum | Meilenstein |
|---|---|
| 2026-03-12 | GDD v1.24 finalisiert |
| 2026-03-12 | Stack & Architektur-Entscheidungen (ADR-001–ADR-007) |
| 2026-03-13 | AP3 Server-Core Skeleton, AP1 Galaxiengenerator |
| 2026-03-13 | God-Mode-Viewer lauffähig |
| 2026-03-14 | AP2 spezifiziert (Atmosphären, Biochemie, Ressourcen) |
| 2026-03-17 | BL-11: Image-Based Galaxy Generator |
| 2026-03-20 | BL-03: Galaxy-Scraper (75 Templates) |
| 2026-03-20 | AP2: Planetensystem-Generator lauffähig |
| 2026-03-20 | BL-12/15/18: Elliptische Orbits, Systembaum, Planetengrößen |
| 2026-03-20 | BL-12 Erw.: Mondorbits (Hill-Sphäre) |
| 2026-03-21 | BL-20/21/24: Prozedurale Shader, Asteroidengürtel, Mondsystem-Ansicht |
| 2026-03-21 | Vitest Test-Infrastruktur (25 Unit-Tests) |
| 2026-03-22 | AP4 Design: economy_v1.0.md, production-mechanics_v1.0.md, economy-mvp-architecture_v1.0.md finalisiert |
| 2026-03-22 | AP4 Backend (Schritt 9): Migration 006, Economy-Paket, 9 API-Routen, Tick-Engine |
| 2026-03-22 | AP4 Frontend (Schritt 10): EconomyPage vollständig (Deposits, Facilities, Storage, Log, BuildPanel) |
| **2026-03-23** | **AP4 Schritt 1: Facility-Typ-Spezialisierung** — jedes Gut hat eigenen Anlagentyp; recipes_v1.1.yaml (33 Rezepte, `output_good`-Feld, steel_chromium T2); game-params_v1.8.yaml (9 spezialisierte Typen); Migration 007 (smelter→steel_mill etc.); registry.go OutputGood-Map |
| **2026-03-23** | **AP4 Schritt 2: 3-Ebenen-Lager** — storage_nodes Tabelle (planetary/orbital/intersystem); Migration 008; storage.go vollständig neu (GetOrCreateNode, ProduceToNode, ConsumeFromNode, GetSystemNodes); production.go + handlers angepasst; Frontend: NodeHeader + Tier-gruppierte StorageTable |
| **2026-03-23** | **AP4 Schritt 3: Fertigungsaufträge + Pool-Scheduler** — Migration 009 (production_orders + current_order_id auf facilities); scheduler.go (LATERAL JOIN, HandleOrderBatchComplete); production.go (CurrentOrderID, unassignFacility, paused_input Auto-Retry); CRUD-Routen; Frontend: ProductionOrdersSection, OrderRow, NewOrderForm |
| **2026-03-23** | **AP4 Prioritätssystem** — Batch immer vor continuous, priority INT, Scheduler-Sortierung: mode ASC → priority DESC → created_at ASC |
| **2026-03-23** | **AP4 Bugfix:** paused_input-Anlagen mit Auftrag werden jeden Tick neu versucht (loadRunningFacilities erweitert) |

---

## Nächste Schritte (priorisiert)

| Priorität | Aufgabe |
|---|---|
| 🔥 Hoch | **Integrations-Tests** `go test -tags integration ./internal/economy/...` — Konzept fertig, Freigabe erteilt · Schema-basiert in gleicher DB, aufräumen nach Suite |
| 🔥 Hoch | **BOM-Resolver** `POST /economy/system/{starId}/production-plan` — Konzept fertig, Freigabe erteilt · DAG-Expansion, Demand-Aggregation, Kritischer Pfad, Mine-Ticks, Zyklenerkennung |
| Mittel | Stock-Sufficiency-Anzeige in FacilityCard (Traffic-Light: Inputfaktoren vs. Lager) |
| Mittel | AP4 Schritt 4: Pipeline-Entity (Transport zwischen Storage-Nodes) |
| Mittel | BL-16 Benennungssystem (vor AP3-Auth) |
| Niedrig | AP3 Remainder (Auth/JWT, WebSocket, Redis) |
| Niedrig | BL-13 Planetenrotation, BL-14 Mehrfachsternsysteme |

---

## DB-Migrationen (Übersicht)

| Nr | Datei | Inhalt | Status |
|---|---|---|---|
| 001 | `001_initial.up.sql` | Grundschema: galaxies, stars, nebulae, planets, moons, ftlw_cells | ✅ |
| 002 | `002_planet_model.up.sql` | Physikalisches Atmosphärenmodell | ✅ |
| 003 | `003_galaxy_status_steps.up.sql` | Galaxy-Status-Werte | ✅ |
| 004 | `004_orbital_mechanics.up.sql` | Kepler-Orbital-Parameter | ✅ |
| 005 | `005_moon_orbits.up.sql` | Mondorbit-Abstände (Hill-Sphäre) | ✅ |
| 006 | `006_economy.up.sql` | planet_deposits, facilities, system_storage, production_log, player_surveys | ✅ |
| 007 | `007_facility_type_rename.up.sql` | smelter→steel_mill, refinery→semiconductor_plant, bioreaktor→biosynth_lab | ✅ |
| 008 | `008_storage_nodes.up.sql` | storage_nodes Tabelle; system_storage→orbital migriert; storage_node_id auf facilities | ✅ |
| 009 | `009_production_orders.up.sql` | production_orders Tabelle (mode, batch, demand, priority); current_order_id auf facilities | ✅ |

---

## AP4 Economy-System — implementierte Dateien (aktuell)

| Datei | Inhalt |
|---|---|
| `recipes_v1.1.yaml` | 33 Rezepte, Feld `output_good`, steel_chromium T2 |
| `game-params_v1.8.yaml` | 9 spezialisierte Facility-Typen, facility_efficiency/build_ticks |
| `internal/economy/registry.go` | Recipe.OutputGood, FacilityRegistry.OutputGood Map |
| `internal/economy/storage.go` | Vollständig neu: node-basierte API (GetNodeStorage, SetNodeStorage, GetOrCreateNode, GetSystemNodes, ProduceToNode, ConsumeFromNode) |
| `internal/economy/production.go` | Facility.CurrentOrderID; StorageNodeID; paused_input Retry; unassignFacility |
| `internal/economy/scheduler.go` | SchedulerHandler, runScheduler (LATERAL JOIN), assignFacilityToOrder, HandleOrderBatchComplete |
| `internal/api/economy_handlers.go` | createOrder, updateOrder, cancelOrder, loadOrdersForSystem; orderDTO; storageNodeResponse |
| `frontend/src/api/economy.ts` | ProductionOrder, StorageNode, createOrder, cancelOrder, updateOrderPriority |
| `frontend/src/pages/EconomyPage.tsx` | ProductionOrdersSection, OrderRow, NewOrderForm; NodeHeader; Tier-gruppierte StorageTable |

---

## Konzepte genehmigt, noch nicht implementiert

### BOM-Resolver (Produktionsplan)
- **Endpoint:** `POST /economy/system/{starId}/production-plan { good_id, quantity }`
- **Algorithmus:** DAG-Traversal (Zyklenerkennung per DFS), Topologische Sortierung, Demand-Aggregation (Mehrfachverwendung eines Guts → summiert), Lagerbestand global abziehen, Kritischer Pfad berechnen
- **Mine-Ticks:** aus Deposit-Daten berechnet (Anlagenrate vs. Deposit.MaxRate)
- **Rezeptauswahl:** höchste verfügbare Tier, bei der alle Inputs erreichbar sind
- **Response:** feasible bool, total_ticks, missing_facilities[], raw_inputs map, plan[] (Tier-sortierte Tabelle)

### Integrations-Tests (`//go:build integration`)
- Schema-isoliert in gleicher DB (Schema `galaxis_test_<uuid>`), aufräumen nach Suite
- `go test -tags integration ./internal/economy/...`
- Geplante Tests: Mine produziert, Rezeptanlage pausiert/resumed, Scheduler weist zu, Batch-Countdown, Demand-Pausierung bei Target

---

## Offene Entscheidungen (TBD)

| Thema | Priorität |
|---|---|
| System-ID-Schema (BL-16) | Hoch (vor AP3-Auth) |
| Planeten-ID-Schema (BL-16) | Hoch (vor AP3-Auth) |
| Pipeline-Kapazität Schritt 4: explizite Zuweisung vs. automatisch | Mittel |
| Produktions-Cloud: AWS vs. GCP | Mittel (erst bei Deployment) |
| Frontend-Zustandsmanagement | Mittel (vor AP5/AP6) |
| Fiktive Tier-5-Ressourcen | Niedrig |
