# Projekt-Kontext

## Arbeitsweise
- Erstelle zuerst Konzept und Architektur, dokumentiere Entscheidungen
- Schlage Alternativen für Frameworks und Technologien vor und interviewe mich
- Starte nie mit der Implementation bevor ich die Freigabe gebe

## Dokumente aktuell halten
- `dokumentenregister.md` — Übersicht + Projektstatus
- `architecture.md` — Architektur, Komponenten, Datenfluss
- `tech-decisions.md` — ADRs mit Begründung
- `security.md` — Sicherheitsmaßnahmen
- `git-commit-guide.md` — Branch-Strategie, Commit-Konventionen
- `progress.md` — Sprint-Log, offene Punkte, nächste Schritte

## Spielparametrierung
- Alle balancierbaren und kalibrierbaren Spielparameter leben in `game-params_v1.0.yaml`
- **Jeder neu identifizierte Parameter** (Kalibrierungskonstante, Balancing-Wert, technisches Limit)
  wird sofort dort eingetragen – mit Kommentar zu Beschreibung, Einheit und Auswirkung
- Markierungskonventionen im YAML:
  - `[KALIBRIERUNG]` – physikalisch abgeleitete Konstante, wird durch Tests justiert
  - `[BALANCING]` – Gameplay-Parameter, wird durch Spieltests justiert
  - `[PERFORMANCE]` – technisches Limit, beeinflusst Server-Last
- Nie einen Magic Number direkt im Code hardcoden – immer aus game-params laden
- Bei Versionierung: geänderte game-params-Datei → neue Versionsnummer (`game-params_v1.1.yaml`)

## Datenbankstrategie — JSON over Columns

**Grundprinzip:** Datenbankeinträge werden auf ein Minimum reduziert.
Komplexe, sich häufig ändernde oder konfigurationsnahe Daten leben in YAML-Dateien
oder JSONB-Spalten — nie in separaten relationalen Spalten.

### Wann relationale Spalten:
- Echte FK-Beziehungen (star_id, planet_id, player_id)
- Felder, auf die WHERE / JOIN / ORDER BY angewendet werden muss (status, facility_type, tick_n)

### Wann JSONB:
- Konfigurationen und Parameter, die sich ohne Migration ändern sollen
- Inventare, Listen, Maps (z. B. `{ "iron_ore": 47.5 }`)
- Verschachtelte Strukturen ohne eigene Abfragebedarf (z. B. Deposit-Zustand pro Ressource)

### Wann YAML-Datei (nie in DB):
- Rezepte (`recipes_v1.0.yaml`)
- Gut-Definitionen (Sensitivitätsklasse, Masse, ID)
- Anlagen-Parameter (η per Level, Output/Tick, Baukosten)
- Alle game-params — in-memory geladen beim Server-Start

### Konsequenz:
- Neue Güter, Rezepte, Effizienzwerte → YAML-Änderung, **keine Migration**
- Neue Felder in Spielzustand (z. B. Deposit bekommt `depth`) → JSONB, **keine Migration**
- Neue echte Entität mit Relationen (z. B. Trade-Routen) → neue Tabelle mit Migration

## Stack
- Frontend: React + Vite + TypeScript
- Commits: Conventional Commits (`feat/fix/docs/chore`)
