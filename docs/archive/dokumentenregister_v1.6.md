# Dokumentenregister – Galaxis

**Projektstatus:** Economy2-System lauffähig — Rezept-getriebene UI, Bau-als-Auftrag, Tick-Generator, galaxis-devctl · Nächster Schritt: Produktion verifizieren + Integrations-Tests
**GDD-Version:** v1.24
**Datum:** 2026-03-27

---

## Dokumente

| Dokument | Zweck | Status |
|---|---|---|
| `dokumentenregister_v1.6.md` | Projektstatus-Übersicht | ✅ aktuell |
| `architecture_v1.0.md` | Systemarchitektur, Komponenten, Datenfluss | ⚠️ Economy2-System noch nicht eingearbeitet |
| `tech-decisions_v1.0.md` | ADRs / Stack-Entscheidungen | ✅ aktuell |
| `progress_v1.5.md` | Sprint-Log, offene Punkte, nächste Schritte | ✅ aktuell |
| `economy_v1.0.md` | Wirtschaftsmodell (Rohstoffe → Schiffe, Markt, Logistik) | ✅ Design finalisiert |
| `production-mechanics_v1.0.md` | Produktionsmechanik: Einheitensystem, Sensitivitätsklassen, Rezepte, Lagermodell, Pipeline-Konzept | ✅ Design finalisiert |
| `economy-mvp-architecture_v1.0.md` | MVP-Architektur: DB-Schema, API-Routen, Go-Pakete | ⚠️ Beschreibt altes Economy-System; Economy2 nicht dokumentiert |
| `econ2_recipes_v1.0.yaml` | 21 Produktionsrezepte Economy2 (Mine, Smelter, Raffinerie, Präzision, Construction) | ✅ aktuell |
| `recipes_v1.1.yaml` | 33 Rezepte altes Economy-System | ✅ (altes System) |
| `game-params_v1.8.yaml` | Zentrale Spielparameter: Mine-Params, Bootstrap-Config, Production-Block | ✅ aktuell |
| `security_v1.0.md` | Threat Model, Auth, Secrets, Rate Limiting | ✅ aktuell |
| `git-commit-guide_v1.0.md` | Branch-Strategie, Conventional Commits | ✅ aktuell |
| `server-core-map_v1.0.md` | Kartenfunktionen-Spezifikation | ✅ aktuell |
| `sensor-fow_v1.0.md` | Sensor-Mechanik & Fog of War | ✅ aktuell |
| `performance-estimate_v1.0.md` | Größen-/Performance-Abschätzung FTLW-Grid | ✅ aktuell |
| `spielanleitung_v1.0.md` | Spielanleitung (spielerorientiert) | ✅ aktuell |
| `research-mechanics_v1.0.md` | Forschungsmechanik (stochastisches Modell) | ✅ aktuell |
| `tdd_image_galaxy_generator_v1.0.md` | TDD: Image-Based Galaxy Generator | ✅ aktuell |
| `biochemistry_archetypes_v1.0.yaml` | Biochemie-Archetypen für Alien-Spezies | ✅ aktuell |
| `galaxy_morphology_catalog_v1.0.yaml` | 75 Morphologie-Templates (SIMBAD/Hubble) | ✅ aktuell |
| `tech-tree_v1.0.jsonld` | Technologiebaum als JSON-LD | ✅ v1 (erweiterbar) |

---

## Arbeitspakete (Übersicht)

| AP | Bezeichnung | Status |
|---|---|---|
| AP0 | Game Design Document | ✅ abgeschlossen (GDD v1.24) |
| AP1 | Galaxiengenerator + Admin-Tool | ✅ implementiert (50k Sterne, FTLW-Grid, Image-Based Generator) |
| AP2 | Planetensystem-Generator | ✅ Basis + BL-12/15/18/20/21/24 · BL-13/14/16/17/19/25 offen |
| AP3 | Server-Core & Tick-Engine | 🔄 Skeleton + Tick-Engine ✅ · Auth/WebSocket/Redis ausstehend |
| AP4 | Wirtschaftssystem & Ressourcen | 🔄 Economy2: Produktion, MRP, Bau-als-Auftrag, Transport-Routen ✅ · Integrations-Tests + Pipeline ausstehend |
| AP5 | FTL-Navigation & Flottenlogistik | 🔲 ausstehend |
| AP6 | Schiffsdesign, Kampf, Sensoren, UI | 🔲 ausstehend |
| AP7 | Technologie- & Forschungsbaum | 📋 v1 spezifiziert (JSON-LD) |

---

## Implementierter Stack (aktuell)

| Schicht | Technologie | Details |
|---|---|---|
| Backend | Go 1.25, chi v5, pgx/v5 | REST API, SSE, Job-Store, Tick-Engine (manual advance + current) |
| Datenbank | PostgreSQL 16 (Docker) | 10 Migrationen, golang-migrate |
| Cache | Redis (Docker) | Noch ungenutzt |
| Frontend | React 19, Vite 5, TypeScript, Tailwind | God-Mode-Viewer + Systemansicht + Economy2Page (Rezept-UI, Tick-Generator) |
| 3D | Three.js + React Three Fiber + Drei | Galaxie- + System-Canvas (Kepler-Ellipsen, prozedurale Shader) |
| Scraper | Python + SIMBAD/SDSS + Gemini 2.5 Flash | 75 Morphologie-Templates |
| Tests | Vitest (Frontend, 25 Unit-Tests) · Go Integrations-Tests (ausstehend) | |

---

## Parallele Systeme (Strangler-Fig-Migration)

| System | Beschreibung | Status |
|---|---|---|
| `internal/economy` | Altes Economy-System (AP4 Schritte 1–3) | ✅ lauffähig, wird durch Economy2 abgelöst |
| `internal/economy2` | Neues Economy-System (Strangler-Fig) | ✅ aktiv in Entwicklung — läuft parallel |
