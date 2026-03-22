# Dokumentenregister – Galaxis

**Projektstatus:** AP1 + AP2 implementiert · AP4 Backend vollständig implementiert · Frontend EconomyPage ausstehend
**GDD-Version:** v1.24
**Datum:** 2026-03-22

---

## Dokumente

| Dokument | Zweck | Status |
|---|---|---|
| `dokumentenregister_v1.4.md` | Projektstatus-Übersicht | ✅ aktuell |
| `architecture_v1.0.md` | Systemarchitektur, Komponenten, Datenfluss | ⚠️ Migration 006, Economy-Paket noch nicht eingearbeitet |
| `tech-decisions_v1.0.md` | ADRs / Stack-Entscheidungen | ✅ aktuell |
| `progress_v1.3.md` | Sprint-Log, offene Punkte, nächste Schritte | ✅ aktuell |
| `server-core-map_v1.0.md` | Spezifikation Kartenfunktionen (Erzeugung + Laufzeit) | ✅ aktuell |
| `sensor-fow_v1.0.md` | Sensor-Mechanik & Fog of War | ✅ aktuell |
| `performance-estimate_v1.0.md` | Größen- und Performance-Abschätzung FTLW-Grid | ✅ aktuell |
| `spielanleitung_v1.0.md` | Spielanleitung (spielerorientiert) | ✅ aktuell |
| `tech-tree_v1.0.jsonld` | Technologiebaum als JSON-LD | ✅ v1 (erweiterbar) |
| `game-params_v1.6.yaml` | Zentrale Spielparameter inkl. vollständigem Production-Block (Sensitivity, Deposits, Survey, Elevator, Warnings, ColonyShip) | ✅ aktuell |
| `tdd_image_galaxy_generator_v1.0.md` | TDD: Image-Based Galaxy Generator | ✅ aktuell |
| `security_v1.0.md` | Threat Model, Auth (AP3/Keycloak), Secrets, Rate Limiting, FoW | ✅ aktuell |
| `git-commit-guide_v1.0.md` | Branch-Strategie, Conventional Commits, Scopes | ✅ aktuell |
| `biochemistry_archetypes_v1.0.yaml` | Biochemie-Archetypen für Alien-Spezies | ✅ aktuell |
| `galaxy_morphology_catalog_v1.0.yaml` | Katalog realer Galaxienfotos (75 Einträge, 21 Hubble-Typen) | ✅ aktuell |
| `research-mechanics_v1.0.md` | Forschungsmechanik (stochastisches Modell) | ✅ aktuell |
| `economy_v1.0.md` | Wirtschaftsmodell: Rohstoffe → Halbzeug → Komponenten → Schiffe, Markt, Logistik | ✅ Design finalisiert |
| `production-mechanics_v1.0.md` | Produktionsmechanik: Einheitensystem, Sensitivitätsklassen I–IV, 7 Anlagen, Rezepte, Lagermodell, Bootstrap, Pipeline-UI-Konzept (Post-MVP) | ✅ Design finalisiert · Pipeline-Graph-Konzept ergänzt |
| `economy-mvp-architecture_v1.0.md` | MVP-Architektur: E1–E5/F1–F3/D1–D8 Entscheidungen, DB-Schema (5 Tabellen), API-Routen, Go-Pakete, Implementierungsreihenfolge | ✅ Backend implementiert · Frontend ausstehend |
| `recipes_v1.0.yaml` | Alle Produktionsrezepte (Stufe 2–4), in-memory geladen, nie in DB | ✅ aktuell (30 Rezepte) |

---

## Arbeitspakete (Übersicht)

| AP | Bezeichnung | Phase | Status |
|---|---|---|---|
| AP0 | Game Design Document | Konzept | ✅ abgeschlossen (GDD v1.24) |
| AP1 | Galaxiengenerator (Makro) + Admin-Tool | Weltgenerator | ✅ implementiert (50k Sterne, FTLW-Grid, Image-Based Generator BL-11) |
| AP2 | Planetensystem-Generator (Mikro) | Weltgenerator | ✅ Basis + BL-12/15/18/20/21/24 fertig · BL-13/14/16/17/19/25 offen |
| AP3 | Server-Core & Tick-Engine | Backend | 🔄 Skeleton ✅ · Tick-Engine erweitert (Advance()) · Remainder ausstehend (Auth, WebSocket, Redis) |
| AP4 | Wirtschaftssystem & Ressourcen | Backend | 🔄 Design finalisiert · **Backend vollständig implementiert** · Frontend ausstehend |
| AP5 | FTL-Navigation & Flottenlogistik | Backend | 🔲 ausstehend |
| AP6 | Schiffsdesign, Kampf, Sensoren, UI | Frontend + Backend | 🔲 ausstehend |
| AP7 | Technologie- & Forschungsbaum | Design + Backend | 📋 v1 spezifiziert (JSON-LD) |

---

## Implementierter Stack (aktuell)

| Schicht | Technologie | Details |
|---|---|---|
| Backend | Go 1.23, chi v5, pgx/v5 | REST API, SSE-Progress, Job-Store, Economy-Tick-Engine |
| Datenbank | PostgreSQL 16 (Docker) | 6 Migrationen, golang-migrate |
| Cache | Redis (Docker) | Noch ungenutzt (AP3 Remainder) |
| Frontend | React 19, Vite, TypeScript, Tailwind | God-Mode-Viewer + Generator-UI + Systemansicht |
| 3D | Three.js + React Three Fiber + Drei | Galaxie-Canvas + System-Canvas (3D, Kepler-Ellipsen, prozedurale Shader) |
| Scraper | Python + SIMBAD TAP + SDSS + Gemini 2.5 Flash | 75 Morphologie-Templates |
| Tests | Vitest (node environment) | 25 Unit-Tests: uuidSeed, starColorTriad, calcPlanetVisR, computeOrbitPos, makeLcg |

---

## Economy-Backend (AP4) — implementierte Dateien

| Datei | Inhalt |
|---|---|
| `migrations/006_economy.up.sql` | 5 Tabellen: planet_deposits, facilities, system_storage, production_log, player_surveys |
| `internal/config/config.go` | ProductionConfig + Untertypen ergänzt |
| `internal/economy/registry.go` | LoadRegistries() — lädt recipes.yaml + game-params in-memory |
| `internal/economy/deposit.go` | Lazy Init, D1-Formel, Deplete, GetDeposits |
| `internal/economy/survey.go` | ExecuteSurvey, GetSurvey, filterSnapshot, UpdateOwnMiningSnapshot |
| `internal/economy/storage.go` | GetStorage, SetStorage, Has, Consume, Produce |
| `internal/economy/production.go` | ProductionHandler, Mine-Sonderfall, Rezept-Tick-Algorithmus, Deposit-Warnungen |
| `internal/economy/log.go` | appendLog, pruneOldLogs, GetLog (rolling 100 Ticks) |
| `internal/economy/player.go` | PlayerZeroID Konstante |
| `internal/economy/broadcast.go` | SSE-Broadcaster für Tick-Events |
| `internal/api/economy_handlers.go` | 9 REST/SSE-Routen |
| `internal/tick/engine.go` | Advance() + shared atomic tickCounter |
| `cmd/server/main.go` | Registries, Broadcaster, ProductionHandler verdrahtet |

---

## BL-Items (Backlog – AP2 Systemansicht)

| ID | Beschreibung | Status |
|---|---|---|
| BL-12 | Elliptische Orbits + Temperaturgrenzen | ✅ erledigt |
| BL-15 | Systembaum Master-Detail | ✅ erledigt |
| BL-18 | Logarithmische Planetengrößen | ✅ erledigt |
| BL-20 | Prozedurale Stern-/Planeten-/Mondshader | ✅ erledigt |
| BL-21 | Asteroidengürtel visuell | ✅ erledigt |
| BL-24 | Doppelklick Planet → Mondsystem-Ansicht | ✅ erledigt |
| BL-13 | Planetenrotation + Bahn-Spin-Kopplung | 🔲 offen |
| BL-14 | Mehrfachsternsysteme | 🔲 offen |
| BL-16 | Benennungssystem für Systeme und Planeten | 🔲 offen |
| BL-17 | Spektralunterklassen (MK-System) | 🔲 offen |
| BL-19 | Hover-Details für Himmelskörper | 🔲 offen |
| BL-25 | Planetenringe (Saturn-artig) | 🔲 offen |

---

## Offene Entscheidungen

| Thema | Status |
|---|---|
| System-ID-Schema (BL-16) | TBD – Hoch (vor AP3-Auth) |
| Planeten-ID-Schema (BL-16) | TBD – Hoch (vor AP3-Auth) |
| Produktions-Cloud: AWS vs. GCP | TBD (erst bei Deployment) |
| Frontend-Zustandsmanagement | TBD (vor AP5/AP6) |
| FTLW Cut-off-Wert | TBD (Balancing) |
| Fiktive Tier-5-Ressourcen – Namen & Eigenschaften | TBD (Niedrig) |
