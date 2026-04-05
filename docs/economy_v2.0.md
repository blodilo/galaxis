# Galaxis — Wirtschaftsmodell v2.0

**Status:** Konzept · Löst economy_v1.0.md + production-mechanics_v1.1.md (Fabrik-Kapitel) ab
**Datum:** 2026-04-05
**Kernänderungen gegenüber v1.x:**
- Fabriken sind Güter (deployable items) — kein separater Build-Mechanismus mehr
- 5 einheitliche Anlagentypen (`extractor`, `refinery`, `plant`, `assembly_plant`, `construction_yard`)
- Stufe 4 produziert Anlagen als Güter; Stufe 5 = Werft + Mega-Assembler
- Domänenmodell explizit: Astronomie vs. Infrastruktur getrennt
- `construction`-Factory-Type entfällt; alles läuft über denselben Produktionsmechanismus

Fortbestehende Entscheidungen aus v1.x: Tick-Modell, Effizienz-Akkumulator,
Deposit-Modell (`amount`/`quality`/`max_mines`), Sensitivitätsklassen, Marktmodell 1A.

---

## 1. Domänenmodell

### Astronomie-Domäne (passiv, entdeckt, nicht gebaut)

Enthält alles, was physikalisch vor dem Spieler existiert:
`stars`, `planets`, `moons`, `asteroid_belts`

Deposits (`resource_deposits` JSONB) sind geologische Eigenschaften des Körpers —
sie gehören zur Astronomie-Domäne, nicht zur Infrastruktur.

Spieler *interagieren* mit dieser Domäne primär durch Information (Scans, Sensor-FOW).

### Infrastruktur-Domäne (aktiv, gebaut, betrieben)

Enthält alles, was ein Spieler errichtet oder deployt:

```
Installation (verankert an einem astronomischen Objekt oder im Orbit)
  └── hat Anlagen (deployed factories, aktiv produktiv)
  └── hat Lager (item stock, nach Sensitivitätsklasse)
  └── hat Routen (ein- und ausgehende Transportverbindungen)
```

Eine **Installation** ist der wirtschaftliche Ankerpunkt — was bisher `econ2_node` hieß.
Eine Installation kann planetar (Oberfläche), orbital oder stellar (z. B. Lagrange-Punkt) sein.

---

## 2. Fabriken als Güter — das Kernprinzip

Minen, Raffinerien, Werften etc. existieren in zwei Zuständen:

| Zustand | Beschreibung | Wo gespeichert |
|---|---|---|
| **Item** | Gut im Lager, transportierbar, handelbar | `item_stock` |
| **Installation** | Deployed, aktiv produktiv, ortsgebunden | `installations` |

**Deploy-Aktion:** Spieler wählt ein deployables Item im Lager → wählt Standort (Installation/Orbit/Planet) → das Item wird aus dem Lager entfernt, eine neue Anlage entsteht.

**Demontage (Post-MVP):** Anlage → Item mit Materialverlust (~70% Rückgewinnung).

### Deployable Item IDs

Konvention: `fac_<typ>_mk<level>` — z. B. `fac_extractor_mk1`, `fac_refinery_mk2`

```
fac_extractor_mk1      → deploybar als extractor Lv1
fac_refinery_mk1       → deploybar als refinery Lv1
fac_plant_mk1          → deploybar als plant Lv1
fac_assembly_plant_mk1 → deploybar als assembly_plant Lv1
fac_construction_yard_mk1 → deploybar als construction_yard Lv1
```

Schiffe sind ebenfalls Items nach dem Deploy-Prinzip, aber mit eigenem Bewegungs-Subsystem.

---

## 3. Die 5 Anlagentypen

| factory_type | Deutsch | Stufe | Produziert |
|---|---|---|---|
| `extractor` | Extraktionsanlage | 1 | Rohmaterialien aus Deposits |
| `refinery` | Raffinerie / Schmelze | 2 | Verarbeitete Grundstoffe |
| `plant` | Fertigungswerk | 3 | Komponenten, Chemikalien |
| `assembly_plant` | Montagewerk | 4 | Komplexe Güter + **deployable Tier 1–3 Anlagen** |
| `construction_yard` | Konstruktionsdock | 5 | Schiffe + **deployable Tier 4–5 Anlagen** |

### Constraints pro Typ

| factory_type | Standort | Besonderheit |
|---|---|---|
| `extractor` | Planetenoberfläche oder Orbit | Rohstoffspezifisch, Slot-Limit per Deposit |
| `refinery` | Oberfläche oder Orbit | — |
| `plant` | Orbit bevorzugt | Mikrogravitation → bessere Qualität (+η-Bonus orbital) |
| `assembly_plant` | Orbit | — |
| `construction_yard` | Orbit | Slot-Limit nach Schiffs-/Anlagengröße |

---

## 4. Minimale Gütertabelle (MVP)

Minimaler geschlossener Kreislauf — alle späteren Güter sind Erweiterungen.

### Stufe 1 — Rohstoffe (extrahiert)

| Gut-ID | Deutsch | Quelle | Klasse |
|---|---|---|---|
| `iron` | Eisenerz | Felsplanet | I |
| `silicon` | Silikate | Felsplanet | I |
| `titanium` | Titan | Felsplanet | I |
| `rare_earth` | Seltene Erden | Felsplanet (alt) | II |
| `helium_3` | Helium-3 | Gasriese | II |
| `hydrogen` | Wasserstoff | Gasriese / Eismond | II |

### Stufe 2 — Grundstoffe (Raffinerie-Output)

| Gut-ID | Deutsch | Klasse |
|---|---|---|
| `steel` | Stahl | I |
| `titansteel` | Titanstahl | I |
| `semiconductor` | Halbleiter-Wafer | III |
| `fusion_fuel` | Fusionstreibstoff | II |

### Stufe 3 — Komponenten (Plant-Output)

| Gut-ID | Deutsch | Klasse |
|---|---|---|
| `component` | Basiskomponente | III |
| `circuit_board` | Schaltkreis | III |

### Stufe 4 — Assemblierte Güter (Assembly-Plant-Output)

Das Montagewerk produziert zwei Kategorien:

**4a — Deployable Anlagen (Tier 1–3)**

| Gut-ID | Deployable | Deutsch |
|---|---|---|
| `fac_extractor_mk1` | ✓ | Extraktionsanlage Mk1 |
| `fac_refinery_mk1` | ✓ | Raffinerie Mk1 |
| `fac_plant_mk1` | ✓ | Fertigungswerk Mk1 |
| `freighter_sys_mk1` | ✓ (Schiff) | System-Frachter Mk1 |

**4b — Schwere Tier-4-Komponenten** *(Inputs für das Konstruktionsdock)*

| Gut-ID | Deutsch | Funktion im Construction Yard |
|---|---|---|
| `drive_unit` | Antriebseinheit | Haupt-Triebwerk des Docks |
| `reactor_module` | Reaktormodul | Energieversorgung |
| `structural_frame` | Strukturrahmen | Tragstruktur |

Das Konstruktionsdock selbst wird aus diesen drei Komponenten ebenfalls
**im Assembly Plant assembliert** — es ist das einzige Tier-5-Item das ein
Assembly Plant herstellen kann.

| Gut-ID | Deployable | Deutsch |
|---|---|---|
| `fac_construction_yard_mk1` | ✓ | Konstruktionsdock Mk1 |

### Stufe 5 — Konstruktionsdock-Output

Das Konstruktionsdock produziert ausschließlich Schiffe und höhere Assembly Plants.

| Gut-ID | Deployable | Deutsch |
|---|---|---|
| `fac_assembly_plant_mk1` | ✓ | Montagewerk Mk1 (Upgrade/Ersatz) |
| `frigate_mk1` | ✓ (Schiff) | Fregatte Mk1 |
| `freighter_ftl_mk1` | ✓ (Schiff) | FTL-Frachter Mk1 |

---

## 5. Rezepttabellen

### 5a — Refinery (Stufe 2)

| Rezept-ID | Inputs | Output | Ticks | η |
|---|---|---|---|---|
| `ref_steel` | iron×5 | steel×5 | 1 | 0.90 |
| `ref_titansteel` | iron×2 + titanium×1 | titansteel×2 | 2 | 0.90 |
| `ref_semiconductor` | silicon×1 + rare_earth×1 | semiconductor×2 | 3 | 0.85 |
| `ref_fusion_fuel` | helium_3×2 + hydrogen×3 | fusion_fuel×4 | 1 | 0.92 |

### 5b — Plant (Stufe 3)

| Rezept-ID | Inputs | Output | Ticks | η |
|---|---|---|---|---|
| `plant_component` | titansteel×2 + semiconductor×1 | component×3 | 4 | 0.88 |
| `plant_circuit` | semiconductor×4 | circuit_board×2 | 5 | 0.85 |

### 5c — Assembly Plant (Stufe 4)

Produziert deployable Anlagen-Items (Tier 1–3), schwere Tier-4-Komponenten
und das Konstruktionsdock selbst.

**Deployable Anlagen:**

| Rezept-ID | Inputs | Output | Ticks | η |
|---|---|---|---|---|
| `asm_extractor_mk1` | steel×10 + component×2 | fac_extractor_mk1×1 | 6 | 1.0 |
| `asm_refinery_mk1` | titansteel×15 + component×5 | fac_refinery_mk1×1 | 10 | 1.0 |
| `asm_plant_mk1` | titansteel×20 + component×8 + circuit_board×4 | fac_plant_mk1×1 | 15 | 1.0 |
| `asm_freighter_sys` | titansteel×12 + component×6 + circuit_board×3 + fusion_fuel×4 | freighter_sys_mk1×1 | 20 | 1.0 |

**Schwere Tier-4-Komponenten:**

| Rezept-ID | Inputs | Output | Ticks | η |
|---|---|---|---|---|
| `asm_drive_unit` | titansteel×8 + component×4 + fusion_fuel×3 | drive_unit×1 | 8 | 1.0 |
| `asm_reactor_module` | titansteel×6 + component×5 + circuit_board×3 | reactor_module×1 | 10 | 1.0 |
| `asm_structural_frame` | titansteel×12 + component×3 | structural_frame×1 | 6 | 1.0 |

**Konstruktionsdock** *(assembliert aus Tier-4-Komponenten)*:

| Rezept-ID | Inputs | Output | Ticks | η |
|---|---|---|---|---|
| `asm_construction_yard_mk1` | drive_unit×2 + reactor_module×3 + structural_frame×5 | fac_construction_yard_mk1×1 | 40 | 1.0 |

> η=1.0 bei Anlagen-Items und Großkomponenten: Output ist immer ganzzahlig.

### 5d — Construction Yard (Stufe 5)

Produziert ausschließlich Schiffe und höhere Assembly Plants.

| Rezept-ID | Inputs | Output | Ticks | η |
|---|---|---|---|---|
| `cy_assembly_plant_mk1` | drive_unit×1 + reactor_module×2 + structural_frame×3 | fac_assembly_plant_mk1×1 | 30 | 1.0 |
| `cy_frigate_mk1` | titansteel×20 + component×12 + circuit_board×6 + fusion_fuel×8 | frigate_mk1×1 | 25 | 1.0 |
| `cy_freighter_ftl` | drive_unit×2 + reactor_module×1 + titansteel×15 + circuit_board×8 + fusion_fuel×15 | freighter_ftl_mk1×1 | 45 | 1.0 |

---

## 6. Produktionskreislauf

```
DEPOSITS (Planets/Moons)
  │
  ▼ extractor (max_rate × quality / Tick)
STUFE 1: iron, silicon, titanium, rare_earth, helium_3, hydrogen
  │
  ▼ refinery
STUFE 2: steel, titansteel, semiconductor, fusion_fuel
  │
  ▼ plant
STUFE 3: component, circuit_board
  │
  ▼ assembly_plant
STUFE 4a: fac_extractor_mk1   →  deploy → mehr extractors
          fac_refinery_mk1    →  deploy → mehr refineries
          fac_plant_mk1       →  deploy → mehr plants
          freighter_sys_mk1   →  deploy → System-Transport

STUFE 4b: drive_unit          ─┐
          reactor_module       ├→ assembly_plant assembliert →
          structural_frame    ─┘

          fac_construction_yard_mk1  →  deploy → Construction Yard
  │
  ▼ construction_yard
STUFE 5: fac_assembly_plant_mk1  →  deploy → mehr/größere Assembly Plants
         frigate_mk1             →  deploy → Kampfschiff
         freighter_ftl_mk1       →  deploy → Interstellarer Transport
```

---

## 7. Bootstrap — Kolonieschiff

Das Kolonieschiff deployt sofort vier Anlagen und bringt Starter-Material mit.
Der Spieler muss das Konstruktionsdock selbst erarbeiten.

### Mitgelieferte Anlagen (deployed, Lv1)

| Anlage | Typ | Anmerkung |
|---|---|---|
| Extraktionsanlage | `extractor` | Für Eisenvorkommen (automatisch gewählt) |
| Raffinerie | `refinery` | Schmelzt Eisen → Stahl und Titan → Titanstahl |
| Fertigungswerk | `plant` | Produziert Komponenten + Schaltkreise |
| Montagewerk | `assembly_plant` | Kann sofort neue Extractors + Refineries bauen |

> Das Montagewerk kommt per Kolonieschiff — es ist das einzige Item das „geschenkt" wird.
> Alle weiteren Montagewerke und das erste Konstruktionsdock müssen erarbeitet werden.

### Mitgelieferte Materialien

| Gut | Menge | Zweck |
|---|---|---|
| `steel` | 200 E | Startpuffer für erste Bauaufträge |
| `titansteel` | 50 E | Puffer für erste Raffinerie-Erweiterung |

### Nicht enthalten (muss erarbeitet werden)

| Was | Wie |
|---|---|
| Mehr Extractors | Assembly Plant → `fac_extractor_mk1` deployen |
| Raffinerie für Halbleiter | Assembly Plant → `fac_refinery_mk1` deployen + Silikate/RE fördern |
| Konstruktionsdock | Assembly Plant → erst `fac_plant_mk1` bauen, dann `cy_construction_yard_mk1` |
| Erstes Schiff (Fregatte) | Construction Yard → `cy_frigate_mk1` |

### Optimale Bootstrap-Sequenz

```
Tick 1–2:   Extractor → iron, titanium
            Refinery  → steel, titansteel

Tick 3–8:   Assembly Plant → fac_extractor_mk1 (silicon + rare_earth) [6 Ticks]
            Deploy → Silicon/Rare-Earth-Extractor aktiv

Tick 9–18:  Assembly Plant → fac_refinery_mk1 (Halbleiter-Raffinerie) [10 Ticks]
            Deploy → Halbleiter-Raffinerie aktiv
            Refinery → semiconductor läuft

Tick 19–22: Extractor → helium_3, hydrogen starten (fac_extractor_mk1 bauen)
            Plant → component, circuit_board

Tick 23–30: Assembly Plant → fac_refinery_mk1 (Fusion Fuel) [10 Ticks]
            Deploy → Fusion-Fuel-Raffinerie aktiv

            Parallel: Assembly Plant → drive_unit [8], reactor_module [10], structural_frame [6]

Tick 31–70: Assembly Plant → drive_unit×2 + reactor_module×3 + structural_frame×5
            → fac_construction_yard_mk1 [40 Ticks] (sobald Komponenten verfügbar)

Tick 71+:   Deploy Construction Yard
            Construction Yard → frigate_mk1 [25 Ticks]
            KREISLAUF GESCHLOSSEN
```

---

## 8. Selbst-Reproduktion der Wirtschaft

Das Modell ist selbst-reproduzierend: Jede Anlage kann durch die eigene
Produktionskette ersetzt und erweitert werden.

| Anlage verloren/erschöpft | Reparaturpfad |
|---|---|
| Extractor | Assembly Plant → `fac_extractor_mk1` → deploy |
| Refinery | Assembly Plant → `fac_refinery_mk1` → deploy |
| Plant | Assembly Plant → `fac_plant_mk1` → deploy |
| Assembly Plant | Construction Yard → `cy_assembly_plant_mk1` → deploy |
| Construction Yard | Assembly Plant → drive_unit + reactor_module + structural_frame → `fac_construction_yard_mk1` → deploy |

Ist *das letzte* Construction Yard zerstört: Assembly Plant kann ein neues
bauen — solange Tier-4-Komponenten produziert werden können.
Sind auch alle Assembly Plants weg: Totalverlust, Neustart mit Kolonieschiff.

---

## 9. Offene Punkte

| Thema | Priorität | Notiz |
|---|---|---|
| Deploy-API + UI | Hoch | POST /installations, Item aus Stock → aktive Anlage |
| Upgrade-Mechanik (Mk1 → Mk2) | Mittel | Anlage in-place upgraden vs. neue deployen |
| Demontage-Mechanik | Mittel | ~70% Materialrückgewinnung |
| Aufzug (Oberfläche → Orbit) | Mittel | Flaschenhals-Mechanik aus production-mechanics_v1.1 übernehmen |
| Sensitivitätsklassen-Enforcement | Mittel | Aus production-mechanics_v1.1 übernehmen |
| Gas Harvester als Extractor-Subtyp | Niedrig | Orbital-only, eigene Rezepte für Gas-Deposits |
| Extractor Stufe 2–5 (Upgrade) | Niedrig | Mk2–Mk5 über Construction Yard |
| Marktmodul / Handelsrouten | Niedrig | Aus economy_v1.0.md übernehmen wenn Schiffe laufen |
| NPC-Spawn für erstes Construction Yard | Offen | Alternative zum "geschenkt"-Bootstrap? |

---

## 10. Abgrenzung zu bestehenden Dokumenten

| Dokument | Verhältnis zu v2.0 |
|---|---|
| `economy_v1.0.md` | Grundprinzipien (Markt, Tick-Modell, Ressourcenregionalität) weiterhin gültig |
| `production-mechanics_v1.1.md` | Deposit-Modell (Abschnitte 6a/6b), Sensitivitätsklassen, Einheitensystem, Effizienz-Akkumulator weiterhin gültig |
| `production-mechanics_v1.1.md` | Anlagen-Kapitel (7) wird durch dieses Dokument abgelöst |
| `econ2_recipes_v1.0.yaml` | Wird durch Rezepte in Abschnitt 5 dieses Dokuments ersetzt (neue factory_types) |

---

*Erstellt: 2026-04-05 — Wirtschaftsmodell v2.0 · Fabriken als Güter, 5 Anlagentypen, Stufe-4/5-Produktionskette*
