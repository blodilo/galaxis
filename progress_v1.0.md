# Progress – Galaxis v1.0

**Datum:** 2026-03-16

---

## Aktueller Status

**Phase:** AP1 + God-Mode-Viewer implementiert und lauffähig · AP3 teilweise fertig · AP2 vollständig spezifiziert

---

## Erledigte Meilensteine

| Datum | Meilenstein |
|---|---|
| 2026-03-12 | GDD v1.24 finalisiert (Gemini-Chat vollständig) |
| 2026-03-12 | Stack & Architektur-Entscheidungen getroffen (ADR-001 bis ADR-007) |
| 2026-03-12 | Pflichtdokumente erstellt (architecture, tech-decisions, progress, dokumentenregister) |
| 2026-03-13 | Spezifikationsphase abgeschlossen (Karte, Sensoren, Forschung, Tech Tree, Spielparameter) |
| 2026-03-13 | AP3 Server-Core Skeleton implementiert (Go-Modul, Docker Compose, Config, DB-Schema, Tick Engine, HTTP-Server) |
| 2026-03-13 | AP1 Galaxiengenerator vollständig implementiert (Dichte, Sterne, Nebel, FTLW-Grid, 50.001 Sterne in DB) |
| 2026-03-13 | God-Mode-Viewer (React/Three.js) lauffähig – Sterne, Nebel, Filter, Inspektor, Bloom/CA |
| 2026-03-14 | AP2 vollständig spezifiziert: physikalisches Atmosphärenmodell, 5 Biochemie-Archetypen, 24 Ressourcengruppen, Spektraltyp-Matrix, Statistik-Output |
| 2026-03-14 | biochemistry_archetypes_v1.0.yaml erstellt mit 17 Primärquellen (HITRAN, Pierrehumbert, Pavlov u.a.) |
| 2026-03-14 | US-001 aufgenommen: Biochemie-Wahl für Alien-Spezies auf wissenschaftlicher Grundlage |
| 2026-03-14 | game-params v1.1: arm_winding 2.0 → 0.35 (BL-01), alle Referenzen aktualisiert |
| 2026-03-14 | BL-08 implementiert: Generator-Admin-Tool (Morphologie-Picker, Param-Editor, Job-Polling) |
| 2026-03-14 | ADR-011 entschieden: WebGPU + TSL Volumetrisches Raymarching (BL-06a/b) |
| 2026-03-14 | game-params v1.2: rendering-Sektion ergänzt (nebula_raymarch_steps u.a. für ADR-011) |
| 2026-03-16 | Makefile: .env auto-load + `make dev` Target |
| 2026-03-16 | dev.sh: One-Command Dev-Stack (DB → Backend → Frontend → Browser) |

---

## Nächste Schritte (priorisiert)

### User Stories

| ID | Story | Status | Abhängigkeit |
|---|---|---|---|
| US-001 | Als Spieler möchte ich die Biochemie meiner Spezies auf wissenschaftlicher Grundlage auswählen können, damit meine strategischen Entscheidungen über Expansion und Kolonisierung an realen physikalischen Gesetzmäßigkeiten hängen. | 📋 spezifiziert | AP2 |

**US-001 Akzeptanzkriterien:**
- Spieler wählt bei Spielstart einen Biochemie-Archetyp (konfigurierbare Anzahl, Standard: 5)
- Jeder Archetyp zeigt: Metabolismus-Reaktion (Summenformel), Temp/Druck-Bereich, Primärvorkommen
- In-Game Enzyklopädie mit wissenschaftlicher Erklärung und Quellenangabe
- Planeten klassifiziert als: nativ bewohnbar / mit Habitat / Ressourcen-Kolonie / unbewohnbar (relativ zur eigenen Biochemie)
- `biomass_potential` in DB als JSONB pro Archetyp gespeichert (für Terraforming-Vorbereitung)
- Biochemie-Liste YAML-konfigurierbar; neue Archetypen ohne Code-Änderung ergänzbar

### Offen: Design
- [ ] **AP7 – Tech Tree erweitern** (Elektronische Kampfführung, Akademien)

### In Arbeit: Implementierung
- [x] **AP3 – Server-Core Skeleton** ✅
- [x] **AP1 – Galaxiengenerator** ✅ (50k Sterne, 65 Nebel, FTLW-Grid)
- [x] **God-Mode-Viewer** ✅ (läuft auf localhost:5174)
  - Go-Projektstruktur (cmd/galaxy-gen, cmd/server, internal/*)
  - Docker Compose (PostgreSQL 16, Redis 7) mit Health-Checks
  - Config-Loading (game-params YAML → Go-Structs)
  - DB-Migrationen (001_initial: galaxies, stars, ftlw_chunks, planets, moons, …)
  - pgx/v5 Connection Pool mit Retry-Logik
  - golang-migrate Runner (läuft beim Serverstart automatisch)
  - chi HTTP-Server mit graceful shutdown + /health-Endpoint
  - Tick Engine (konfigurierbare Tick-Länge, Handler-Registry)
  - galaxy-gen CLI Skeleton

- [ ] **AP3 – Ausstehend**
  - Event Queue (Redis Sorted Set)
  - Action Queue Handler
  - WebSocket Hub
  - Auth (JWT)

- [ ] **AP2 – Planetensystem-Generator** _(Eager-Generierung nach galaxy-gen, nicht JIT)_
  - Frostgrenze berechnen (Hayashi 1981)
  - Akkretionsmodell + Titius-Bode-Korrektur; Poisson-Verteilung Planetenanzahl nach Spektraltyp
  - Asteroidengürtel (gestörte Akkretion + L4/L5 Trojaner)
  - Mond-Generierung (Gasriesen: 2–5 Monde; Kollisionsmonde: 10% Gesteinsplaneten)
  - **Physikalisches Atmosphärenmodell** (Druck, Komposition, Treibhauseffekt mit Quellen)
  - **Biochemie-Archetypen** (konfigurierbar via `biochemistry_archetypes_v1.0.yaml`, US-001)
  - `biomass_potential` als JSONB pro Archetyp
  - Temperatur (Stefan-Boltzmann + Treibhauseffekt + CIA für H2)
  - Dichte, Oberflächengravitation, Achsneigung, Rotationsperiode, Ringsystem
  - Ressourcen-Deposits nach Spektraltyp-Matrix (24 Gruppen)
  - Nutzfläche berechnen (Gravitation × Temperatur × Druck)
  - Statistik-Output (Archetyp-Verteilung, Balancing-Check, Ressourcen-Histogramm)

- [ ] **AP4 – Wirtschaftssystem**
- [ ] **AP5 – FTL-Navigation & Flotten**
- [ ] **AP6 – Schiffsdesign, Kampf, Frontend**

---

## Offene Entscheidungen (TBD)

| Thema | Priorität |
|---|---|
| Produktions-Cloud: AWS vs. GCP | Mittel (erst bei Deployment relevant) |
| 3D-Rendering-Library (Three.js / Babylon.js) | Mittel (vor AP6-Frontend) |
| Frontend-Zustandsmanagement | Mittel (vor AP6-Frontend) |
| FTLW Cut-off-Wert (% Basiswert) | Niedrig (konfigurierbar) |
| k-Faktor FTLW-Formel | Niedrig (Balancing-Parameter) |
| Default Tick-Längen pro Spielmodus | Niedrig (konfigurierbar) |
| Fiktive Tier-5-Ressourcen – Namen & Eigenschaften | Niedrig (vor AP4) |

---

## Backlog – God-Mode-Viewer / Galaxiengenerator

| ID | Thema | Beschreibung | Aufwand |
|---|---|---|---|
| BL-01 | arm_winding + scaleLength Fix | `arm_winding: 2.0 → 0.35`, `scaleLength: 7% → 18%`, Arm-Envelope erweitern | 30 Min |
| BL-02 | Hierarchisches Sampling | Region-first statt globales Rejection-Sampling → 50k Sterne in Sekunden statt Minuten | 1 Tag |
| BL-03 | Foto-Template Morphologie | Reales Galaxienfoto als 2D-Dichtekarte; benötigt BL-02. Katalog in `galaxy_morphology_catalog_v1.0.yaml` (8 Typen Sa–Irr). 7 Bilder noch herunterzuladen (Prioritätsliste im Katalog). | 1 Tag |
| BL-04 | Morphologie-Vorschau-Run | Status `preview_ready` → Frontend rendert sofort; benötigt BL-02 | 0,5 Tag |
| BL-08 | Generator-Frontend | Route `/generate`: Morphologie-Auswahl aus Katalog, Parameter, Start-Button → POST `/api/generate` | 1 Tag | ✅ |
| BL-09 | Morphologie-Dichte-Integration | Gewählte Morphologie-ID (aus Admin-Tool) als 2D-Dichtekarte in galaxy.Generator einbinden; ersetzt analytische Basis-Dichte. Benötigt BL-02 (hierarchisches Sampling) + BL-03 (Foto-Template). Vorerst: morphology_id wird nur in DB gespeichert. | 2 Tage |
| BL-10 | Adaptiver Octree (FTLW-Grid) | Ersetzt Flat-Voxelgrid (500 ly) durch sternzahlbegrenzten Octree mit dualem Subdivisions-Kriterium (stellar + Dichte-Gradient). DB-Schema: `ftlw_octree` + `ftlw_octree_adjacency`. FTLW kumulativ (inkl. Fernbeiträge). A* auf Adjazenzliste. Benötigt BL-02 + BL-03 (Simplex Noise). ~100k Blattknoten bei 50k Sternen. ADR-010. | 3 Tage |
| BL-05 | 500k Sterne | Binary-Transfer (Float32Array statt JSON) + BL-02 | 1,5 Tage |
| BL-06a | WebGPURenderer-Migration | God-Mode-Viewer auf `WebGPURenderer` + TSL umstellen. Akzeptanz: Sterne/Filter/Inspektor/Bloom visuell identisch, WebGL-Fallback verifiziert. Baseline für BL-06b. ADR-011. | 1 Tag |
| BL-06b | Volumetrisches TSL Raymarching | FBM-Funktion in TSL (6 Oktaven, Seed = Nebel-Seed), Raymarching-Pass, LOD-Übergang Sprites→Volumetric (≥20% Viewport), Nebeltyp-Visualisierung (H-II/SNR/Globular), `nebula_raymarch_steps` aus game-params (Default 64). Benötigt BL-06a + BL-03. ADR-011. | 2,5 Tage |
| BL-07 | Mehrfachsternsysteme | Generator: Begleitsterne zuweisen; Inspektor: Begleiter anzeigen | 3 h |

## Empfohlene Implementierungsreihenfolge

```
AP3 (Server-Core) ✅ Skeleton
  └─► AP1 (Galaxiengenerator) ✅
        └─► AP2 (Planetensystem-Generator, Eager nach galaxy-gen) ← NÄCHSTES AP
              └─► AP3 Remainder (Auth, WebSocket, Redis Event Queue)
                    └─► AP5 (FTL-Navigation & Flotten)
                          └─► AP4 (Wirtschaftssystem)
                                └─► AP6 (Schiffsdesign & Kampf)
                                      └─► AP7 (KI)
```
