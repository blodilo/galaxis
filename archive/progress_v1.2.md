# Progress – Galaxis v1.2

**Datum:** 2026-03-21

---

## Aktueller Status

**Phase:** AP1 + AP2 implementiert und lauffähig · God-Mode-Viewer · Image-Based Generator (BL-11) · Systemansicht mit prozeduralen Shadern, Prominenzen, Layer-System, Mondsystem-Ansicht, InstancedMesh-Asteroiden · Vitest-Test-Infrastruktur eingerichtet

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
| 2026-03-21 | **BL-20: Prozedurale Stern-/Planeten-/Mondshader** – Hash-basiertes Value-Noise (GLSL ES 2.0-sicher), Limb-Darkening (Eddington), animierte Granulation (2 Skalen), Sonnenflecken, Fresnel-Korona, Prominenz-Kugel, Gas-/Eis-/Rocky-/Lava-Planeten, Krater-Monde |
| 2026-03-21 | **BL-21: Asteroidengürtel visuell** – InstancedMesh IcosahedronGeometry (flatShading), deterministischer LCG-Seed, Potenzgesetz-Größen, Dreieck-Höhenverteilung, PointLight für Beleuchtung, Staub-Ring-Shader |
| 2026-03-21 | **BL-24: Doppelklick Planet → Mondsystem-Ansicht** – MoonSystemScene mit Planetenshader, animierten Mondorbits, SmartOrbitControls |
| 2026-03-21 | **Visual Tuner** – Runtime-Tuning aller visuellen Parameter via Sidebar-Panel, localStorage-Persistenz |
| 2026-03-21 | **Layer-Steuerung** – Orbitalbahnen / Rotationsachse / Richtungspfeil per Checkbox ein-/ausblendbar |
| 2026-03-21 | **Selektion-Aura** – Rim-lit transparente Kugel statt Torus; Rotationsachse + 3/4-Bogen als eigener Layer |
| 2026-03-21 | **OrbitChevron** – `>` exakt auf Kepler-Ellipse (computeOrbitPos + dTheta), goldfarben |
| 2026-03-21 | **SmartOrbitControls** – maxDistance aus Szenenausdehnung, Auto-Zentrieren auf Ursprung bei Zoom-Limit |
| 2026-03-21 | **Vitest Test-Infrastruktur** – 25 Tests: uuidSeed, starColorTriad, calcPlanetVisR, computeOrbitPos, makeLcg |

---

## Nächste Schritte (priorisiert)

### User Stories

| ID | Story | Status | Abhängigkeit |
|---|---|---|---|
| US-001 | Als Spieler möchte ich die Biochemie meiner Spezies auf wissenschaftlicher Grundlage auswählen können. | 📋 spezifiziert | AP2 |

### In Arbeit / Offen: Implementierung

- [ ] **AP3 – Ausstehend:** Event Queue, Action Queue Handler, WebSocket Hub, Auth (JWT)
- [ ] **AP4 – Wirtschaftssystem**
- [ ] **AP5 – FTL-Navigation & Flotten**
- [ ] **AP6 – Schiffsdesign, Kampf, Frontend**

---

## Backlog – Planetensystem-Generator (AP2)

| ID | Thema | Beschreibung | Priorität | Aufwand | Status |
|---|---|---|---|---|---|
| BL-12 | Elliptische Orbits + Temperaturgrenzen | Kepler-Ellipsen mit e, ω, i; Perihel/Aphel; T_eq_min/max; Migration 004 | Hoch | 1 Tag | ✅ erledigt |
| BL-13 | Planetenrotation + Bahn-Spin-Kopplung | Tidale Verriegelung physikalisch, Obliquity-Schwankungen, Spin-Orbit-Resonanzen | Mittel | 0,5 Tage | 🔲 offen |
| BL-14 | Mehrfachsternsysteme | Binär/Trinär, S-Typ/P-Typ HZ, Hill-Sphären-Check, DB companion_stars | Mittel | 3 Tage | 🔲 offen |
| BL-15 | Planeten-UI: Master-Detail | SystemTree (Stern→Planeten→Monde), aufklappbar, bidirektionale Selektion | Hoch | 1,5 Tage | ✅ erledigt |
| BL-16 | Benennungssystem für Systeme und Planeten | SYS-XXXX Hash + custom_name, Planeten-ID römisch, Monde a/b/c | Mittel | 1 Tag | 🔲 offen |
| BL-17 | Spektralunterklassen | MK-System G2V/K5III, Unterklasse 0–9 + Leuchtkraftklasse, Teff-Interpolation | Mittel | 0,5 Tage | 🔲 offen |
| BL-18 | Relative Größen in Systemansicht (log. Skala) | calcPlanetVisR, orbital-gap clamp, Mond-Normalisierung | Hoch | 0,5 Tage | ✅ erledigt |
| BL-19 | Hover-Details für Himmelskörper | Tooltip-Overlay oder Sidebar-Update bei Hover, fixiert bei Klick | Mittel | 0,5 Tage | 🔲 offen |
| BL-20 | Texturen für Sterne und Planeten | Prozedurale Shader, Prominenz-Shell, animierte Granulation, Biochemie-Paletten | Niedrig | 2 Tage | ✅ erledigt |
| BL-21 | Asteroidengürtel (visuell) | InstancedMesh IcosahedronGeometry, deterministischer LCG, Potenzgesetz-Verteilung | Niedrig | 1 Tag | ✅ erledigt |
| BL-24 | Planet-Zoom: Doppelklick → Mondsystem-Ansicht | MoonSystemScene, SmartOrbitControls, Breadcrumb-Navigation | Mittel | 1 Tag | ✅ erledigt |
| BL-25 | Planetenringe (Saturn-artig) | Generator-Flag `has_rings` (bereits in DB). Frontend: RingGeometry + Textur-Noise, Neigung aus Inklination | Niedrig | 1 Tag | 🔲 offen |
| BL-27 | Monorepo-Migration (pnpm Workspaces) | `creaminds/graph-ui` extrahieren, pnpm Workspaces. Siehe ADR-012. | Niedrig | 0,5 Tage | 🔲 offen |
| BL-26 | Spiel-Wiki / Online-Dokumentation | Docusaurus (MIT), selbst gehostet, privat. Benötigt BL-27. Siehe ADR-012. | Niedrig | 1,5–2 Tage | 🔲 offen |

---

## Backlog – God-Mode-Viewer / Galaxiengenerator

| ID | Thema | Beschreibung | Aufwand | Status |
|---|---|---|---|---|
| BL-10 | Adaptiver Octree (FTLW-Grid) | Ersetzt Flat-Voxelgrid durch sternzahlbegrenzten Octree. DB: `ftlw_octree` + Adjazenzliste. A* | 3 Tage | 🔲 offen |
| BL-05 | 500k Sterne | Binary-Transfer (Float32Array) + hierarchisches Sampling | 1,5 Tage | 🔲 offen |
| BL-06a | WebGPURenderer-Migration | God-Mode-Viewer auf WebGPURenderer + TSL umstellen. ADR-011 | 1 Tag | 🔲 offen |
| BL-06b | Volumetrisches TSL Raymarching | FBM in TSL, Raymarching-Pass, LOD-Übergang Sprites→Volumetric. Benötigt BL-06a | 2,5 Tage | 🔲 offen |
| BL-22 | Z-Achse: Kinematische Alters-Schichtung | z-Spread gekoppelt an Spektralklasse. Exponentialverteilung. | 0,5 Tage | 🔲 offen |
| BL-23 | FBM Domain Warping | Organische Galaxie-Variation, 2-Oktaven-FBM, warpStrength ≈ 0.3–0.8 × Radius. | 1 Tag | 🔲 offen |

---

## Offene Entscheidungen (TBD)

| Thema | Priorität |
|---|---|
| Produktions-Cloud: AWS vs. GCP | Mittel |
| Frontend-Zustandsmanagement | Mittel |
| System-ID-Schema (BL-16) | Hoch (vor AP3-Auth) |
| Planeten-ID-Schema (BL-16) | Hoch (vor AP3-Auth) |
| FTLW Cut-off-Wert | Niedrig |
| Fiktive Tier-5-Ressourcen – Namen & Eigenschaften | Niedrig |

---

## Empfohlene Implementierungsreihenfolge

```
AP3 (Server-Core) ✅ Skeleton
  └─► AP1 (Galaxiengenerator) ✅
        └─► AP2 (Planetensystem-Generator) ✅ Basis + BL-12/15/18/20/21/24
              ├─► BL-13 (Rotation + Kopplung)       ← nächstes
              ├─► BL-16 (Benennungssystem)           ← nächstes (vor AP3-Auth)
              ├─► BL-17 (Spektralunterklassen)
              ├─► BL-14 (Mehrfachsternsysteme)
              ├─► BL-19 (Hover-Details)
              └─► BL-25 (Planetenringe)
        └─► AP3 Remainder (Auth, WebSocket, Redis)
              └─► AP5 (FTL-Navigation & Flotten)
                    └─► AP4 (Wirtschaftssystem)
                          └─► AP6 (Schiffsdesign & Kampf)
                                └─► AP7 (KI)
```

---

## DB-Migrationen (Übersicht)

| Nr | Datei | Inhalt |
|---|---|---|
| 001 | `001_initial.up.sql` | Grundschema: galaxies, stars, nebulae, planets (Basis), moons (Basis), ftlw_cells |
| 002 | `002_planet_model.up.sql` | Physikalisches Atmosphärenmodell: atm_pressure, atm_composition, greenhouse_delta_k, axial_tilt, rotation_period, has_rings, biochem_archetype, biomass_potential |
| 003 | `003_galaxy_status_steps.up.sql` | Galaxy-Status-Werte: morphology, spectral, objects, error |
| 004 | `004_orbital_mechanics.up.sql` | Kepler-Orbital-Parameter: eccentricity, arg_periapsis_deg, inclination_deg, perihelion_au, aphelion_au, temp_eq_min_k, temp_eq_max_k |
| 005 | `005_moon_orbits.up.sql` | Mondorbit-Abstände: orbit_distance_au (Hill-Sphäre-basiert) |
