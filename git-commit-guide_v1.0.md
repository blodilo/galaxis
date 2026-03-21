# Git Commit Guide – Galaxis v1.0

**Datum:** 2026-03-21

---

## Repository

`https://github.com/blodilo/galaxis`

---

## Branch-Strategie

```
main                   ← Stabiler Stand (geschützt, nur via PR)
feat/<bl-id>-<name>    ← Feature-Branches (BL-Items)
fix/<beschreibung>     ← Bugfixes
docs/<beschreibung>    ← Reine Dokumentationsänderungen
chore/<beschreibung>   ← Dependencies, Tooling, Konfiguration
```

**Beispiele:**
```
feat/bl-25-planet-rings
feat/image-generator       ← aktuell aktiv
fix/asteroid-shader-crash
docs/economy-v1-finalize
chore/update-threejs
```

Direkte Commits auf `main` nur für Hotfixes und Dokumentation.

---

## Commit-Konvention (Conventional Commits)

```
<type>(<scope>): <kurzbeschreibung>

[optionaler Body]
[Co-Authored-By: ...]
```

### Typen
| Typ | Wann |
|---|---|
| `feat` | Neue BL-Items, neue Features |
| `fix` | Bugfixes |
| `docs` | Nur Dokumentation (GDD, progress, specs) |
| `chore` | Dependencies, Tooling, Konfiguration, Umbenennung |
| `refactor` | Umstrukturierung ohne Verhaltensänderung |
| `test` | Vitest-Tests hinzufügen oder ändern |
| `perf` | Performance-Verbesserungen |

### Scopes
| Scope | Bedeutung |
|---|---|
| `frontend` | React/Vite/Three.js Frontend |
| `generator` | Galaxiengenerator (Go Backend) |
| `shader` | GLSL Shader (StarShader, PlanetShader, …) |
| `tools` | Python Tools (Scraper, Importer) |
| `economy` | Wirtschaftsmodell-Dokumente |
| `docs` | Allgemeine Dokumentation |
| `deps` | Abhängigkeits-Updates |

### Beispiele
```
feat(shader): Voronoi-Granulation V5 (Power Diagram, temporaler Lebenszyklus)
feat(frontend): Spektralfarben-Picker + Leuchtkraft-Slider im VisualTuner
fix(frontend): React uncontrolled-Warning in loadFromStorage (null-Filter)
docs(economy): Wirtschaftsmodell v1.0 finalisiert (alle 8 Entscheidungen)
chore: @ds/tokens → @creaminds/design umbenennen
perf(generator): InstancedMesh für Asteroidengürtel (BL-21)
```

---

## Workflow: BL-Item implementieren

```
1. git checkout -b feat/bl-XX-name
2. Implementieren + Tests
3. git add <spezifische Dateien>   # nie git add -A
4. git commit -m "feat(...): ..."
5. git push -u origin feat/bl-XX-name
6. PR auf main öffnen
7. Nach Merge: Branch löschen
```

---

## Aktuell offene Branches

| Branch | Inhalt | Status |
|---|---|---|
| `feat/image-generator` | BL-11 Image Generator + Shader V5 + Economy v1.0 | Aktiv |
