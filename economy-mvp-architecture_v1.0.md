# Galaxis — Economy-MVP Architektur v1.0

**Status:** Design finalisiert · Alle Entscheidungen getroffen (D1–D8) · Implementation freigabepflichtig
**Abhängigkeiten:** `production-mechanics_v1.0.md`, `game-params_v1.6.yaml`, `recipes_v1.0.yaml`
**Datum:** 2026-03-22

---

## Finalisierte Entscheidungen

| # | Frage | Entscheidung | Begründung |
|---|---|---|---|
| E1 | Tick-Steuerung | **Hybrid:** Echtzeit-Timer + `POST /admin/tick/advance` | MVP nutzt manuell; Echtzeit-Modus braucht keine Codeänderung |
| E2 | Player-Modell | **player_id als UUID, hardcoded, kein Auth** | `PLAYER_ZERO_ID` als Konstante; Keycloak-Integration kommt in AP3 |
| E3 | Frontend-Einstieg | **Neue Route `/economy/:starId`** | Schnellster Weg; System-View-Embed folgt später |
| E4 | UI-Stil | **Cards + Tabellen** | Cards für Anlagen/Deposits, Tabelle für Lager; React Flow als 2. Iteration |
| E5 | API-Muster | **REST + SSE** | Etabliertes Muster im Projekt; kein WebSocket für MVP |
| F1 | Rezepte | **`recipes_v1.0.yaml`** | Separates YAML, eigener Versionspfad, nie in DB |
| F2 | Tick-Log | **Rolling, letzte 100 Ticks** | Kein unbegrenztes DB-Wachstum |
| F3 | Deposit-Init | **Lazy — beim ersten Survey** | Survey-Qualität bestimmt welche Info der Spieler erhält |
| D1 | Deposit-Formel | `total = base × quality`, `max_rate = base_rate × (0.5 + q×0.5)`, `slots = max(1, round(base_slots × q))` | Qualität skaliert alle drei Dimensionen |
| D2 | Survey-Voraussetzung | Mine baut ohne Survey → 422. Infrastruktur (Aufzug, Schmelze) ohne Survey erlaubt. MVP: kein Schiff nötig. | Full mechanic Post-MVP (AP5) |
| D3 | Kolonieschiff-Survey | Automatisch Runde 1 (quality=0.50) bei Landung. 3-Runden-Pfad zu 1.00. | Besseres Equipment als normales Survey-Team |
| D4 | Deposits pro Ressource | Ein Aggregat pro Ressource pro Planet (MVP) | Mehrere Körper Post-MVP |
| D5 | Aufzug-Allokation | Prioritätswarteschlange (Spieler), Fallback proportional | Aktives Management bringt Vorteil |
| D6 | Assembler-Queue | Queue-Tiefe = `max_action_queue_depth` (50) | Offline-freundlich |
| D7 | Survey-Staleness | Snapshot update: nur bei eigenem Mining (jeden Tick) oder neuem Survey. Anderer Spieler → `stale`-Flag. | Kein doppelter Datensatz |
| D8 | Deposit-Warnungen | 20% gelb (Badge + Log), 5% rot (Badge + Log + SSE-Push) | Zwei Schwellen für abgestufte Dringlichkeit |

---

## Deposit-Survey-Modell (F3 Detail)

Deposits werden beim Planeten-Survey in `planet_deposits.state` geschrieben.
Die Survey-Qualität (`survey_quality` 0.0–1.0) bestimmt die Informationstiefe:

```
quality < 0.30  →  "Eisenerz vorhanden"          (kein Mengenwert, kein Rate-Wert)
quality < 0.60  →  "Eisenerz: ~40.000–60.000 E"  (grober Mengenbereich)
quality < 0.90  →  "Eisenerz: 49.850 E, max 30/Tick" (Menge + Rate bekannt)
quality ≥ 0.90  →  vollständige Daten inkl. Slot-Count
```

`planet_deposits.state` enthält immer die echten Werte.
Der API-Response filtert die Felder anhand der survey_quality des Spielers.
→ Kein doppelter Datensatz, kein "Fog-of-War"-Duplikat in der DB.

---

## Datenarchitektur

### Philosophie

```
YAML (versioniert, in-memory bei Start)     PostgreSQL (veränderlicher Spielzustand)
────────────────────────────────────        ────────────────────────────────────────
recipes_v1.0.yaml    → RecipeRegistry       planet_deposits   JSONB state
game-params_v1.5.yaml → GoodRegistry        facilities        4 Spalten + JSONB config
                        FacilityRegistry    system_storage    JSONB contents
                        DepositRegistry     production_log    JSONB events (rolling)
```

Änderungen an Rezepten, Gütern, Effizienzwerten → YAML-Commit, keine DB-Migration.

### Migration 006 — Economy (5 Tabellen)

```sql
-- Deposit-Zustand pro Planet (initialisiert beim Survey)
-- state: { "iron_ore": { "remaining": 49850, "max_rate": 30, "slots": 3,
--           "survey_quality": 0.85 }, ... }
CREATE TABLE planet_deposits (
  planet_id  UUID PRIMARY KEY REFERENCES planets(id),
  state      JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Produktionsanlagen
-- config: { "level": 1, "recipe_id": "titansteel", "ticks_remaining": 2,
--           "efficiency_acc": 0.72, "deposit_id": "iron_ore" }
CREATE TABLE facilities (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id     UUID NOT NULL,                          -- hardcoded MVP: PLAYER_ZERO_ID
  star_id       UUID NOT NULL REFERENCES stars(id),
  planet_id     UUID REFERENCES planets(id),            -- NULL = orbital
  facility_type TEXT NOT NULL,
  status        TEXT NOT NULL DEFAULT 'idle',
  config        JSONB NOT NULL DEFAULT '{}',
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_facilities_star   ON facilities(star_id);
CREATE INDEX idx_facilities_player ON facilities(player_id);
CREATE INDEX idx_facilities_status ON facilities(status);

-- Systemlager pro Spieler
-- contents: { "iron_ore": 47.5, "steel": 12.0, "semiconductor_wafer": 3.2 }
-- Sensitivitätsklasse wird aus GoodRegistry ermittelt — nicht in DB
CREATE TABLE system_storage (
  player_id  UUID NOT NULL,
  star_id    UUID NOT NULL REFERENCES stars(id),
  contents   JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (player_id, star_id)
);

-- Tick-Log, rolling (letzte 100 Ticks pro System)
-- events: [ { "type": "produced", "facility_id": "...", "good": "titansteel",
--             "qty": 4.0, "acc_before": 0.72, "acc_after": 0.12 }, ... ]
CREATE TABLE production_log (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  player_id  UUID NOT NULL,
  star_id    UUID NOT NULL REFERENCES stars(id),
  tick_n     BIGINT NOT NULL,
  events     JSONB NOT NULL DEFAULT '[]',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_production_log_star_tick ON production_log(star_id, tick_n DESC);
```

-- Spieler-Surveys: persistierter Wissensstand pro (Spieler, Planet)
-- Snapshot enthält die Deposit-Daten gefiltert nach quality zum Survey-Zeitpunkt.
-- Wahrheit liegt in planet_deposits.state — dieser Eintrag ist der "Stand vom letzten Scan".
--
-- snapshot: { "iron_ore": { "present": true, "remaining_approx": "40000-60000",
--              "remaining_exact": 49850, "max_rate": 30, "slots": 3 } }
-- Welche Felder befüllt sind hängt von quality ab (s. Survey-Qualitätsmodell).
CREATE TABLE player_surveys (
  player_id   UUID NOT NULL,
  planet_id   UUID NOT NULL REFERENCES planets(id),
  surveyed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  tick_n      BIGINT NOT NULL,     -- Spieltick des Surveys (für Staleness-Anzeige)
  quality     FLOAT NOT NULL,      -- 0.0–1.0, bestimmt Informationstiefe
  snapshot    JSONB NOT NULL,      -- gefilterter Deposit-Stand zum Survey-Zeitpunkt
  PRIMARY KEY (player_id, planet_id)
  -- ON CONFLICT (player_id, planet_id) DO UPDATE — neuerer Survey überschreibt
);
CREATE INDEX idx_player_surveys_player ON player_surveys(player_id);

**Orbital Slots:** `used = COUNT(*) FROM facilities WHERE star_id=X AND planet_id IS NULL`
`total_slots` kommt aus game-params → kein eigenes Feld, keine Tabelle.

---

## Go-Paketstruktur

```
internal/
  economy/
    registry.go       — lädt recipes.yaml + game-params → GoodRegistry, RecipeRegistry, ...
    production.go     — tick-Handler: zieht Inputs, berechnet Output, schreibt Storage
    deposit.go        — Deposit-Zustand, Abbau-Logik, Erschöpfungs-Events
    survey.go         — Survey ausführen: planet_deposits lesen, filtern, player_surveys schreiben
    storage.go        — JSONB-Lesen/Schreiben, Kapazitätsprüfung
    log.go            — Rolling-Log-Schreiben, alte Ticks löschen

  api/
    economy_handlers.go  — REST-Handler für Economy-Routen
```

### Tick-Engine-Erweiterung

```go
// Bestehende Engine bekommt eine Advance-Methode:
func (e *Engine) Advance(ctx context.Context) {
    e.tickN.Add(1)
    e.fireTick(ctx, e.tickN.Load(), time.Now())
}
// → POST /admin/tick/advance ruft engine.Advance(ctx) auf
```

---

## API-Routen

```
GET  /api/v1/economy/system/:starId
     → vollständiger State: deposits (gefiltert nach survey_quality),
       facilities, storage, orbital_slots_used/total, last_tick_n

POST /api/v1/economy/system/:starId/build
     Body: { "facility_type": "smelter", "planet_id": "uuid|null", "level": 1 }
     → prüft Materialkosten, Deposit-Slots/Orbital-Slots, schreibt facilities

POST /api/v1/economy/system/:starId/facilities/:facilityId/recipe
     Body: { "recipe_id": "titansteel" }
     → weist Rezept zu, setzt status = "running"

GET  /api/v1/economy/system/:starId/log?limit=20
     → letzte N Tick-Events aus production_log

GET  /api/v1/economy/system/:starId/events
     → SSE-Stream: pushed nach jedem Tick ein Event
     data: { "tick": 5, "storage_delta": {...}, "facility_updates": [...],
             "deposit_updates": [...], "events": ["Schmelze: +4 E Titanstahl"] }

POST /api/v1/economy/planets/:planetId/survey
     Body: { "quality": 0.85 }   ← im MVP fest; später aus Survey-Schiff-Stats berechnet
     → liest planet_deposits.state, filtert nach quality, schreibt player_surveys
     → Response: gefilterter Snapshot (was der Spieler jetzt weiß)

GET  /api/v1/economy/planets/:planetId/survey
     → liest player_surveys WHERE (player_id, planet_id)
     → Response: { snapshot, quality, surveyed_at, tick_n, stale: bool }
     → stale = true wenn planet_deposits.updated_at > player_surveys.surveyed_at
       (d.h. ein anderer Spieler hat zwischenzeitlich Deposits verändert)

GET  /api/v1/economy/system/:starId/surveys
     → alle player_surveys des Spielers für alle Planeten in diesem System
     → für Systemübersicht: welche Planeten schon gescannt, wie alt ist der Scan

POST /api/v1/admin/tick/advance
     → löst manuellen Tick aus (MVP-only, später hinter Admin-Auth)
```

---

## Frontend-Struktur (`/economy/:starId`)

```
EconomyPage
  ├── TickControls         — aktueller Tick, [Advance]-Button, SSE-Status-Dot
  ├── OrbitalSlotsBar      — X / 8 Slots belegt
  ├── DepositsSection
  │   └── DepositCard[]    — Ressource, verbleibend (aus Survey-Snapshot), max_rate,
  │                           belegte Mine-Slots, "Scan veraltet"-Badge wenn stale=true
  ├── FacilitiesSection
  │   └── FacilityCard[]   — Typ, Level, Status, Rezept, η, ETA
  ├── StorageTable         — good | qty | Sensitivitätsklasse | Kapazität
  ├── BuildPanel           — Dropdown Anlage + Planet/Orbital, [Bauen]-Button
  └── EventLog             — letzten 20 Tick-Events, SSE-getrieben
```

SSE-Hook aktualisiert Store → alle Komponenten re-rendern reaktiv.
Kein Polling, kein manuelles Refresh.

---

## Produktions-Tick-Algorithmus (Pseudocode)

```
FOR EACH facility WHERE status = 'running' AND star_id = X AND player_id = Y:

  recipe = RecipeRegistry[config.recipe_id]
  η      = FacilityRegistry[facility_type].efficiency[config.level - 1]

  IF config.ticks_remaining > 1:
    config.ticks_remaining -= 1
    CONTINUE  -- Batch läuft noch

  -- Batch abgeschlossen: Inputs prüfen
  FOR EACH input IN recipe.inputs:
    IF storage[input.good_id] < input.qty:
      SET status = 'paused_input', BREAK
      LOG event: { type: "paused_input", missing: input.good_id }

  -- Inputs abziehen
  FOR EACH input IN recipe.inputs:
    storage[input.good_id] -= input.qty
    IF input.good_id = mine_resource AND deposit:
      deposit.remaining -= input.qty
      IF deposit.remaining <= 0: LOG event: { type: "deposit_depleted" }

  -- Output berechnen (Float-Akkumulator)
  FOR EACH output IN recipe.outputs:
    config.efficiency_acc += output.qty * η
    produced = floor(config.efficiency_acc)
    config.efficiency_acc -= produced

    IF storage_capacity_ok:
      storage[output.good_id] += produced
      LOG event: { type: "produced", good: output.good_id, qty: produced }
    ELSE:
      SET status = 'paused_output'
      LOG event: { type: "paused_output", good: output.good_id }

  -- Nächsten Batch starten
  config.ticks_remaining = recipe.ticks

-- Rolling Log: DELETE WHERE tick_n < (current - 100)
```

---

## Player-Zero Konstante (MVP)

```go
// internal/economy/player.go
const PlayerZeroID = "00000000-0000-0000-0000-000000000001"

// Alle Economy-Handler lesen player_id aus Header X-Player-ID.
// Wenn absent → PlayerZeroID (MVP-Fallback).
// AP3: Header wird durch Keycloak-JWT ersetzt, Konstante entfällt.
```

---

## Implementierungsreihenfolge (nach Freigabe)

```
1. Migration 006 schreiben + testen
2. internal/economy/registry.go — YAML-Loader
3. internal/economy/deposit.go  — Deposit-Zustand + Abbau
3b. internal/economy/survey.go  — Survey-Logik + Snapshot-Filterung
4. internal/economy/storage.go  — JSONB-Helpers
5. internal/economy/production.go — Tick-Handler, bei Engine registrieren
6. internal/economy/log.go      — Rolling-Log
7. api/economy_handlers.go      — alle 6 Routen
8. tick/engine.go               — Advance()-Methode ergänzen
9. Frontend: EconomyPage + Komponenten
10. E2E-Test: Kolonisierung → Survey → Build → Tick × 30 → Zielzustand prüfen
```

---

*Erstellt: 2026-03-22 · Alle Designentscheidungen fixiert · Freigabe zur Implementation ausstehend*
