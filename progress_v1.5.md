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

---

## Nächste Schritte (priorisiert)

| Priorität | Aufgabe |
|---|---|
| 🔥 Hoch | **Balancing** — Mine-Build-Ticks auf 1 gesetzt (Dev); Produktion testen mit Tick-Generator |
| 🔥 Hoch | **Integrations-Tests Economy2** — `go test ./internal/economy2/...` mit echter DB |
| Mittel | **MRP-Verbesserung** — Bau-Aufträge aus `tryAllocatePending` ausschließen (construction hat keine Inputs die aufgelöst werden müssen falls leer) |
| Mittel | **Transport-Routen UI** — Node-IDs aus Dropdown statt UUID-Freitext |
| Mittel | **Continuous-Aufträge** — Anzeige ohne Fortschrittsbalken, Drosselungs-Warnung |
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
| `internal/economy2/handlers.go` | 10 REST-Routen: facilities, orders, routes, stock, nodes, my-nodes, bootstrap, recipes |
| `internal/tick/engine.go` | Engine: Register, Start, Advance (→tick-Nr.), Current() |
| `internal/api/router.go` | NewRouter; `/api/v2/admin/tick/advance` + `/current` |
| `frontend/src/types/economy2.ts` | Node, ItemStock, Facility, Order, Route, Recipe, MyNodeEntry |
| `frontend/src/api/economy2.ts` | Alle API-Calls inkl. listRecipes(); createOrder mit order_type:'build' |
| `frontend/src/pages/Economy2Page.tsx` | MyAssetsView, AnlagenPanel (Bau-Rezept-Dropdown), AuftraegePanel (Rezept-Dropdown), TickGenerator-Widget |
| `frontend/src/components/PlanetInspector.tsx` | bootstrap() beim Heimatplaneten anlegen |
