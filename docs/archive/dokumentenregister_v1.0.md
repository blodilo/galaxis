# Dokumentenregister – Galaxis

**Projektstatus:** AP1 + AP2 implementiert und lauffähig · God-Mode-Viewer + Generator-Admin-Tool + Image-Based Generator · Systemansicht mit 3D, elliptischen Orbits, Master-Detail-Baum, physikalischen Mondorbits
**GDD-Version:** v1.24
**Datum:** 2026-03-20

---

## Dokumente

| Dokument | Zweck | Status |
|---|---|---|
| `dokumentenregister_v1.0.md` | Projektstatus-Übersicht | ✅ aktuell |
| `architecture_v1.0.md` | Systemarchitektur, Komponenten, Datenfluss | ⚠️ Migrations 004+005 noch nicht eingearbeitet |
| `tech-decisions_v1.0.md` | ADRs / Stack-Entscheidungen | ✅ aktuell |
| `progress_v1.1.md` | Sprint-Log, offene Punkte, nächste Schritte | ✅ aktuell |
| `server-core-map_v1.0.md` | Spezifikation Kartenfunktionen (Erzeugung + Laufzeit) | ✅ aktuell |
| `sensor-fow_v1.0.md` | Sensor-Mechanik & Fog of War | ✅ aktuell |
| `performance-estimate_v1.0.md` | Größen- und Performance-Abschätzung FTLW-Grid | ✅ aktuell |
| `spielanleitung_v1.0.md` | Spielanleitung (spielerorientiert) | ✅ aktuell |
| `tech-tree_v1.0.jsonld` | Technologiebaum als JSON-LD | ✅ v1 (erweiterbar) |
| `game-params_v1.3.yaml` | Zentrale Spielparameter (Kalibrierung, Balancing, Performance-Limits) | ✅ aktuell |
| `tdd_image_galaxy_generator_v1.0.md` | TDD: Image-Based Galaxy Generator | ✅ aktuell |
| `biochemistry_archetypes_v1.0.yaml` | Biochemie-Archetypen für Alien-Spezies | ✅ aktuell |
| `galaxy_morphology_catalog_v1.0.yaml` | Katalog realer Galaxienfotos (75 Einträge, 21 Hubble-Typen) | ✅ aktuell |
| `research-mechanics_v1.0.md` | Forschungsmechanik (stochastisches Modell) | ✅ aktuell |

---

## Arbeitspakete (Übersicht)

| AP | Bezeichnung | Phase | Status |
|---|---|---|---|
| AP0 | Game Design Document | Konzept | ✅ abgeschlossen (GDD v1.24) |
| AP1 | Galaxiengenerator (Makro) + Admin-Tool | Weltgenerator | ✅ implementiert (50k Sterne, FTLW-Grid, Image-Based Generator BL-11) |
| AP2 | Planetensystem-Generator (Mikro) | Weltgenerator | ✅ Basis implementiert · BL-12/15/18 fertig · BL-13/14/16/17/19/20/21 offen |
| AP3 | Server-Core & Tick-Engine | Backend | 🔄 Skeleton ✅ · Remainder ausstehend (Auth, WebSocket, Redis) |
| AP4 | Wirtschaftssystem & Ressourcen | Backend | 🔲 ausstehend |
| AP5 | FTL-Navigation & Flottenlogistik | Backend | 🔲 ausstehend |
| AP6 | Schiffsdesign, Kampf, Sensoren, UI | Frontend + Backend | 🔲 ausstehend |
| AP7 | Technologie- & Forschungsbaum | Design + Backend | 📋 v1 spezifiziert (JSON-LD) |

---

## Implementierter Stack (aktuell)

| Schicht | Technologie | Details |
|---|---|---|
| Backend | Go 1.23, chi v5, pgx/v5 | REST API, SSE-Progress, Job-Store |
| Datenbank | PostgreSQL 16 (Docker) | 5 Migrationen, golang-migrate |
| Cache | Redis (Docker) | Noch ungenutzt (AP3 Remainder) |
| Frontend | React 19, Vite, TypeScript, Tailwind | God-Mode-Viewer + Generator-UI + Systemansicht |
| 3D | Three.js + React Three Fiber + Drei | Galaxie-Canvas + System-Canvas (3D, Kepler-Ellipsen) |
| Scraper | Python + SIMBAD TAP + SDSS + Gemini 2.5 Flash | 75 Morphologie-Templates |

---

## Offene Entscheidungen

| Thema | Status |
|---|---|
| System-ID-Schema (BL-16) | TBD – Hoch (vor AP3-Auth) |
| Planeten-ID-Schema (BL-16) | TBD – Hoch (vor AP3-Auth) |
| Produktions-Cloud: AWS vs. GCP | TBD (erst bei Deployment) |
| 3D-Rendering-Library (Three.js / Babylon.js) | TBD (vor AP6-Frontend) – tendiert zu Three.js (bereits im Einsatz) |
| Frontend-Zustandsmanagement | TBD (vor AP5/AP6) |
| FTLW Cut-off-Wert | TBD (Balancing) |
| Fiktive Tier-5-Ressourcen – Namen & Eigenschaften | TBD (Niedrig) |
