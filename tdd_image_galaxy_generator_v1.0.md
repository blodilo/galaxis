# Galaxis – Technical Design Document
## Image-Based Galaxy Generator (BL-11)
**Version:** 1.0 · **Datum:** 2026-03-17
**Status:** Bereit zur Implementierung (pending Commit-Entscheidung)
**Autor:** Martin Theis · **Review:** Claude Sonnet 4.6

---

## 1. Motivation & Abgrenzung

Der bisherige prozeduraler Generator (`Step1Morphology`) erzeugt Spiralarme durch analytische
Dichtefelder (logarithmische Spiralen + Gaussian Spread). Ergebnis: mathematisch korrekte, aber
visuell unbefriedigende Verteilungen – die organische Komplexität echter Galaxien (irreguläre
Armstruktur, asymmetrische Kerne, Sternenstaub-Bänder) ist nicht reproduzierbar.

**Ziel:** Ersetze `Step1Morphology` durch einen bildbasierten Generator, der hochauflösende
Teleskopaufnahmen als Dichtetemplates nutzt. Alle nachgelagerten Schritte (Step2–Step4)
bleiben unverändert.

---

## 2. Architektur-Überblick

```
game-params.yaml          morphology_catalog.yaml
  num_stars, radius_ly      asset_path → assets/morphology/<file>
  seed, exotic_counts       orientation (face-on / tilted + stretchY)
        │                         │
        ▼                         ▼
┌─────────────────────────────────────────────┐
│  Step 1 (neu): ImageStep1Morphology         │
│                                             │
│  A: AnalyzeImage         → ImageAnalysis   │
│  B: GeneratePositions    → []Star (XYZ,M)  │
│  C: SpectralCascade      → []Star (Typen)  │
│  D: PlaceExotics         → []Star (Extras) │
│                                             │
│  DB-Write: InsertStars                      │
│  Status: galaxies.status = 'morphology'     │
└─────────────────────────────────────────────┘
        │
        ▼
Step2Spectral  →  buildStarProps (Masse, Temp, Leuchtkraft)  [unverändert]
Step3Objects   →  FTLW-Grid                                   [unverändert]
Step4Planets   →  Planetensysteme                             [unverändert]
```

---

## 3. Datenstrukturen

### 3.1 ImageAnalysis (interner Zwischenspeicher)

```go
type ImageAnalysis struct {
    Width, Height int
    CDF           []float64  // Kumulative Helligkeitsverteilung, len = W*H
    TotalLum      float64    // Summe aller gamma-korrigierten Helligkeitswerte
    Intensities   []float32  // Normalisierte Helligkeit [0,1] pro Pixel, für Z-Streuung
    InitialClass  []uint8    // Spektralklassen-Index (0=O … 6=M) pro Pixel, für Kaskade
    // KEIN []string – uint8-Array ist 8× kompakter (2 MB statt 16 MB bei 4K)
}
```

### 3.2 Spektralklassen-Konstanten (erweiterbar)

```go
// Hauptreihe: Reihenfolge O→M (Index 0–6)
var mainSequenceOrder = []string{"O", "B", "A", "F", "G", "K", "M"}

// Zielquoten (astrophysikalisch, Summe = 1.0)
var spectralQuotas = map[string]float64{
    "O": 0.00003, "B": 0.00130, "A": 0.00600,
    "F": 0.03000, "G": 0.07600, "K": 0.12100, "M": 0.76567,
}

// RGB-Referenzwerte für Farb-Matching (Pass 1 Initialisierung)
var spectralRGB = map[string][3]uint8{
    "O": {157, 180, 255}, "B": {170, 191, 255}, "A": {202, 216, 255},
    "F": {251, 248, 255}, "G": {255, 244, 232}, "K": {255, 221, 180},
    "M": {255, 189, 111},
}

// Hex-Farben für DB-Speicherung (Frontend-Rendering)
var spectralHex = map[string]string{
    "O": "#9db4ff", "B": "#aabfff", "A": "#cad8ff", "F": "#fbf8ff",
    "G": "#fff4e8", "K": "#ffddb4", "M": "#ffbd6f",
}
```

### 3.3 Exotika-Konfiguration (game-params.yaml)

Neuer Abschnitt unter `galaxy:`:

```yaml
galaxy:
  # ...bestehende Parameter...

  exotic_counts:
    # [BALANCING] Anzahl exotischer Objekte. Werden ZUSÄTZLICH zu num_stars erzeugt.
    # Exotika zählen nicht gegen das num_stars-Cap.
    wr:         15    # Wolf-Rayet Sterne (cyan, nahe helle Nebelregionen)
    rstar:      80    # Rote Überriesen (dunkelrot, Außenbereiche)
    sstar:      40    # S-Sterne (orange-rot, mittlere Regionen)
    pulsar:     25    # Pulsare (blass-blau, nahe SNR-Regionen)
    stellar_bh: 10    # Stellare Schwarze Löcher (fast schwarz)
    # SMBH: immer genau 1, am Helligkeitsschwerpunkt – nicht konfigurierbar

  exotic_placement: "color_affine"
    # Strategie: "color_affine" (Standard) | "random"
    # color_affine: Exotika werden in Bildregionen platziert, deren Initialklasse
    #               farblich am nächsten liegt (z.B. WR → A/B-Pixel-Regionen).
    # random:       Gleichverteilte Platzierung über die CDF (Fallback für Tests).
```

---

## 4. Pipeline im Detail

### 4.1 Schritt A: AnalyzeImage

**Input:** `imagePath string` (aus Morphologie-Katalog: `assets/morphology/<file>`)
**Output:** `ImageAnalysis`, error

```
FOR EACH pixel (x, y):
    rgb ← img.At(x, y)                     // Go: RGBA() → /257 für 0–255
    r, g, b ← float64(rgb) / 255.0         // normalisiert [0, 1]

    lum ← 0.2989·r + 0.5870·g + 0.1140·b  // Rec. 709 Graustufen
    intensity ← pow(lum, 1.5)              // Gamma-Korrektur: Kontrast der Arme

    idx ← y·Width + x
    CDF[idx]          ← CDF[idx-1] + intensity   // Laufende Summe
    Intensities[idx]  ← float32(lum)             // Unkorrekt für Z-Verteilung!
                                                  // → lum (nicht gamma) für natürl. Z-Dicke
    InitialClass[idx] ← closestSpectralClass(r·255, g·255, b·255)

TotalLum ← CDF[W·H - 1]  // Gesamtsumme = letzter CDF-Wert
```

**Speicherbedarf 4K-Bild (4096×4096 = 16.7M Pixel):**
- `CDF []float64`:      ~134 MB
- `Intensities []float32`: ~67 MB
- `InitialClass []uint8`:  ~16 MB
- **Gesamt:** ~217 MB Peak während Step 1 (wird nach GeneratePositions freigegeben)

> **Optimierung (optional):** CDF als float32 – Präzision auf 7 Stellen statt 15,
> ausreichend für Binärsuche. Reduziert Peak auf ~150 MB.

**Sternzahl-Bestimmung:**
Nicht dynamisch aus `TotalLum` (zu sensitiv auf Bildhelligkeit). Stattdessen direkt
aus `game-params.galaxy.num_stars`. Der Parameter bleibt der Single Source of Truth.

### 4.2 Schritt B: GeneratePositions

**Input:** `analysis ImageAnalysis`, `numStars int`, `radiusLY float64`,
          `seed int64` (aus `game-params.galaxy.seed`)
**Output:** `[]model.Star` (XYZ gesetzt, Type = M als Platzhalter)

```
rng ← PCG(seed, 0xIMAGE_STREAM)   // Deterministischer Seed, separater Stream

FOR i = 0 to numStars-1:
    dart ← rng.Float64() · TotalLum

    // O(log n) Binärsuche im CDF
    pixelIdx ← sort.Search(len(CDF), func(j) { CDF[j] >= dart })
    pixelIdx  = min(pixelIdx, W·H - 1)

    px ← pixelIdx % Width
    py ← pixelIdx / Width

    // Sub-Pixel Noise (verhindert Gitter-Artefakte)
    xf ← float64(px) + (rng.Float64() - 0.5)
    yf ← float64(py) + (rng.Float64() - 0.5)

    // Auf Lichtjahre skalieren (Face-On: kein stretchY)
    // Für Schrägansichten (z.B. Andromeda): yf ← yf · stretchY
    star.X ← (xf / (Width/2)  - 1.0) · radiusLY
    star.Y ← -(yf / (Height/2) - 1.0) · radiusLY

    // Z-Achse: Bulge (helle Pixel) vs. Scheibe (dunkle Pixel)
    // Intensities[pixelIdx] = nicht-gamma-korrigierte Helligkeit
    intensity ← Intensities[pixelIdx]
    zSpread   ← (intensity · 0.06 + 0.004) · radiusLY
    // Kalibrierung: 50.000 ly Radius → heller Kern: ±3.000 ly, Arme: ±200 ly
    // Milchstraße: Scheibe ~±500 ly, Bulge ~±5.000 ly → [KALIBRIERUNG]
    star.Z ← rng.NormFloat64() · zSpread

    star.Type = StarTypeM   // Platzhalter; wird durch Kaskade ersetzt
    star.PlanetSeed ← planetSeed(galaxySeed, star.ID)
```

### 4.3 Schritt C: SpectralCascade (korrigierte 2-Pass-Kaskade)

**Input:** `stars []model.Star`, `analysis.InitialClass []uint8`
**Output:** `stars []model.Star` (Type gesetzt)

Die Initialklasse jedes Sterns stammt aus dem Bild-Pixel. Galaxienbilder sind typisch
warm (K/M-dominant), aber mit überrepräsentierten hellen Sternen (O/B in Kernnähe).
Die Kaskade erzwingt die `spectralQuotas` in zwei Passes.

```
// Zielanzahlen aus Quoten berechnen
targets[c] ← int(spectralQuotas[c] · numStars)  // für O, B, A, F, G, K
targets["M"] ← numStars - sum(targets[ohne M])   // M fängt Rundungsdifferenz auf

// Sterne nach Initialklasse aus Bild gruppieren
classLists[c] ← Indices aller Stars mit InitialClass == c

// ── PASS 1: O→M (Abkühl-Welle, Top-Down) ───────────────────────────
// Schiebt Überschüsse heißer Klassen zum nächstkühleren Nachbarn.
// Nach Pass 1: jede Klasse ≤ Ziel; Überschüsse sammeln sich in M.
FOR i ← 0 to 5 (O, B, A, F, G, K):
    excess ← len(classLists[i]) - targets[i]
    IF excess > 0:
        shuffle(classLists[i])  // Geografische Natürlichkeit erhalten
        verschiebe excess Sterne: classLists[i] → classLists[i+1]

// ── PASS 2: M→O (Aufheiz-Sog, Bottom-Up) ───────────────────────────
// Füllt Defizite kühler Klassen aus dem nächstkühleren Nachbarn.
// Iteriert von K nach O (absteigend), M ist das Reservoir.
FOR i ← 5 downto 0 (K, G, F, A, B, O):
    deficit ← targets[i] - len(classLists[i])
    IF deficit > 0:
        take ← min(deficit, len(classLists[i+1]))
        shuffle(classLists[i+1])
        verschiebe take Sterne: classLists[i+1] → classLists[i]

// Finale Zuweisung
FOR class, indices IN classLists:
    FOR idx IN indices:
        stars[idx].Type     ← StarType(class)
        stars[idx].ColorHex ← spectralHex[class]
        // Masse/Temp/Radius werden NICHT hier gesetzt – das ist Step2Spectral
```

**Konvergenz-Garantie:** Pass 1 akkumuliert alle Überschüsse in M (M ist immer ≥ 0 nach Pass 1,
da M die größte Klasse mit 76,5% Zielquote ist). Pass 2 zieht dann Defizite von K→G→F→A→B→O,
wobei jeder Schritt M über die Kette anzapft. O und B (zusammen < 0,15%) erhalten bei 50k Sternen
max. 77 Exemplare – praktisch immer verfügbar aus K/M-Reservoir.

### 4.4 Schritt D: PlaceExotics

**Input:** `analysis ImageAnalysis`, `exoticCfg ExoticConfig`, `galaxyID uuid.UUID`, `seed int64`
**Output:** `[]model.Star` (separate Liste, wird zusätzlich zu Hauptsternen insertiert)

```
// Affine Platzierungs-Map: Welche Initialklasse bevorzugen die Exotika?
affinityClass := map[string]uint8{
    "WR":        classIndex("A"),   // Cyan nahe blauen Sternbildungsregionen
    "RStar":     classIndex("M"),   // Dunkelrot in roten Außenbereichen
    "SStar":     classIndex("K"),   // Orange-rot in mittleren Regionen
    "Pulsar":    classIndex("B"),   // Blass-blau nahe heißen Regionen / SNR
    "StellarBH": classIndex("M"),   // Fast schwarz, in dichten dunklen Regionen
}

rng ← PCG(seed, 0xEXOTIC_STREAM)

FOR exoticType, count IN exoticCounts:
    targetClass ← affinityClass[exoticType]

    // Kandidaten-Pixel: Pixel mit passender Initialklasse
    // Kein Vorfilter nötig – sample via CDF, verwerfe bis Klasse passt (rejection)
    // Effizient weil A/B/K/M zusammen ~99.8% aller Pixel abdecken
    FOR i = 0 to count-1:
        REPEAT:
            dart  ← rng.Float64() · TotalLum
            idx   ← binarySearch(CDF, dart)
        UNTIL InitialClass[idx] == targetClass

        // Position wie in GeneratePositions
        star ← positionFromPixel(idx, analysis, rng, radiusLY)
        star.Type = StarType(exoticType)
        exoticStars ← append(exoticStars, star)

// SMBH: am photometrischen Schwerpunkt des Bildes (hellstes Cluster-Zentrum)
smbhPixel ← findBrightnessCentroid(analysis)
smbh      ← positionFromPixel(smbhPixel, analysis, rng, radiusLY)
smbh.Type = StarTypeSMBH
smbh.X, smbh.Y, smbh.Z = 0, 0, 0  // Override: SMBH immer exakt im Zentrum
exoticStars ← append(exoticStars, smbh)
```

**Hinweis:** `PlaceExotics` gibt eine **separate Liste** zurück. Der Aufrufer insertiert
beide Listen unabhängig. Die Exotika zählen nicht gegen `num_stars`.

---

## 5. Integration in bestehende Pipeline

### 5.1 Änderungen an generator.go

`Step1Morphology` wird ersetzt. Signatur bleibt **identisch** (kein Breaking Change für
Aufrufer in `generate_handlers.go`):

```go
func (g *Generator) Step1Morphology(
    ctx  context.Context,
    galaxyID uuid.UUID,
    emit func(string, int, int, string),
) error
```

Intern: Ruft `AnalyzeImage → GeneratePositions → SpectralCascade → PlaceExotics` auf.

**Neue Datei:** `internal/galaxy/image_generator.go`
Enthält: `AnalyzeImage`, `GeneratePositions`, `SpectralCascade`, `PlaceExotics`,
`findBrightnessCentroid`, `closestSpectralClass`.

`generator.go` behält `Step2Spectral`, `Step3Objects`, `Step4Planets`, `computeFTLW`,
`planetSeed` unverändert.

### 5.2 Änderungen an config.go

```go
type GalaxyConfig struct {
    // ...bestehende Felder...
    ExoticCounts ExoticCounts `yaml:"exotic_counts" json:"exotic_counts"`
    ExoticPlacement string   `yaml:"exotic_placement" json:"exotic_placement"`
}

type ExoticCounts struct {
    WR        int `yaml:"wr"         json:"wr"`
    RStar     int `yaml:"rstar"      json:"rstar"`
    SStar     int `yaml:"sstar"      json:"sstar"`
    Pulsar    int `yaml:"pulsar"     json:"pulsar"`
    StellarBH int `yaml:"stellar_bh" json:"stellar_bh"`
}
```

### 5.3 Morphologie-Katalog: Bildpfad-Lookup

`Step1Morphology` erhält den `morphologyID` (aus `generateRequest.MorphologyID`).
Der Katalog-YAML wird beim Server-Start eingelesen. Lookup:

```go
catalog.FindByID(morphologyID) → MorphologyTemplate{ AssetPath: "assets/morphology/..." }
```

Bereits implementiert in `api/catalog_handlers.go` (Katalog-Laden) –
der Pfad muss nur an den Generator weitergegeben werden.

### 5.4 Emit-Punkte (SSE-Progress)

```
emit("morphology", 0,         numStars, "Analysiere Bild…")
emit("morphology", 0,         numStars, "N Pixel verarbeitet")   // alle 1M Pixel
emit("morphology", i·batch,   numStars, "")                      // alle 1000 Sterne
emit("morphology", numStars,  numStars, "Spektralkaskade…")
emit("morphology", numStars,  numStars+totalExotics, "Exotika platziert")
```

---

## 6. Performance & Speicher

| Phase | Laufzeit (4K, 50k Sterne) | RAM Peak |
|---|---|---|
| AnalyzeImage (16.7M Pixel) | ~300–500 ms | ~217 MB |
| GeneratePositions (50k × log₂(16.7M)) | < 10 ms | +~3 MB |
| SpectralCascade (2 Passes, 50k Elemente) | < 1 ms | vernachlässigbar |
| PlaceExotics (~170 Rejection-Samples) | < 1 ms | vernachlässigbar |
| InsertStars DB (50k + 170 Batch) | ~2–5 s | vernachlässigbar |
| **Gesamt Step 1** | **~3–6 s** | **~217 MB** |

Zum Vergleich: Alter Generator (Rejection Sampling, 50k Sterne): ~60–120 s, ~50 MB.
Der neue Generator ist **signifikant schneller** bei höherer visueller Qualität.

> `ImageAnalysis` wird nach `GeneratePositions` explizit auf `nil` gesetzt,
> damit der GC den 217 MB-Block vor dem DB-Write freigeben kann.

---

## 7. Systemgrenzen & Offene Punkte

| Nr. | Thema | Auswirkung | Vorschlag |
|---|---|---|---|
| 7.1 | Schrägansichten (z.B. Andromeda, M31) | Y-Achse gestaucht | `stretchY`-Parameter im Katalog-YAML: `deproject: { stretch_y: 1.4 }` |
| 7.2 | Zu dunkle Bilder (wenig Kontrast) | Sterne clustern im Zentrum | Normalisierung: CDF auf `[0, TotalLum]` wirkt bereits als Auto-Exposure |
| 7.3 | PNG vs. JPEG Artefakte | JPEG-Kompression kann Spektralklassen-Rauschen erzeugen | Empfehlung: PNG oder JPEG ≥ 95% Qualität im Katalog |
| 7.4 | Exotika Rejection-Sampling | Bei sehr kleinen Klassen (A/B) ggf. viele Verwürfe | Timeout: max. 10k Versuche, dann Fallback auf nächste verwandte Klasse |
| 7.5 | Step2Spectral überschreibt ColorHex | `buildStarProps` setzt eigene Farben | Kein Problem – Step2 ist für Farbe zuständig, Kaskade nur für Type |
| 7.6 | Galaxien mit `enabled: false` im Katalog | Generator würde fehlschlagen | Validierung in `triggerStep1`: Reject wenn Template disabled |

---

## 8. Commit-Basis-Entscheidung

Zwei Optionen:

### Option A: `master` (424a19d) — sauberer Schnitt

```
master (424a19d)
└── feat/image-generator
    ├── Neuer image_generator.go
    ├── Angepasste config.go (ExoticCounts)
    └── Angepasste generator.go (Step1 ersetzt)
```

**Pro:** Kein Legacy-Code, saubere Basis.
**Contra:** Fehlt die gesamte Step1–4-Infrastruktur, SSE-Progress, GalaxyPicker,
DB-Migrations 002+003, Planet-Generator, SystemScene. Alles müsste neu gemergt werden.
**Empfehlung: Nein.**

### Option B: Feature-Branch (b3b8bbc) — empfohlen

```
feat/step-generation-sse-galaxy-picker (b3b8bbc)
└── feat/image-generator
    ├── Neuer internal/galaxy/image_generator.go
    ├── Angepasste internal/config/config.go  (+ExoticCounts)
    ├── Angepasste internal/galaxy/generator.go (Step1 ersetzt)
    └── Angepasste game-params_v1.3.yaml (+exotic_counts)
```

**Pro:** Enthält die vollständige Step1–4-Pipeline, SSE, GalaxyPicker, Migrations, Planets.
Der neue Generator ersetzt **nur** `Step1Morphology` / `placeStarsPlaceholder` – alle
anderen Dateien bleiben unberührt.
**Contra:** Der Branch enthält noch offene Bugs (M-Sterne-Viewer-Reload) – aber die sind
bereits gefixt im letzten Commit des Branches.
**Empfehlung: Ja. Branch `feat/image-generator` von `b3b8bbc` abzweigen.**

---

## 9. Abnahmekriterien (Definition of Done)

- [ ] `go test ./internal/galaxy/...` grün (Kaskade-Unit-Tests: Quoten eingehalten, Summe = numStars)
- [ ] Visuelle Verifikation: Galaxie mit M101-Template zeigt organische Spiralarme
- [ ] Spektralverteilung in DB verifiziert: `SELECT star_type, COUNT(*) FROM stars GROUP BY star_type`
      → M ≈ 76.5%, K ≈ 12.1%, O < 0.01%
- [ ] Exotika nicht in `star_type IN ('O','B','A','F','G','K','M')` — separat zählbar
- [ ] Step1 SSE-Progress zeigt sinnvolle Fortschrittswerte im Frontend
- [ ] Galaxie mit `enabled: false` Template → API 400 Bad Request
- [ ] RAM-Footprint auf Testserver gemessen ≤ 300 MB während Step1
