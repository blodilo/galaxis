# Dokumentenregister – Galaxis

**Projektstatus:** AP1 + AP3-Skeleton + God-Mode-Viewer + Generator-Admin-Tool implementiert · AP2 spezifiziert, Implementierung ausstehend
**GDD-Version:** v1.24
**Datum:** 2026-03-14

---

## Dokumente

| Dokument | Zweck | Status |
|---|---|---|
| `dokumentenregister_v1.0.md` | Projektstatus-Übersicht | ✅ aktuell |
| `architecture_v1.0.md` | Systemarchitektur, Komponenten, Datenfluss | ✅ aktuell |
| `tech-decisions_v1.0.md` | ADRs / Stack-Entscheidungen | ✅ aktuell |
| `progress_v1.0.md` | Sprint-Log, offene Punkte, nächste Schritte | ✅ aktuell |
| `server-core-map_v1.0.md` | Spezifikation Kartenfunktionen (Erzeugung + Laufzeit) | ✅ aktuell |
| `sensor-fow_v1.0.md` | Sensor-Mechanik & Fog of War (SR-Parameter, Detektionsreichweiten, Netzwerk) | ✅ aktuell |
| `performance-estimate_v1.0.md` | Größen- und Performance-Abschätzung FTLW-Grid, Maximalkartengröße | ✅ aktuell |
| `spielanleitung_v1.0.md` | Spielanleitung (spielerorientiert) – Sichtbarkeit, Sensoren, Exploration, Biochemie & Spezies-Wahl | ✅ aktuell |
| `tech-tree_v1.0.jsonld` | Technologiebaum als JSON-LD (kompatibel mit graph-Projekt-Ontologie) | ✅ v1 (erweiterbar) |
| `game-params_v1.1.yaml` | Zentrale Spielparameter (Kalibrierung, Balancing, Performance-Limits) | ✅ aktuell (arm_winding 0.35) |
| `biochemistry_archetypes_v1.0.yaml` | Biochemie-Archetypen für Alien-Spezies (physikalische Parameter, Treibhauswerte, Quellenangaben) | ✅ aktuell |
| `galaxy_morphology_catalog_v1.0.yaml` | Katalog realer Galaxienfotos als Morphologie-Templates (8 Einträge, Sa–Irr, Quellen + Lizenzen) | ✅ aktuell |
| `research-mechanics_v1.0.md` | Forschungsmechanik (stochastisches Modell, Inputs, Bauplan-Output) | ✅ aktuell |
| `graph-project-prompt_v1.0.md` | Prompt: 4 minimale Änderungen am graph-Projekt für extra_data-Import/Export | ✅ aktuell |
| `concept.md` | Python-Prototyp GalaxyGenerator (Referenz) | ✅ Referenz |
| `Gemini Galaxis UNTERNEHMEN.txt` | Vollständiger Design-Chat (GDD-Historie) | ✅ Archiv |

---

## Arbeitspakete (Übersicht)

| AP | Bezeichnung | Phase | Status |
|---|---|---|---|
| AP0 | Game Design Document | Konzept | ✅ abgeschlossen (GDD v1.24) |
| AP1 | Galaxiengenerator (Makro) + Admin-Tool | Weltgenerator | ✅ implementiert (50k Sterne, FTLW-Grid, Generator-Frontend BL-08) |
| AP2 | Planetensystem-Generator (Mikro) | Weltgenerator | 📋 vollständig spezifiziert (Eager + Biochemie-Archetypen) |
| AP3 | Server-Core & Tick-Engine | Backend | 🔄 Skeleton ✅ · Remainder ausstehend |
| AP4 | Wirtschaftssystem & Ressourcen | Backend | 🔲 ausstehend |
| AP5 | FTL-Navigation & Flottenlogistik | Backend | 🔲 ausstehend |
| AP6 | Schiffsdesign, Kampf, Sensoren, UI | Frontend + Backend | 🔲 ausstehend |
| AP7 | Technologie- & Forschungsbaum | Design + Backend | 📋 v1 spezifiziert (JSON-LD) |

---

## Offene Entscheidungen

| Thema | Status |
|---|---|
| Tech Tree – Erweiterung um fehlende Technologien (z.B. Elektronische Kampfführung, Akademien) | 🔄 in Arbeit |
| Ressourcen-Verteilungsformel pro Element und Planetentyp | ✅ Spektraltyp-Matrix definiert (24 Gruppen, ADR-006, biochemistry_archetypes_v1.0.yaml) |
| Biochemie-Archetypen – physikalische Parameter mit Quellenangaben | ✅ abgeschlossen (biochemistry_archetypes_v1.0.yaml, ADR-008) |
| Eager vs. JIT Planetengenerierung | ✅ entschieden (ADR-009: Eager für Dev, JIT für Produktion) |
| FTLW k-Faktor (Gameplay-Skalierung) | TBD (Balancing) |
| Produktions-Cloud: AWS vs. GCP | TBD (erst bei Deployment) |
| 3D-Rendering-Library (Three.js / Babylon.js) | TBD (vor AP6-Frontend) |
