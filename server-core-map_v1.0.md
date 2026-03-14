# Server Core – Kartenfunktionen v1.0

**Datum:** 2026-03-13
**Referenz:** GDD v1.24 · architecture_v1.0.md

---

## Übersicht: Zwei Phasen

```
┌─────────────────────────────────────────────────────────────────┐
│  ERZEUGUNGSPHASE (einmalig, vor Spielstart)                     │
│                                                                  │
│  galaxy-gen (Go-Binary / CLI)                                   │
│    1. Galaxie-Morphologie                                       │
│    2. Nebel                                                     │
│    3. Sterne                                                    │
│    4. FTLW-Voxelgrid                                            │
│    5. → PostgreSQL (persistent)                                 │
└─────────────────────────────────────────────────────────────────┘
                        ↓ Daten in DB
┌─────────────────────────────────────────────────────────────────┐
│  LAUFZEITPHASE (Game Server, während der Partie)                │
│                                                                  │
│  map-service (integriert im Game Server)                        │
│    • Galaxy Query API (Sterne, FTLW, Regionen)                  │
│    • JIT Planet Generator (bei System-Scan)                     │
│    • FTL Pathfinder (Route + Kosten)                            │
│    • FTLW Modifier (Spieler baut Tunnel/Tore)                   │
└─────────────────────────────────────────────────────────────────┘
```

---

## Teil 1: Erzeugungsphase

### 1.1 Prozess-Übersicht

Der Galaxiegenerator ist eine **separate Go-Binary** (`galaxy-gen`), die einmalig vor Spielstart ausgeführt wird. Sie ist kein Teil des laufenden Game Servers. Vorteile:
- Kann zeitintensiv rechnen ohne den Game Server zu blockieren
- Deterministisch durch konfigurierten Seed
- Wiederholbar (gleicher Seed → gleiche Galaxie)

```
galaxy-gen --config game.yaml --seed 42 --db postgresql://...
```

### 1.2 Konfigurationsparameter

```yaml
galaxy:
  seed: 42                    # Deterministischer Startwert
  num_stars: 50000            # Anzahl Sternensysteme
  radius_ly: 50000            # Galaxis-Radius in Lichtjahren
  type: SBb                   # Morphologie-Typ (SBa, SBb, SBc)
  arms: 2                     # Anzahl Spiralarme
  arm_winding: 2.0            # Wie stark sich Arme drehen
  arm_spread: 0.5             # Auffransung der Arme
  smbh_mass_solar: 4_300_000  # Masse des zentralen Schwarzen Lochs

ftlw:
  k_factor: 1.0               # Skalierungs-Gameplay-Faktor
  cutoff_percent: 1.0         # Unter x% des Vakuumwerts: ignoriert
  voxel_size_ly: 500          # Voxelgröße in Lichtjahren

game:
  tick_duration_minutes: 60   # Strategietick-Länge
  max_players: 100
```

### 1.3 Generierungs-Pipeline (Schritte)

#### Schritt 1: Morphologie – Dichtefeld aufbauen

Berechnet die Sterndichte `ρ(x,y,z)` als Summe aus drei Komponenten:

```
ρ_gesamt(x,y,z) = ρ_disk(R,z) + ρ_bulge(x,y,z) + ρ_arms(x,y,z)

ρ_disk(R,z)  = ρ₀ · exp(-R/R_d) · exp(-|z|/h_z)
               R_d (Scale Length) ≈ 3500 ly
               h_z (Scale Height) ≈ 1000 ly

ρ_bulge(x,y,z) = de-Vaucouleurs-Profil (sphärisch + Bar-Streckung)
                 Bar: x-Achse 2× gestreckt gegenüber y

ρ_arms(R,θ)  = Gaußscher Boost entlang log. Spiralen: R(θ) = a · e^(b·θ)
               Boost-Breite: σ_arm (konfigurierbar)
```

Output: In-memory-Dichtefeld (wird für Sternplatzierung verwendet, nicht persistiert)

#### Schritt 2: SMBH platzieren

Erstes Objekt: Schwarzes Loch im Zentrum (0,0,0) mit konfigurierter Masse.

```sql
INSERT INTO stars (galaxy_id, x, y, z, star_type, mass_solar, ...)
VALUES (?, 0, 0, 0, 'SMBH', 4300000, ...)
```

#### Schritt 3: Nebel erzeugen

Typen: H-II (Sternentstehung), SNR (Supernova-Überreste), Globular (Kugelsternhaufen)

- H-II: Entlang der Spiralarme, Simplex Noise für organische Form
- SNR: Zufällig verteilt (primär in Spiralarmen, außerhalb Bulge)
- Globular: Oberhalb/unterhalb der galaktischen Scheibe (|z| groß)

Jeder Nebel speichert: Zentrum, Radius, Typ, Dichte

#### Schritt 4: Sterne platzieren

Für jeden Stern:
1. Position samplen: Rejection Sampling basierend auf `ρ_gesamt`
2. Prüfen: Liegt die Position in einem Nebel?
3. Spektralklasse würfeln (lokal gewichtet, s.u.)
4. Masse, Leuchtkraft, Radius aus Klasse ableiten (Formeln + ±10% Noise)
5. Seed für JIT-Planetengenerierung erzeugen (`sha256(galaxy_seed + star_id)`)
6. In DB schreiben (Batch-Inserts, 1000er-Blöcke)

**Spektralklassen-Gewichtung nach Nebeltyp:**

| Nebeltyp | O | B | A | F | G | K | M | WR | Pulsar | BH | R/S |
|---|---|---|---|---|---|---|---|---|---|---|---|
| H-II Region | hoch | hoch | mittel | mittel | niedrig | niedrig | niedrig | hoch | – | – | – |
| SNR | – | – | – | niedrig | niedrig | mittel | mittel | – | hoch | mittel | – |
| Globular / Bulge | – | – | – | – | niedrig | mittel | hoch | – | niedrig | niedrig | hoch |
| Freie Scheibe | – | selten | selten | mittel | mittel | hoch | sehr hoch | – | – | – | selten |

**Physikalische Eigenschaften pro Klasse (Formeln):**

```
Hauptreihensterne (O,B,A,F,G,K,M):
  Masse M:        Tabelle [Min, Max] → uniform sample → × (1 + N(0, 0.1))
  Leuchtkraft L:  M^3.5  × noise
  Radius R:       M^0.8  × noise  (massereich)
               OR M^0.5  × noise  (massearm)
  Temperatur T:   Tabelle per Klasse

R/S-Sterne (Rote/S-Riesen):
  Masse:  1–3 M☉, Radius: 100–500 R☉ (entkoppelt!)

Wolf-Rayet:
  Masse: 10–200 M☉, extrem heiß, hohe Leuchtkraft

Pulsare:
  Masse: 1.4–2.1 M☉ (eng), Radius: 10 km (fix, in DB als Meter)

Schwarze Löcher (stellar):
  Masse: 5–100 M☉, Radius = Schwarzschild-Radius (2GM/c²)
```

#### Schritt 5: FTLW-Voxelgrid berechnen

Das Gitter wird **einmalig nach Sternplatzierung** berechnet und persistiert.

```
Für jede Voxel-Zelle (x_v, y_v, z_v):
  FTLW = FTLW_vakuum  (Konfigurationsparameter, z.B. 1.0)

  Für jeden Stern im Einflussbereich (r < cutoff_radius):
    r = Abstand(Voxel-Zentrum, Stern-Position)
    beitrag = k * M_solar / r²
    if beitrag > cutoff_percent * FTLW_vakuum:
      FTLW += beitrag

  Speichere FTLW in voxel-Tabelle
```

**Cutoff-Radius** pro Stern (berechnet, nicht iteriert über gesamte Galaxie):
```
r_cutoff = sqrt(k * M / (cutoff_percent * FTLW_vakuum))
```

→ Nur Sterne innerhalb `r_cutoff` ihres Voxels werden berücksichtigt.

**Speicherung:** 3D-Grid als komprimiertes Binary-Blob pro "Chunk" (z.B. 10×10×10 Voxel) in PostgreSQL. Chunk-Koordinaten als Index.

---

## Teil 2: Datenmodell (PostgreSQL)

### Tabelle: `galaxies`
```sql
CREATE TABLE galaxies (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    seed        BIGINT NOT NULL,
    config      JSONB NOT NULL,         -- vollständige Konfiguration
    status      TEXT NOT NULL,          -- 'generating' | 'ready' | 'active'
    created_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### Tabelle: `nebulae`
```sql
CREATE TABLE nebulae (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    galaxy_id   UUID REFERENCES galaxies(id),
    type        TEXT NOT NULL,          -- 'HII' | 'SNR' | 'Globular'
    center_x    DOUBLE PRECISION,
    center_y    DOUBLE PRECISION,
    center_z    DOUBLE PRECISION,       -- alle Koordinaten in Lichtjahren
    radius_ly   DOUBLE PRECISION,
    density     DOUBLE PRECISION        -- relativer Dichtewert 0–1
);
```

### Tabelle: `stars`
```sql
CREATE TABLE stars (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    galaxy_id       UUID REFERENCES galaxies(id),
    nebula_id       UUID REFERENCES nebulae(id) NULL,
    x               DOUBLE PRECISION NOT NULL,  -- Lichtjahre
    y               DOUBLE PRECISION NOT NULL,
    z               DOUBLE PRECISION NOT NULL,
    star_type       TEXT NOT NULL,
    -- 'O'|'B'|'A'|'F'|'G'|'K'|'M'|'WR'|'RStar'|'SStar'
    -- |'Pulsar'|'StellarBH'|'SMBH'
    spectral_class  TEXT,              -- NULL für exotische Typen
    mass_solar      DOUBLE PRECISION,
    luminosity_solar DOUBLE PRECISION,
    radius_solar    DOUBLE PRECISION,
    temperature_k   DOUBLE PRECISION,
    color_hex       TEXT,
    planet_seed     BIGINT NOT NULL,   -- für JIT-Planetengenerierung
    planets_generated BOOLEAN DEFAULT FALSE
);

-- Räumlicher Index für Region-Queries
CREATE INDEX idx_stars_position ON stars USING btree (galaxy_id, x, y, z);
-- Für schnelle Bounding-Box-Abfragen:
CREATE INDEX idx_stars_xyz ON stars (galaxy_id, x, y, z)
    WHERE star_type != 'SMBH';
```

### Tabelle: `ftlw_chunks`
```sql
CREATE TABLE ftlw_chunks (
    galaxy_id   UUID REFERENCES galaxies(id),
    chunk_x     INTEGER,
    chunk_y     INTEGER,
    chunk_z     INTEGER,
    -- Komprimiertes Binary: float32-Array der FTLW-Werte im Chunk
    data        BYTEA NOT NULL,
    PRIMARY KEY (galaxy_id, chunk_x, chunk_y, chunk_z)
);
```

### Tabelle: `ftlw_overrides`
*(Spieler-modifizierte Voxelwerte – Lategame-Mechanik)*
```sql
CREATE TABLE ftlw_overrides (
    galaxy_id   UUID REFERENCES galaxies(id),
    voxel_x     INTEGER,
    voxel_y     INTEGER,
    voxel_z     INTEGER,
    multiplier  DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    built_by    UUID,               -- Spieler/Fraktion
    structure_type TEXT,            -- 'tunnel' | 'gate' | 'stabilizer'
    PRIMARY KEY (galaxy_id, voxel_x, voxel_y, voxel_z)
);
```

### Tabelle: `star_systems` (JIT-generiert)
```sql
CREATE TABLE star_systems (
    star_id         UUID PRIMARY KEY REFERENCES stars(id),
    generated_at    TIMESTAMPTZ DEFAULT NOW()
);
```

### Tabelle: `planets`
```sql
CREATE TABLE planets (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    star_id                 UUID REFERENCES stars(id),
    orbit_index             SMALLINT,          -- 0 = innerster Orbit
    planet_type             TEXT NOT NULL,
    -- 'rocky'|'gas_giant'|'ice_giant'|'asteroid_belt'
    orbit_distance_au       DOUBLE PRECISION,
    mass_earth              DOUBLE PRECISION,
    radius_earth            DOUBLE PRECISION,
    surface_gravity_g       DOUBLE PRECISION,
    atmosphere_type         TEXT,
    -- 'terran'|'volcanic'|'cryogenic'|'arid'|'none'
    surface_temp_k          DOUBLE PRECISION,
    albedo                  DOUBLE PRECISION,
    usable_surface_fraction DOUBLE PRECISION,  -- 0.0–1.0
    biomass_potential       DOUBLE PRECISION,  -- 0.0–1.0
    resource_deposits       JSONB              -- {element_id: amount}
);
```

### Tabelle: `moons`
```sql
CREATE TABLE moons (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    planet_id         UUID REFERENCES planets(id),
    orbit_index       SMALLINT,
    mass_earth        DOUBLE PRECISION,
    radius_earth      DOUBLE PRECISION,
    composition_type  TEXT,        -- 'icy'|'rocky'|'mixed'
    resource_deposits JSONB
);
```

---

## Teil 3: Laufzeit-API (Map Service)

### 3.1 Galaxie-Übersicht (Zoom-Ebene 1)

```
GET /api/v1/galaxy/{galaxy_id}/stars
    ?bbox=x1,y1,z1,x2,y2,z2     # Bounding Box in Lichtjahren
    &lod=1                        # Level of Detail: 1=Übersicht, 3=Detail

Response: {
  stars: [{ id, x, y, z, star_type, color_hex, mass_solar }],
  total_in_bbox: int
}
```

**LOD-Strategie:**
- LOD 1 (Galaxie-Zoom): Sampling – 1 Stern pro Voxel (repräsentativ)
- LOD 2 (Sektor-Zoom): Alle Sterne in der Region
- LOD 3 (System-Zoom): Vollständige Sterndaten + Nebelinformation

### 3.2 FTLW-Feld abfragen

```
GET /api/v1/galaxy/{galaxy_id}/ftlw
    ?bbox=x1,y1,z1,x2,y2,z2
    &resolution=coarse            # coarse | fine

Response: {
  voxel_size_ly: 500,
  origin: {x, y, z},
  dimensions: {nx, ny, nz},
  values: [float32]              # flattened 3D array
}
```

### 3.3 FTL-Route berechnen

```
POST /api/v1/galaxy/{galaxy_id}/ftl-route
Body: {
  from_star_id: UUID,
  to_star_id: UUID,
  ship_ftl_rating: float         # Schiffsspezifischer FTL-Koeffizient
}

Response: {
  waypoints: [{x, y, z}],       # optimierter Pfad
  total_ftlw_cost: float,        # Summe FTLW entlang Route
  estimated_ticks: int,          # bei gegebener Schiffsgeschwindigkeit
  route_risk: float              # Piraterie-Wahrscheinlichkeit (TBD)
}
```

**Algorithmus:** A* auf dem FTLW-Voxelgrid
- Heuristik: euklidische Distanz × FTLW_vakuum
- Kantengewicht: FTLW im Ziel-Voxel × Voxel-Größe
- Optimierung: Hierarchisches Grid (grob → fein)

### 3.4 System-Scan (triggert JIT-Generierung)

```
POST /api/v1/galaxy/{galaxy_id}/stars/{star_id}/scan
     Authorization: Bearer <player_token>

→ Prüft ob planets_generated = true
→ Falls nein: JIT-Planetengenerator läuft (synchron, < 100ms)
→ Persistiert Ergebnis, setzt planets_generated = true

Response: {
  star: { ...vollständige Sterndaten... },
  planets: [{ ...planet + moons... }],
  asteroid_belts: [...],
  ftlw_local: float              # FTLW im System (für Sublicht-Bewegung)
}
```

### 3.5 FTLW-Override setzen (Spielmechanik – Lategame)

```
POST /api/v1/galaxy/{galaxy_id}/ftlw/override
Body: {
  voxel_x, voxel_y, voxel_z: int,
  structure_type: 'tunnel' | 'gate' | 'stabilizer',
  multiplier: float              # 0.1 = 90% Reduktion des FTLW
}
```

→ Schreibt in `ftlw_overrides`
→ Game Server invalidiert Redis-Cache für betroffene Voxel

---

## Teil 4: Caching-Strategie (Redis)

| Daten | Cache-Key | TTL | Invalidierung |
|---|---|---|---|
| FTLW-Chunks (base) | `ftlw:{galaxy_id}:{cx}:{cy}:{cz}` | 24h | Nie (statisch) |
| FTLW-Overrides | `ftlw_ov:{galaxy_id}:{vx}:{vy}:{vz}` | 1h | Bei Override-Änderung |
| Star-Region-Query | `stars:{galaxy_id}:{bbox_hash}:{lod}` | 5min | Nie (statisch) |
| Berechnete FTL-Route | `route:{from}:{to}:{ship_rating}` | 10min | Bei FTLW-Override |
| Planet-System (JIT) | `system:{star_id}` | 30min | Nie (nach Generierung statisch) |

---

## Teil 5: JIT-Planetengenerator (Detailablauf)

Wird aufgerufen wenn `planets_generated = false` und ein Spieler das System scannt.

```
Input:  star (Typ, Masse, Leuchtkraft, planet_seed)

1. RNG initialisieren mit planet_seed (deterministisch!)

2. Frostgrenze berechnen:
   d_frost = sqrt(L / L☉) * 2.7 AU

3. Titius-Bode-Bahnen generieren:
   d[n] = 0.4 + 0.3 * 2^n  (in AU, ab n=0)
   Anzahl Bahnen: 3–8 (abhängig von Sternmasse)

4. Für jede Bahn:
   a. Akkretionsmodell: verfügbare Masse in dieser Zone
   b. Würfeln ob Planet oder Gürtel entsteht
      (Gasriesen-Nachbar → erhöhte Gürtel-Wahrscheinlichkeit)
   c. Typ bestimmen: d < d_frost → Gestein, d >= d_frost → Gas/Eis
   d. Masse, Radius, Dichte berechnen
   e. Schwerkraft: g = G * M / r²
   f. Atmosphäre würfeln (gewichtet nach Typ, Schwerkraft, Distanz)
   g. Albedo aus Atmosphäre ableiten
   h. T_eq berechnen (Stefan-Boltzmann)
   i. T_surface = T_eq + Treibhauseffekt(Atmosphäre, Druck)
   j. Nutzfläche = f(T_surface, Druck, Atmosphäre)
   k. Biomasse = f(Atmosphäre, Nutzfläche)
   l. Ressourcen würfeln (nach Nebeltyp, Sterntyp, Planetentyp)

5. L4/L5-Asteroiden für Gasriesen hinzufügen

6. Monde generieren:
   - Gasriesen: 2–5 große + n kleine Monde (zirkumplanetare Scheibe)
   - Gesteinsplaneten: P=10% für Kollisionsmond

7. Alles in DB persistieren (Transaktion)
8. planets_generated = true
```

---

## Offene Punkte (für nächste Iteration)

| Thema | Priorität |
|---|---|
| Ressourcen-Verteilungsformel pro Element und Planetentyp | Hoch (vor AP4) |
| Naming-System für Sterne und Planeten | Mittel |
| Sichtbarkeit / Fog of War auf Kartenebene | Mittel (vor AP5) |
| Validierung: Maximale Größe des FTLW-Grids (Speicher) | Hoch (vor Impl.) |
| A*-Performance-Test bei 50.000 Sternen | Hoch (vor Impl.) |
