# Dokumentenregister – Galaxis

**Projektstatus:** AP4 weitgehend implementiert — Facility-Spezialisierung, 3-Ebenen-Lager, Fertigungsaufträge + Pool-Scheduler lauffähig · BOM-Resolver + Integrations-Tests konzipiert, ausstehend
**GDD-Version:** v1.24
**Datum:** 2026-03-23

---

## Dokumente

| Dokument | Zweck | Status |
|---|---|---|
| `dokumentenregister_v1.5.md` | Projektstatus-Übersicht | ✅ aktuell |
| `architecture_v1.0.md` | Systemarchitektur, Komponenten, Datenfluss | ⚠️ Migrationen 007–009, storage_nodes, Scheduler noch nicht eingearbeitet |
| `tech-decisions_v1.0.md` | ADRs / Stack-Entscheidungen | ✅ aktuell |
| `progress_v1.4.md` | Sprint-Log, offene Punkte, nächste Schritte | ✅ aktuell |
| `economy_v1.0.md` | Wirtschaftsmodell (Rohstoffe → Schiffe, Markt, Logistik) | ✅ Design finalisiert |
| `production-mechanics_v1.0.md` | Produktionsmechanik: Einheitensystem, Sensitivitätsklassen, Rezepte, Lagermodell, Pipeline-Konzept | ✅ Design finalisiert |
| `economy-mvp-architecture_v1.0.md` | MVP-Architektur: DB-Schema, API-Routen, Go-Pakete | ⚠️ Stand Schritt 9 — Schritte 1–3 (Spezialisierung, storage_nodes, Orders) noch nicht eingearbeitet |
| `recipes_v1.1.yaml` | 33 Produktionsrezepte mit `output_good`-Feld, steel_chromium T2 | ✅ aktuell |
| `game-params_v1.8.yaml` | Zentrale Spielparameter: 9 spezialisierte Anlagentypen, vollständiger Production-Block | ✅ aktuell |
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
| AP3 | Server-Core & Tick-Engine | 🔄 Skeleton + Economy-Tick ✅ · Auth/WebSocket/Redis ausstehend |
| AP4 | Wirtschaftssystem & Ressourcen | 🔄 Design ✅ · Backend (Schritte 1–3) ✅ · BOM-Resolver + Tests ausstehend · Pipeline (Schritt 4) ausstehend |
| AP5 | FTL-Navigation & Flottenlogistik | 🔲 ausstehend |
| AP6 | Schiffsdesign, Kampf, Sensoren, UI | 🔲 ausstehend |
| AP7 | Technologie- & Forschungsbaum | 📋 v1 spezifiziert (JSON-LD) |

---

## Implementierter Stack (aktuell)

| Schicht | Technologie | Details |
|---|---|---|
| Backend | Go 1.25, chi v5, pgx/v5 | REST API, SSE, Job-Store, Economy-Tick-Engine |
| Datenbank | PostgreSQL 16 (Docker) | 9 Migrationen, golang-migrate |
| Cache | Redis (Docker) | Noch ungenutzt |
| Frontend | React 19, Vite, TypeScript, Tailwind | God-Mode-Viewer + Generator-UI + Systemansicht + EconomyPage |
| 3D | Three.js + React Three Fiber + Drei | Galaxie- + System-Canvas (Kepler-Ellipsen, prozedurale Shader) |
| Scraper | Python + SIMBAD/SDSS + Gemini 2.5 Flash | 75 Morphologie-Templates |
| Tests | Vitest (Frontend, 25 Unit-Tests) · Go Integrations-Tests (ausstehend) | |
