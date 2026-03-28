# Architektur – Galaxis v1.0

**Datum:** 2026-03-14
**GDD-Referenz:** v1.24

---

## Überblick

Galaxis ist ein Hard-Sci-Fi Grand Strategy MMO mit strikter Client-Server-Trennung. Der Server ist die alleinige Autorität über den Spielzustand. Clients (menschliche Spieler und KI-Headless-Clients) kommunizieren ausschließlich über eine definierte API.

```
┌─────────────────────────────────────────────────────────────────┐
│                         CLIENTS                                 │
│  ┌──────────────────────┐   ┌───────────────────────────────┐  │
│  │  Web-Frontend        │   │  KI-Headless-Client (Go)      │  │
│  │  React + Vite + TS   │   │  Identische API-Nutzung       │  │
│  └──────────┬───────────┘   └────────────────┬──────────────┘  │
└─────────────┼───────────────────────────────-┼─────────────────┘
              │ WebSocket / REST                │
┌─────────────▼─────────────────────────────────▼─────────────────┐
│                       API GATEWAY (Go)                           │
│  Auth · Rate Limiting · Request Routing · WebSocket Hub          │
└──────────┬───────────────────────────────────────────────────────┘
           │
┌──────────▼──────────────────────────────────────────────────────┐
│                    GAME SERVER (Go)                              │
│                                                                  │
│  ┌─────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ Tick Engine │  │ Event Queue  │  │ Action Queue Handler │   │
│  │ (Strategie) │  │ (Future Evts)│  │ (Befehlsverarbeitung)│   │
│  └──────┬──────┘  └──────┬───────┘  └──────────────────────┘   │
│         │                │                                       │
│  ┌──────▼──────────────────────────────────────────────────┐    │
│  │              Domain Services                            │    │
│  │  Galaxy · Economy · Fleet · Combat-Dispatcher           │    │
│  └─────────────────────────────────────────────────────────┘    │
└──────────┬──────────────────────────────────────────────────────┘
           │ Spawn on demand
┌──────────▼──────────────────────────────────────────────────────┐
│              COMBAT SERVER PODS (Go, containerized)             │
│  Eigener schneller Kampftick · Orbital-Solver (Patched Conics)  │
│  Ergebnis → Game Server nach Abschluss                          │
└─────────────────────────────────────────────────────────────────┘
           │
┌──────────▼──────────────────────────────────────────────────────┐
│                     PERSISTENZSCHICHT                            │
│  ┌──────────────────────────┐   ┌────────────────────────────┐  │
│  │ PostgreSQL               │   │ Redis                      │  │
│  │ Spielzustand, Entitäten  │   │ Event Queue, Tick-State,   │  │
│  │ Ressourcen, Galaxie-Meta │   │ Session-Cache, Pub/Sub     │  │
│  └──────────────────────────┘   └────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

---

## Komponenten

### API Gateway
- Authentifizierung & Autorisierung (JWT)
- WebSocket-Hub für Echtzeit-Updates an Clients
- REST-Endpunkte für Befehle (Action Queue)
- Rate Limiting

### Game Server
Kern der Spiellogik. Läuft kontinuierlich.

#### Tick Engine (Strategietick)
- Tick-Länge konfigurierbar per Server-Instanz (z.B. 15 Min, 1 Std, 6 Std)
- Pro Tick: Ressourcenproduktion, Flottenfortbewegung, Bau-Fortschritt
- Am Ende jedes Ticks: Auflösung nicht-besetzter Gefechte (automatische Berechnung)

#### Event Queue
- Future-Events mit Zeitstempel: "Flotte X erreicht System Y in Tick 540"
- Events feuern nur bei Erreichen des Zeitstempels → minimale CPU-Last
- Persistiert in Redis (schnell) + PostgreSQL (durabel)

#### Action Queue Handler
- Nimmt Spielerbefehle entgegen, validiert sie, legt sie mit Ausführungsdauer in die Queue
- KI-Headless-Client nutzt dieselbe Queue

#### Combat-Dispatcher
- Erkennt, wenn zwei Flotten im gleichen System aufeinandertreffen
- Startet Combat Server Pod (containerized)
- Öffnet Opt-In-Zeitfenster (konfigurierbar, z.B. 24h) für menschliche Spieler
- Nach Ablauf: Ergebnis des Pods empfangen, Spielzustand aktualisieren

### Combat Server Pods
- Werden on-demand gespawnt (Kubernetes/Docker)
- Laufen mit eigenem schnellen Kampftick (Sekunden bis Minuten)
- Deterministischer Orbital-Solver (Patched Conics) für Geschossbahnen
- Nach Gefechtende: Zusammenfassung → Game Server, Pod terminiert

### Persistenzschicht

#### PostgreSQL
- Spielzustand: Spieler, Fraktionen, Flotten, Planeten, Sternensysteme
- Ressourcenkonten, Produktionsketten, Gebäude
- Galaxie-Metadaten (Sternpositionen, Typen, FTLW-Voxelgrid-Snapshots)

#### Redis
- Event Queue (zeitbasierte Prioritätswarteschlange)
- Tick-Koordination (Distributed Lock)
- Session-Cache, WebSocket-Routing
- Pub/Sub für Live-Updates an verbundene Clients

---

## Schlüsselprinzipien

### Planetengenerierung – Zwei-Modus-Strategie (ADR-009)

**Entwicklung / Balancing (Eager):** Der `galaxy-gen`-CLI generiert alle Planetensysteme
unmittelbar nach der Sterngenerierung (`--eager-planets` Flag). Ermöglicht vollständige
Statistiken (Archetyp-Verteilung, Ressourcen-Histogramme) für Balancing-Validierung.

**Produktion (Just-in-Time):** Der Server speichert nur den deterministischen Seed pro Stern.
Planetendaten werden erst generiert und persistiert, wenn ein Spieler das System betritt
oder scannt. Spart Speicher: nur besuchte Systeme existieren in der DB.

Beide Modi nutzen identische Generator-Logik (`internal/planet`). Das Flag
`planets_generated` in der `stars`-Tabelle steuert, welche Systeme bereits berechnet wurden.

### Biochemie-Konfiguration
Atmosphären-Archetypen für Alien-Spezies sind in `biochemistry_archetypes_v1.0.yaml`
definiert. Der Planetengenerator lädt diese Datei dynamisch – neue Archetypen können
ohne Code-Änderung ergänzt werden. Jeder Archetyp enthält physikalische Parameter
mit Primärquellenangaben (HITRAN, Pierrehumbert, Pavlov u.a.).

### Autoritativer Server
Clients senden nur Befehle (Intent), nie Zustand. Alle Berechnungen (Wirtschaft, Bewegung, Kampf) laufen ausschließlich auf dem Server. KI-Clients sind Headless-Prozesse, die dieselbe API nutzen wie Browser-Clients.

### Konfigurierbare Instanzen
Tick-Länge, maximale Spielerzahl und Galaxiegröße sind pro Server-Instanz konfigurierbar. Dadurch unterstützt dieselbe Codebasis Langzeitpartien und Wochenendpartien.

---

## Deployment (Entwicklung → Produktion)

| Phase | Infrastruktur |
|---|---|
| Entwicklung | Eigengehosteter Server, Docker Compose |
| Produktion | AWS oder GCP, containerisiert (Kubernetes) |
| Combat Pods | Kubernetes-Pods, on-demand skalierend |

---

## Frontend

- **Stack:** React + Vite + TypeScript
- **Kommunikation:** WebSocket (Echtzeit-Updates) + REST (Befehle)
- **Ansichten:** Galaktisches Holo-Deck → Sternensystem → Planet → CIC (Gefecht)
- **Prinzip:** Progressive Disclosure – Details erst bei Bedarf einblenden
