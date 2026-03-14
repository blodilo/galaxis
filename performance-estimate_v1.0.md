# Performance- & Größenabschätzung – Galaxis v1.0

**Datum:** 2026-03-13

---

## 1. FTLW-Voxelgrid

### Geometrie

| Parameter | Referenzwert | Notiz |
|---|---|---|
| Galaxieradius | 50.000 ly | Konfigurierbar |
| Galaxiedurchmesser (x/y) | 100.000 ly | |
| Scheibendicke (z, ±2σ h_z) | ~4.000 ly | h_z ≈ 1.000 ly → ±2σ |
| Voxelgröße (Standard) | 500 ly | Konfigurierbar |

### Gridgröße bei 500 ly Voxeln

```
Voxel x: 100.000 / 500 = 200
Voxel y: 100.000 / 500 = 200
Voxel z:   4.000 / 500 =   8
─────────────────────────────
Gesamt:  200 × 200 × 8 = 320.000 Voxel
Speicher (float32, 4 Byte): 1,28 MB RAM
```

### Gridgröße bei verschiedenen Voxelgrößen

| Voxelgröße | Voxel gesamt | RAM (float32) | A*-Nodes |
|---|---|---|---|
| 1.000 ly | 40.000 | 160 KB | sehr schnell |
| 500 ly | 320.000 | 1,3 MB | schnell |
| 250 ly | 2,56 Mio | 10 MB | akzeptabel |
| 100 ly | 40 Mio | 160 MB | langsam ohne Hierarchie |
| 50 ly | 320 Mio | 1,28 GB | nicht praktikabel |

**Empfehlung: 500 ly Standardvoxel.** Das gesamte FTLW-Grid passt trivial in RAM. Feinere Auflösung (250 ly) ist für lokale Gefechtszonen möglich.

---

## 2. A*-Pathfinding Performance

### Grundlage

A* auf einem Grid mit N Nodes hat Komplexität **O(N log N)** im Worst Case.

| Voxelgröße | Nodes | Geschätzte Zeit (moderner CPU) |
|---|---|---|
| 500 ly | 320.000 | < 10 ms |
| 250 ly | 2,56 Mio | 50–200 ms |
| 100 ly | 40 Mio | mehrere Sekunden – zu langsam |

### Hierarchisches A* (empfohlen)

Für Routen über große Distanzen: zweistufig

```
Stufe 1 – Grob (2.500 ly Voxel):
  Grid: 40 × 40 × 2 = 3.200 Nodes → < 1 ms
  Ergebnis: Wegpunkte mit ≈ Genauigkeit 2.500 ly

Stufe 2 – Fein (500 ly Voxel):
  Nur entlang der Grob-Route verfeinern
  Lokale Suchtiefe: ~50 Nodes pro Segment → < 5 ms

Gesamt: < 10 ms pro Route, skaliert bis sehr große Karten
```

---

## 3. Sternendaten (PostgreSQL)

| Sterne | Zeilengröße | Tabellengröße | Planeten (avg 5×) |
|---|---|---|---|
| 10.000 | ~300 Byte | 3 MB | 15 MB |
| 50.000 | ~300 Byte | 15 MB | 75 MB |
| 200.000 | ~300 Byte | 60 MB | 300 MB |
| 1.000.000 | ~300 Byte | 300 MB | 1,5 GB |

Planeten werden JIT generiert und persistiert – nicht alle auf einmal vorhanden.
Bei 50.000 Sternen und 20% erkundeten Systemen: ~15 MB Planetendaten.

### Räumliche Queries (Bounding Box)

```
SELECT * FROM stars WHERE galaxy_id = ?
  AND x BETWEEN x1 AND x2
  AND y BETWEEN y1 AND y2
  AND z BETWEEN z1 AND z2;
```

Mit B-Tree-Index auf (galaxy_id, x, y, z):
- 50.000 Sterne, Abfrage 10% der Fläche: ~1–5 ms
- 500.000 Sterne: ~5–20 ms

Bei sehr großen Karten: Partitionierung nach Sektor oder PostGIS-Extension.

---

## 4. Maximale kartengröße – Abschätzung

### Limitierende Faktoren

| Faktor | Limit | Bemerkung |
|---|---|---|
| FTLW-Grid RAM | ~320 MB für 250 ly Voxel bei 200.000 ly Durchmesser | Kein echter Blocker |
| A*-Performance | > 10 Mio Nodes kritisch | Hierarchie löst das |
| Sterne DB | > 500.000 → Index-Optimierung nötig | Manageable |
| Tick-Performance | Nur aktive Systeme relevant | Nicht kartengrößenabhängig |
| Sichtbarkeit/FoW Queries | Quadratisch zur Spielerzahl | Wichtigster Blocker bei Multiplayer |

### Praktische Maximalwerte (Referenzserver: 8 vCPU, 32 GB RAM)

| Konfiguration | Sterne | Radius | Voxelgröße | FTLW-RAM | Eignung |
|---|---|---|---|---|---|
| Weekend-Party | 10.000 | 20.000 ly | 500 ly | 0,1 MB | 20 Spieler |
| Standardpartie | 50.000 | 50.000 ly | 500 ly | 1,3 MB | 100 Spieler |
| Epische Partie | 200.000 | 100.000 ly | 500 ly | 5 MB | 100 Spieler+ |
| **Maximum** | **500.000** | **150.000 ly** | **500 ly** | **15 MB** | Langzeitkampagne |

**Fazit:** Das FTLW-Grid ist kein Performance-Bottleneck. Die Sterne-Tabelle ist bis ~500.000 Einträge problemlos. Der eigentliche Skalierungsengpass liegt in der **Spiellogik pro Tick** (aktive Flotten, Wirtschaftsberechnungen) – nicht in der Kartengröße selbst.

### Empfohlene Standardkonfiguration

```yaml
galaxy:
  num_stars: 50000
  radius_ly: 50000

ftlw:
  voxel_size_ly: 500          # Hierarchie: Grob 2500 ly, Fein 500 ly
  hierarchical_pathfinding: true
  coarse_voxel_size_ly: 2500
```

---

## 5. Generierungszeit (Einmalig, pre-game)

Geschätzte Laufzeit von `galaxy-gen` auf Standardhardware:

| Schritt | 50.000 Sterne | 200.000 Sterne |
|---|---|---|
| Morphologie / Dichtefeld | < 1 s | < 5 s |
| Nebel platzieren | < 1 s | < 1 s |
| Sterne sampeln + DB-Insert | 5–15 s | 30–60 s |
| FTLW-Grid berechnen | 10–30 s | 60–120 s |
| **Gesamt** | **~1–2 Min** | **~3–5 Min** |

Einmalig, akzeptabel. Grid-Berechnung parallelisierbar (goroutines je Chunk).
