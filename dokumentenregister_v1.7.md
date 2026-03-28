# Dokumentenregister – Galaxis

**Projektstatus:** Economy2 lauffähig · Messaging-Architektur (NATS) spezifiziert · Nächster Schritt: `feat/messaging` Worktree
**GDD-Version:** v1.24
**Datum:** 2026-03-28

---

## Dokumente

| Dokument | Zweck | Status |
|---|---|---|
| `dokumentenregister_v1.7.md` | Projektstatus-Übersicht | ✅ aktuell |
| `architecture_v1.1.md` | Systemarchitektur inkl. NATS, Economy2 | ✅ aktuell |
| `tech-decisions_v1.0.md` | ADRs 001–009 | ✅ aktuell |
| `progress_v1.5.md` | Sprint-Log, offene Punkte | ✅ aktuell |
| `docs/messaging_concept_v1.0.md` | Bus-Interface-Design, Subject-Schema, Delivery-Tiers, Broker-Evaluation | ✅ aktuell |
| `docs/messaging_nats_v1.0.md` | NATS Implementierung: Server-Config, Go-Adapter, Browser-Client (nats.ws), Auth-Flow | ✅ aktuell |
| `economy_v1.0.md` | Wirtschaftsmodell (Rohstoffe → Schiffe, Markt, Logistik) | ✅ Design finalisiert |
| `production-mechanics_v1.0.md` | Produktionsmechanik: Einheitensystem, Sensitivitätsklassen, Rezepte, Lagermodell | ✅ Design finalisiert |
| `economy-mvp-architecture_v1.0.md` | MVP-Architektur altes Economy-System | ⚠️ veraltet (Economy2 nicht dokumentiert) |
| `econ2_recipes_v1.0.yaml` | Produktionsrezepte Economy2 (Mine ×24, Smelter, Raffinerie, Präzision, Construction) | ✅ aktuell |
| `game-params_v1.8.yaml` | Spielparameter: Mine-Params, Bootstrap-Config | ✅ aktuell |
| `security_v1.0.md` | Threat Model, Auth, Secrets, Rate Limiting | ✅ aktuell |
| `git-commit-guide_v1.0.md` | Branch-Strategie, Conventional Commits | ✅ aktuell |
| `docs/start-guide.md` | Start-Anleitung für galaxis-devctl | ✅ aktuell |
| `server-core-map_v1.0.md` | Kartenfunktionen-Spezifikation | ✅ aktuell |
| `sensor-fow_v1.0.md` | Sensor-Mechanik & Fog of War | ✅ aktuell |
| `spielanleitung_v1.0.md` | Spielanleitung (spielerorientiert) | ✅ aktuell |
| `research-mechanics_v1.0.md` | Forschungsmechanik (stochastisches Modell) | ✅ aktuell |
| `biochemistry_archetypes_v1.0.yaml` | Biochemie-Archetypen für Alien-Spezies | ✅ aktuell |
| `galaxy_morphology_catalog_v1.0.yaml` | 75 Morphologie-Templates | ✅ aktuell |
| `tech-tree_v1.0.jsonld` | Technologiebaum als JSON-LD | ✅ v1 |

---

## Arbeitspakete

| AP | Bezeichnung | Status |
|---|---|---|
| AP0 | Game Design Document | ✅ abgeschlossen (GDD v1.24) |
| AP1 | Galaxiengenerator + Admin-Tool | ✅ implementiert |
| AP2 | Planetensystem-Generator | ✅ Basis · BL-13/14/16/17/19/25 offen |
| AP3 | Server-Core & Tick-Engine | 🔄 Skeleton + Tick-Engine ✅ · Auth/JWT ausstehend · **WebSocket → NATS** |
| AP4 | Wirtschaftssystem & Ressourcen | 🔄 Economy2 ✅ · Integrations-Tests + Transport-Pipeline ausstehend |
| AP5 | FTL-Navigation & Flottenlogistik | 🔲 ausstehend |
| AP6 | Schiffsdesign, Kampf, Sensoren | 🔲 ausstehend |
| AP7 | Technologie- & Forschungsbaum | 📋 v1 spezifiziert |
| **AP8** | **Messaging (NATS + Live-UI)** | 📋 **spezifiziert · Worktree ausstehend** |

---

## Stack

| Schicht | Technologie | Details |
|---|---|---|
| Backend | Go 1.25, chi v5, pgx/v5 | REST API, Tick-Engine |
| Datenbank | PostgreSQL 16 (Docker) | 13 Migrationen |
| **Messaging** | **NATS 2.10 (Docker), JetStream** | **AP8 — noch nicht implementiert** |
| Frontend | React 19, Vite 5, TypeScript, Tailwind | Economy2Page, God-Mode-Viewer |
| 3D | Three.js + R3F + Drei | Galaxie- + Systemansicht |
| Dev-Tools | galaxis-devctl (:9191) | Start/Stop/Logs für postgres, api, frontend |

---

## Nächste Schritte (priorisiert)

| Priorität | Aufgabe | Dokument |
|---|---|---|
| 🔥 | `feat/messaging` Worktree anlegen | `docs/messaging_nats_v1.0.md` Schritt 1–12 |
| 🔥 | Economy2 Integrations-Tests | `progress_v1.5.md` |
| Mittel | Transport-Routen UI (Node-Dropdowns) | — |
| Mittel | MRP: Bau-Aufträge aus tryAllocatePending ausschließen | — |
| Mittel | Auth/JWT (AP3) | — |
