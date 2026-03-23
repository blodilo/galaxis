# Progress – Galaxis v1.3

**Datum:** 2026-03-22

---

## Aktueller Status

**Phase:** AP1 + AP2 implementiert · AP4 Backend vollständig implementiert (Migration 006, Go-Economy-Paket, 9 API-Routen, Tick-Integration) · Frontend EconomyPage ausstehend

---

## Erledigte Meilensteine

| Datum | Meilenstein |
|---|---|
| 2026-03-12 | GDD v1.24 finalisiert (Gemini-Chat vollständig) |
| 2026-03-12 | Stack & Architektur-Entscheidungen getroffen (ADR-001 bis ADR-007) |
| 2026-03-12 | Pflichtdokumente erstellt (architecture, tech-decisions, progress, dokumentenregister) |
| 2026-03-13 | Spezifikationsphase abgeschlossen (Karte, Sensoren, Forschung, Tech Tree, Spielparameter) |
| 2026-03-13 | AP3 Server-Core Skeleton implementiert |
| 2026-03-13 | AP1 Galaxiengenerator vollständig implementiert (Dichte, Sterne, Nebel, FTLW-Grid) |
| 2026-03-13 | God-Mode-Viewer (React/Three.js) lauffähig |
| 2026-03-14 | AP2 vollständig spezifiziert: physikalisches Atmosphärenmodell, 5 Biochemie-Archetypen, 24 Ressourcengruppen |
| 2026-03-14 | biochemistry_archetypes_v1.0.yaml erstellt |
| 2026-03-14 | BL-08: Generator-Admin-Tool implementiert |
| 2026-03-17 | BL-11 (BL-03/BL-09): Image-Based Galaxy Generator – CDF+Inverse-Transform-Sampling, Spektral-Kaskade, Exotika, SSE Progress, game-params v1.3 |
| 2026-03-20 | BL-03: Galaxy-Scraper (SIMBAD + SDSS + Gemini Vision QA) – 75 Templates, 21 Hubble-Typen |
| 2026-03-20 | BL-11 Fix: Deep Randomization Layer (zoneProbTable) – Spektral-Farbbänder beseitigt |
| 2026-03-20 | AP2 Planetensystem-Generator lauffähig: Titius-Bode, Frostlinie, Atmosphären, Biochemie, Monde, Ressourcen |
| 2026-03-20 | Systemansicht (SystemScene) lauffähig, moons-null-Bug behoben |
| 2026-03-20 | BL-12: Elliptische Orbits – Kepler-Parameter (e, ω, i, Perihel, Aphel, T_eq_min/max), Rayleigh-Sampling, Migration 004 |
| 2026-03-20 | BL-15: Systembaum Master-Detail (SystemTree: Stern → Planeten → Monde, aufklappbar, bidirektionale Selektion) |
| 2026-03-20 | BL-18: Logarithmische Planetengrößen (calcPlanetVisR, orbital-gap clamp) |
| 2026-03-20 | BL-12 Erw.: Mondorbits physikalisch korrekt – Hill-Sphäre, geometrische Staffelung, Migration 005 |
| 2026-03-20 | Systemansicht 3D-Ansicht (enableRotate, Schräg-Kamera) |
| 2026-03-20 | Breadcrumbs Galaxie › Stern › Planet (links→rechts, klickbare Navigation) |
| 2026-03-20 | Bugfixes: Makefile Port (:8090), schema_migrations dirty-Flag, Katalog-Duplikat, GeneratorPage Step-Index off-by-one |
| 2026-03-21 | **BL-20:** Prozedurale Stern-/Planeten-/Mondshader (Hash-Noise, Limb-Darkening, Granulation, Prominenzen) |
| 2026-03-21 | **BL-21:** Asteroidengürtel visuell (InstancedMesh, LCG-Seed, Potenzgesetz, Staub-Ring-Shader) |
| 2026-03-21 | **BL-24:** Doppelklick Planet → Mondsystem-Ansicht (MoonSystemScene, SmartOrbitControls) |
| 2026-03-21 | Visual Tuner, Layer-Steuerung, Selektion-Aura, OrbitChevron, SmartOrbitControls |
| 2026-03-21 | Vitest Test-Infrastruktur – 25 Unit-Tests |
| 2026-03-22 | **AP4 Design:** economy_v1.0.md + production-mechanics_v1.0.md finalisiert · Alle 8 Entscheidungen (D1–D8) getroffen |
| 2026-03-22 | **AP4 Design:** economy-mvp-architecture_v1.0.md – DB-Schema, API-Routen, Go-Pakete, Implementierungsreihenfolge |
| 2026-03-22 | **AP4 Design:** recipes_v1.0.yaml – 30 Rezepte Stufe 2–4 · game-params_v1.6.yaml – vollständiger Production-Block |
| 2026-03-22 | **AP4 Backend:** Migration 006 – 5 Tabellen (planet_deposits, facilities, system_storage, production_log, player_surveys) + auf DB angewendet |
| 2026-03-22 | **AP4 Backend:** `internal/config/config.go` – ProductionConfig + alle Untertypen (Sensitivity, Deposit, Survey, Elevator, Warnings, ColonyShip) |
| 2026-03-22 | **AP4 Backend:** `internal/economy/` – registry, deposit, survey, storage, production, log, player, broadcast (7 Dateien, ~700 LOC) |
| 2026-03-22 | **AP4 Backend:** `internal/api/economy_handlers.go` – 9 REST/SSE-Routen verdrahtet |
| 2026-03-22 | **AP4 Backend:** `internal/tick/engine.go` – Advance() + shared atomic tickCounter |
| 2026-03-22 | **AP4 Backend:** `cmd/server/main.go` – Registries, Broadcaster, ProductionHandler verdrahtet |
| 2026-03-22 | **AP4 Konzept:** Pipeline-Graph-Modell dokumentiert (Post-MVP) in production-mechanics_v1.0.md |

---

## Nächste Schritte (priorisiert)

| Priorität | Aufgabe |
|---|---|
| 🔥 Sofort | **AP4 Frontend:** EconomyPage (`/economy/:starId`) – Tab-Button, TickControls, DepositCards, FacilitiesSection, StorageTable, BuildPanel, EventLog |
| Hoch | AP4 E2E-Test: Kolonisierung → Survey → Build → Tick × 30 → Zielzustand |
| Hoch | BL-16 Benennungssystem (vor AP3-Auth) |
| Mittel | AP3 Remainder (Auth/JWT, WebSocket, Redis) |
| Mittel | BL-13 Planetenrotation, BL-14 Mehrfachsternsysteme |
| Niedrig | BL-19 Hover-Details, BL-25 Planetenringe |

---

## Backlog – Planetensystem-Generator (AP2)

| ID | Thema | Status |
|---|---|---|
| BL-12 | Elliptische Orbits + Temperaturgrenzen | ✅ erledigt |
| BL-15 | Systembaum Master-Detail | ✅ erledigt |
| BL-18 | Logarithmische Planetengrößen | ✅ erledigt |
| BL-20 | Prozedurale Shader | ✅ erledigt |
| BL-21 | Asteroidengürtel visuell | ✅ erledigt |
| BL-24 | Mondsystem-Ansicht | ✅ erledigt |
| BL-13 | Planetenrotation + Bahn-Spin-Kopplung | 🔲 offen |
| BL-14 | Mehrfachsternsysteme | 🔲 offen |
| BL-16 | Benennungssystem | 🔲 offen |
| BL-17 | Spektralunterklassen | 🔲 offen |
| BL-19 | Hover-Details | 🔲 offen |
| BL-25 | Planetenringe | 🔲 offen |

---

## DB-Migrationen (Übersicht)

| Nr | Datei | Inhalt |
|---|---|---|
| 001 | `001_initial.up.sql` | Grundschema: galaxies, stars, nebulae, planets, moons, ftlw_cells |
| 002 | `002_planet_model.up.sql` | Physikalisches Atmosphärenmodell |
| 003 | `003_galaxy_status_steps.up.sql` | Galaxy-Status-Werte |
| 004 | `004_orbital_mechanics.up.sql` | Kepler-Orbital-Parameter |
| 005 | `005_moon_orbits.up.sql` | Mondorbit-Abstände (Hill-Sphäre) |
| 006 | `006_economy.up.sql` | Economy: planet_deposits, facilities, system_storage, production_log, player_surveys ✅ angewendet |

---

## Offene Entscheidungen (TBD)

| Thema | Priorität |
|---|---|
| System-ID-Schema (BL-16) | Hoch (vor AP3-Auth) |
| Planeten-ID-Schema (BL-16) | Hoch (vor AP3-Auth) |
| Produktions-Cloud: AWS vs. GCP | Mittel (erst bei Deployment) |
| Frontend-Zustandsmanagement | Mittel (vor AP5/AP6) |
| FTLW Cut-off-Wert | Niedrig (Balancing) |
| Fiktive Tier-5-Ressourcen | Niedrig |
