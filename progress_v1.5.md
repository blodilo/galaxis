# Progress – Galaxis v1.5

**Datum:** 2026-03-27

---

## Aktueller Status

**Phase:** Economy2-System vollständig lauffähig — Rezept-getriebene UI, Bau-als-Auftrag, Tick-Generator · Nächster Schritt: Integrations-Tests + Balancing

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
| 2026-03-22 | AP4 Backend (altes System): Migration 006, Economy-Paket, 9 API-Routen, Tick-Engine |
| 2026-03-22 | AP4 Frontend (altes System): EconomyPage vollständig |
| 2026-03-23 | AP4 Schritt 1–3 (altes System): Facility-Spezialisierung, 3-Ebenen-Lager, Fertigungsaufträge |
| **2026-03-27** | **Economy2-System: Rebase auf feat/image-generator abgeschlossen** — Migrations 001–010, WriteTimeout-Fix (SSE/Planetengenerator), Vite-Proxy :8080 / Port 5175 |
| **2026-03-27** | **Economy2: JSON-Tags** auf allen Go-Structs (ItemStock, Facility, ProductionOrder, Route) — API liefert snake_case |
| **2026-03-27** | **Economy2: Meine Assets** — `GET /econ2/my-nodes` (JOIN stars + facility count); MyAssetsView als Landing Page; PlanetInspector ruft Bootstrap beim Heimatplaneten anlegen auf |
| **2026-03-27** | **Economy2: Bau als Auftrag** — 9 Construction-Rezepte in econ2_recipes_v1.0.yaml; `BuildTickHandler`; `GET /econ2/recipes`; `OrderTypeBuild`; `parseBuildProductID` |
| **2026-03-27** | **Economy2: Rezept-getriebene UI** — AnlagenPanel: Bau-Rezept-Dropdown → erstellt Build-Order, zeigt Fortschrittsbalken; AuftraegePanel: Produktionsrezept-Dropdown, filtert Bau-Aufträge heraus; menschenlesbare Labels |
| **2026-03-27** | **Tick-Generator in Menüleiste** — `POST /admin/tick/advance` (gibt Tick-Nr. zurück), `GET /admin/tick/current`; TickGenerator-Widget (▶/⏹, ×10/÷10, 0.1–100 ticks/s, Tick-Anzeige) |
| **2026-03-27** | **galaxis-devctl** — Standalone Go Prozessmanager auf `:9191`; Start/Stop/Restart für postgres, galaxis-api, galaxis-frontend; SSE Log-Streaming; Echtzeit-Status (Port, PID, Uptime); erkennt bereits laufende Komponenten beim Start |
| **2026-03-27** | **Facility-Location-Refactor** — Migration 011: `planet_id` aus `econ2_facilities` entfernt, `moon_id` auf `econ2_nodes` ergänzt; Facility-Standort ausschließlich über Node; Bootstrap legt planet-level Node an; alle DB-Queries JOINen mit `econ2_nodes` für planet_id |

---

## Meilensteine 2026-04-06 — Economy2 UI Neuaufbau

| Datum | Meilenstein |
|---|---|
| **2026-04-06** | **Migration 017: econ2_goals** — Tabelle `econ2_goals` (player_id, star_id, product_id, target_qty, priority, transport_overrides JSONB); `goal_id` FK auf `econ2_orders` |
| **2026-04-06** | **7 neue Backend-Endpunkte** — goals CRUD (POST/GET/DELETE/PATCH reorder), stock-all, facilities-all, orders-all; `walkRecipeTree` in mrp.go für BOM-basierte Order-Erstellung |
| **2026-04-06** | **Facility Start/Stop** — `POST /facilities/{id}/start` (Extractor: auto-continuous-order; Andere: MRP-Allokation + Zuweisung), `POST /facilities/{id}/stop` (nur DIESE Facility) |
| **2026-04-06** | **Scheduler Fix** — Extractor-Zuweisung nach `deposit_good_id` (nicht nur `factory_type`); Order-Suche nach `star_id` statt `node_id` |
| **2026-04-06** | **Economy2Page komplett neu** — Shell mit 3 Tabs (PLAN/FABRIKEN/NETZWERK) + LeftRail (Drag-to-reorder Goals, Alerts, Lager-Summary) |
| **2026-04-06** | **PlanTab** — GoalPicker + rekursiver BOMTree (7 Status-Zustände: ok/running/waiting/no_factory/route_missing/in_transit/transport_override) + inline FixPanels |
| **2026-04-06** | **FabrikenTab** — Anlagen gruppiert nach Stern, Spaltenheader, Start/Stop pro Facility; Cytoscape-Produktionsgraph (dagre-Layout, Güter=Rechtecke, Anlagen=Hexagons) |
| **2026-04-06** | **NetzwerkTab** — Node-Karten + Route-Schematik + Route-Erstellung per Klick |
| **2026-04-06** | **Bootstrap-Fix** — game-params v1.3/v1.8/v1.9: `mine`→`extractor`, `smelter`→`refinery`; vollständige Kette: Extractor+Refinery+Plant+AssemblyPlant+ConstructionYard; Stock um Titansteel, SemiconductorWafer, StructuralFrame, ReactorModule, DriveUnit erweitert |
| **2026-04-06** | **Altlasten bereinigt** — alte mine/smelter Facilities gelöscht, Zerstören-Button entfernt |

## Nächste Schritte (priorisiert)

| Priorität | Aufgabe |
|---|---|
| 🔥 Hoch | **BOM kumulativer Bedarf** — Gesamtbedarf über alle BOM-Knoten aggregieren statt einzeln gegen Stock prüfen |
| 🔥 Hoch | **Transport-Override persistieren** — PATCH /goals/{id}/transport-overrides Endpunkt + Frontend-Integration |
| Mittel | **Produktionsgraph verbessern** — Input-Kanten für non-Extractor-Anlagen; per-Node-Stock im Graph; interaktive Knoten |
| Mittel | **LeftRail Alerts** — BOM-Status-basierte Bottleneck-Erkennung statt nur Low-Stock |
| Mittel | **Integrations-Tests Economy2** — `go test ./internal/economy2/...` mit echter DB |
| Mittel | AP4 Pipeline (Schritt 4): Transport zwischen Nodes als echte Entität |
| Niedrig | BL-16 Benennungssystem |
| Niedrig | AP3 Remainder (Auth/JWT, WebSocket, Redis) |

---

## DB-Migrationen (Übersicht)

| Nr | Datei | Inhalt | Status |
|---|---|---|---|
| 001 | `001_initial.up.sql` | Grundschema: galaxies, stars, nebulae, planets, moons, ftlw_cells | ✅ |
| 002 | `002_planet_model.up.sql` | Physikalisches Atmosphärenmodell | ✅ |
| 003 | `003_galaxy_status_steps.up.sql` | Galaxy-Status-Werte | ✅ |
| 004 | `004_orbital_mechanics.up.sql` | Kepler-Orbital-Parameter | ✅ |
| 005 | `005_moon_orbits.up.sql` | Mondorbit-Abstände (Hill-Sphäre) | ✅ |
| 006 | `006_economy.up.sql` | planet_deposits, facilities, system_storage, production_log, player_surveys (altes System) | ✅ |
| 007 | `007_facility_type_rename.up.sql` | smelter→steel_mill etc. (altes System) | ✅ |
| 008 | `008_storage_nodes.up.sql` | storage_nodes (altes System) | ✅ |
| 009 | `009_production_orders.up.sql` | production_orders (altes System) | ✅ |
| 010 | `010_economy2.up.sql` | econ2_nodes, econ2_item_stock, econ2_facilities, econ2_orders, econ2_routes, econ2_ships, econ2_warnings | ✅ |
| 011 | `011_econ2_facility_location.up.sql` | planet_id aus facilities entfernt, moon_id auf nodes | ✅ |
| 012 | `012_cascade_planet_deposits.up.sql` | FK cascades | ✅ |
| 013 | `013_align_resource_ids.up.sql` | Resource IDs auf economy2-Konvention | ✅ |
| 014 | `014_deposit_model_v2.up.sql` | planet_deposits → planets.resource_deposits JSONB | ✅ |
| 015 | `015_deposit_rename_amount_to_remaining.up.sql` | amount→remaining in JSONB | ✅ |
| 016 | `016_econ2_align_factory_types.up.sql` | mine→extractor, smelter→refinery + resource keys | ✅ |
| 017 | `017_econ2_goals.up.sql` | econ2_goals Tabelle + goal_id FK auf orders | ✅ |

---

## Economy2-System — implementierte Dateien

| Datei | Inhalt |
|---|---|
| `econ2_recipes_v1.0.yaml` | 21 Rezepte: Mine (6), Smelter (2), Raffinerie (2), Präzision (2), Construction (9) |
| `game-params_v1.8.yaml` | mine-Params (base_rate, level_multiplier), economy2_bootstrap-Config |
| `internal/economy2/recipe.go` | Recipe/RecipeBook/RecipeKey; JSON+YAML-Tags; `All()` |
| `internal/economy2/order.go` | ProductionOrder; OrderType (batch/continuous/build); CRUD |
| `internal/economy2/facility.go` | Facility; FacilityConfig; `Destroy()` (Transaktion); CRUD |
| `internal/economy2/stock.go` | ItemStock; NodeStock, AddToStock, ConsumeAllocated; GetOrCreateNode |
| `internal/economy2/route.go` | Route; AllocateCapacity; CRUD |
| `internal/economy2/mrp.go` | ResolveDemand, AllocateOrder |
| `internal/economy2/scheduler.go` | SchedulerHandler: MRP-Allokation + Zuweisung idle→running |
| `internal/economy2/build.go` | BuildTickHandler: Construction-Orders ticken, bei Abschluss CreateFacility |
| `internal/economy2/production.go` | ProductionHandler: Tick-Produktion, Mine-Abbau, Effizienz-Akkumulation |
| `internal/economy2/mine.go` | MineParams, RateForLevel, Deposit-Abbau |
| `internal/economy2/deposit.go` | readDeposit, countActiveMines |
| `internal/economy2/ship.go` | ShipTickHandler |
| `internal/economy2/bootstrap.go` | RunBootstrap (Startpaket: Stock + Facilities) |
| `internal/economy2/handlers.go` | 19 REST-Routen: facilities (CRUD+start/stop), orders, routes, stock, nodes, my-nodes, bootstrap, recipes, goals (CRUD+reorder), stock-all, facilities-all, orders-all |
| `internal/tick/engine.go` | Engine: Register, Start, Advance (→tick-Nr.), Current() |
| `internal/api/router.go` | NewRouter; `/api/v2/admin/tick/advance` + `/current` |
| `frontend/src/types/economy2.ts` | Node, ItemStock, Facility, Order, Route, Recipe, MyNodeEntry |
| `frontend/src/api/economy2.ts` | Alle API-Calls inkl. listRecipes(); createOrder mit order_type:'build' |
| `frontend/src/pages/Economy2Page.tsx` | Komplett neu: Shell + 3 Tabs (PLAN/FABRIKEN/NETZWERK) + LeftRail, TickGenerator mit Advance-API, NATS Live-Updates |
| `frontend/src/components/economy2/ui.tsx` | Shared UI-Primitives (Card, Button, StatusBadge, StatusLamp, itemLabel, factoryLabel) |
| `frontend/src/components/economy2/BOMTree.tsx` | Rekursiver BOM-Baum mit client-seitiger Status-Berechnung (7 Zustände) |
| `frontend/src/components/economy2/FixPanel.tsx` | Inline Fix-Panels (keine Fabrik / Route fehlt / Item fehlt + Transport-Override) |
| `frontend/src/components/economy2/PlanTab.tsx` | GoalPicker + BOM-Baum pro Ziel |
| `frontend/src/components/economy2/FabrikenTab.tsx` | Anlagen gruppiert nach Stern + Cytoscape-Produktionsgraph |
| `frontend/src/components/economy2/NetzwerkTab.tsx` | Node-Karten + Route-Schematik |
| `frontend/src/components/economy2/LeftRail.tsx` | Drag-to-reorder Goals, Low-Stock Alerts, Lager-Summary |
| `frontend/src/components/economy2/ProductionGraph.tsx` | Cytoscape + dagre: Güter=Rechtecke, Anlagen=Hexagons, Kanten=Orders |
| `frontend/src/components/PlanetInspector.tsx` | bootstrap() beim Heimatplaneten anlegen |
| `cmd/devctl/main.go` | galaxis-devctl: Prozessmanager, HTTP-API, SSE Log-Streaming |
| `cmd/devctl/ui.go` | galaxis-devctl: eingebettetes HTML/JS Dashboard |
