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

## Stack
- Frontend: React + Vite + TypeScript
- Commits: Conventional Commits (`feat/fix/docs/chore`)
