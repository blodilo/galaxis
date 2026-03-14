# Tech Decisions – Galaxis v1.0

**Datum:** 2026-03-14

---

## ADR-001: Backend-Sprache – Go

**Status:** Entschieden

**Kontext:** Der Game Server muss eine Tick-Engine, Event Queue, Pathfinding durch ein 3D-Voxelgrid und bis zu 100 gleichzeitige Spieler plus KI-Clients performant bedienen. Combat Server Pods müssen on-demand gestartet werden.

**Entscheidung:** Go

**Begründung:**
- Exzellente Performance für nebenläufige, langlebige Server-Prozesse (Goroutines, Channels)
- Geringe Latenz durch kompilierte Binaries ohne Runtime-Overhead
- Einfaches Deployment als einzelne Binary oder Container
- Gute Ökosystem-Unterstützung für gRPC, WebSocket, PostgreSQL, Redis
- KI-Headless-Client kann als separate Go-Binary aus demselben Codebase kompiliert werden

**Alternativen verworfen:**
- Python: Zu langsam für Tick-Engine mit vielen gleichzeitigen Events; GIL problematisch
- Rust: Höhere Einstiegshürde, längere Entwicklungszeit
- Node.js: Kein starker Typ-Safety-Vorteil gegenüber Go für diese Domäne

---

## ADR-002: Datenbank – Hybrid PostgreSQL + Redis

**Status:** Entschieden

**Kontext:** Das Spiel hat sehr unterschiedliche Datenzugriffsmuster: dauerhafte Entitätsdaten (Spieler, Planeten, Flotten), zeitbasierte Event-Queues und schnelle Tick-Koordination.

**Entscheidung:** PostgreSQL als primäre Datenbank + Redis für Echtzeit-Schicht

**PostgreSQL für:**
- Spielzustand (Spieler, Fraktionen, Flotten, Planeten, Sternensysteme, Ressourcen)
- Produktionsketten, Gebäude, Technologien
- Galaxie-Metadaten und FTLW-Voxelgrid (serialisiert)
- ACID-Transaktionen bei Tick-Auflösung

**Redis für:**
- Event Queue (Sorted Set nach Tick-Zeitstempel)
- Distributed Lock für Tick-Synchronisation
- WebSocket Session-Routing
- Pub/Sub für Live-Updates an Clients
- Hot-Cache für häufig gelesene, selten geänderte Daten (Sterndaten)

---

## ADR-003: Hosting-Strategie

**Status:** Entschieden

**Entwicklung:** Eigengehosteter Server, Docker Compose (Game Server + PostgreSQL + Redis)

**Produktion:** AWS oder GCP (Entscheidung noch ausstehend)
- Combat Server Pods: Kubernetes, on-demand skalierend
- Game Server: Stateful, mindestens 1 Instanz pro Spielwelt
- Datenbanken: Managed Services (RDS / Cloud SQL + ElastiCache / Memorystore)

---

## ADR-004: Frontend – React + Vite + TypeScript

**Status:** Entschieden (per Projekt-CLAUDE.md)

**Kontext:** Nahtlos zoombare 3D-Karte (Galaxie → System → Planet → CIC).

**Entscheidung:** React + Vite + TypeScript

**Offene Fragen für spätere ADRs:**
- 3D-Rendering-Bibliothek: Three.js / React Three Fiber vs. WebGL direkt vs. Babylon.js
- Zustandsmanagement: Zustand / Redux / Jotai

---

## ADR-005: Tick-Architektur – Zwei-Ebenen-System

**Status:** Entschieden

**Kontext:** Strategische Spielhandlungen (Bau, Bewegung, Wirtschaft) und taktische Gefechte haben grundlegend unterschiedliche Zeitanforderungen.

**Entscheidung:** Zwei entkoppelte Tick-Typen

| Tick-Typ | Frequenz | Zuständigkeit |
|---|---|---|
| Strategietick | Konfigurierbar pro Instanz (z.B. 15 Min – 6 Std) | Wirtschaft, Flottenfortbewegung, Bau, automatische Gefechtslösung |
| Kampftick | Sekunden bis Minuten (fest, im Combat Pod) | Orbital-Solver, Waffenfeuern, Schadenberechnung |

**Gefechtserzeugung:**
- Kollision zweier Flotten → Combat-Dispatcher öffnet Opt-In-Zeitfenster
- Spieler joinen → Combat Pod wird gespawnt, schneller Kampftick
- Kein Join → automatische Auflösung am Ende des nächsten Strategieticks

---

## ADR-006: Ressourcen – Reale Basis + optionale Fiktiv-Elemente

**Status:** Entschieden · aktualisiert 2026-03-14

**Entscheidung:** Basis-Ressourcen ausschließlich aus realen Elementen/Verbindungen,
gruppiert in 24 Ressourcengruppen. Fiktive Ressourcen nur für spielmechanisch
einzigartige Dinge ohne reales Äquivalent (z.B. FTL-Stabilisatoren, max. 2–3).

**Ressourcenkatalog – 24 Gruppen (aus GDD v1.24 destilliert):**

| ID | Gruppe | Tier | Primäre Spektralquellen |
|---|---|---|---|
| `iron` | Eisen/Stahl | 1 | F/G/K, M |
| `aluminum` | Aluminium/Magnesium | 1 | F/G/K, M |
| `copper` | Kupfer | 1 | F/G/K |
| `silicon` | Silizium | 1 | F/G/K, M |
| `carbon` | Kohlenstoff/Graphen | 1 | R/S-Sterne (s-Prozess) |
| `zinc_tin` | Zink/Zinn | 2 | F/G/K |
| `manganese_vanadium` | Mangan/Vanadium | 2 | F/G/K |
| `chromium_molybdenum` | Chrom/Molybdän | 2 | F/G/K, A |
| `sulfur_chlorine` | Schwefel/Chlor | 2 | F/G/K, vulkanisch |
| `calcium_sodium` | Calcium/Natrium | 2 | F/G/K |
| `oxygen` | Sauerstoff | 3 | F/G/K (flüssig/gebunden) |
| `nitrogen` | Stickstoff | 3 | F/G/K, Eismonde |
| `water_ice` | Wasser/Eis | 3 | M, Eiswelten, Gasriesenmonde |
| `phosphorus` | Phosphor | 3 | F/G/K (Biomasse-Systeme) |
| `deuterium` | Deuterium/Tritium | 4 | Gasriesen, Globular Cluster |
| `helium3` | Helium-3 | 4 | Gasriesen, Globular Cluster |
| `lithium_cobalt` | Lithium/Kobalt | 4 | F/G/K |
| `uranium_thorium` | Uran/Thorium | 4 | SNR, Pulsar, StellarBH |
| `titanium_beryllium` | Titan/Beryllium | 5 | O/B/A (refraktär) |
| `tungsten` | Wolfram | 5 | O/B/A (refraktär) |
| `silver_gold` | Silber/Gold | 5 | SNR, Pulsar |
| `platinum_group` | Platingruppe (Pt/Pd/Ir/Os) | 5 | SNR, Pulsar, StellarBH |
| `rare_earth` | Seltene Erden (Nd/Y) | 5 | R/S-Sterne (s-Prozess) |
| `lead` | Blei (Strahlenschutz) | 5 | R/S-Sterne |

**Sonder:** Antimaterie (produziert, nicht abgebaut); Biomasse (archetyp-relativ, s. ADR-009)

**Wissenschaftliche Grundlage Spektraltyp-Verteilung:**
- r-Prozess (Supernovae/Neutronenstern-Kollisionen) → Platingruppe, U/Th, Ag/Au
- s-Prozess (AGB-Riesensterne) → Seltene Erden, Kohlenstoff, Blei
- Fraktionierung heiße Sterne → Refraktäre Metalle (Ti, W, Be)
- Quelle: GDD v1.24 §6; Burbidge et al. (1957), Rev. Mod. Phys. 29(4)

---

---

## ADR-008: Biochemie-Archetypen – YAML statt JSON

**Status:** Entschieden 2026-03-14

**Kontext:** Biochemie-Parameter (Treibhauswerte, Kompositions-Templates, Temp-Bereiche)
müssen menschenlesbar und kommentierbar sein. Neue Archetypen sollen ohne Code-Änderung
ergänzt werden können.

**Entscheidung:** Separate YAML-Datei `biochemistry_archetypes_v1.0.yaml`

**Begründung:**
- YAML erlaubt Kommentare (`#`) → wissenschaftliche Quellenangaben direkt im File
- Konsistent mit `game-params_v1.0.yaml` (gleicher Parser, gleiche Konvention)
- Besser lesbar als JSON für lange, kommentierte Konfigurationen
- Dynamisch einlesbar: Generator lädt alle `enabled: true`-Einträge; beliebig erweiterbar

**Alternativen verworfen:**
- JSON: Keine Kommentare möglich → Quellenangaben wären verloren
- Einbettung in game-params: Zu groß; inhaltlich eigenständige Domäne

---

## ADR-009: Planetengenerierung – Eager für Entwicklung, JIT für Produktion

**Status:** Entschieden 2026-03-14

**Kontext:** Die Architektur (ADR, architecture_v1.0.md) sieht JIT-Planetengenerierung vor.
Für die Balancing- und Entwicklungsphase ist das impraktisch: Planeten-Statistiken
(Archetyp-Verteilung, Ressourcen-Histogramme) können nur mit vollständig generierten
Daten überprüft werden.

**Entscheidung:** Zwei-Modus-Strategie

| Modus | Trigger | Verwendung |
|---|---|---|
| Eager | `galaxy-gen` CLI (Dev/Balancing) | Alle Planeten sofort nach Sterngenerierung |
| JIT | Spielbetrieb (Produktion) | Planeten bei erstem Schiff-Scan generiert und persistiert |

**Technische Umsetzung:** `--eager-planets` Flag im `galaxy-gen` CLI.
`planets_generated`-Boolean in der `stars`-Tabelle steuert beide Modi.

**Begründung:**
- Balancing ohne vollständige Daten nicht möglich
- JIT spart DB-Speicher im Produktivbetrieb (nur besuchte Systeme persistiert)
- Beide Modi nutzen identische Generator-Logik → kein Divergenz-Risiko

---

## ADR-007: KI – Headless Client, gleiche Regeln

**Status:** Entschieden

**Entscheidung:** KI-Fraktionen sind Go-Prozesse, die dieselbe Server-API nutzen wie menschliche Spieler. Kein Cheating, keine versteckten Ressourcen.

**Implementierungsstrategie:** Erst nach Fertigstellung des vollständigen API-Contracts (AP3) implementieren.
