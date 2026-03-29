# Galaxis — Produktionsmechanik v1.1

**Status:** Deposit-Modell überarbeitet · Implementation ausstehend
**Abhängigkeiten:** `economy_v1.0.md`, `game-params_v1.9.yaml`, Planetengenerierung, Fraktionssystem
**Datum:** 2026-03-29
**Änderungen gegenüber v1.0:**
- Deposit-Modell neu definiert (Abschnitt 6a)
- Himmelskörper-Status (`untouched`) eingeführt (Abschnitt 6b)
- Bergwerk-Mechanik präzisiert: Förderformel, `max_rate` in Facility (Abschnitt 6, Bergwerk)
- `max_mines`: unabhängig von `quality`, normalverteilt 1–10

---

## 1. Überblick

Dieses Dokument spezifiziert:
- Das **Einheitensystem** (Güter, Masse pro Einheit, Sensitivitätsklasse)
- Das **Rezeptmodell** (Inputs/Outputs, Effizienzakkumulation)
- Das **Deposit-Modell** (Rohstoffvorkommen, Zugänglichkeit, Förderformel)
- Den **Himmelskörper-Status** (`untouched`-Flag)
- Die **7 Produktionsanlagen** des minimalen Kreislaufs
- Das **Lagermodell** (gemeinsames Stationslager, nach Klasse getrennt)
- Den **Bootstrap** (Kolonieschiff-Starterpaket)
- Ein Konzept für die **Pipeline-UI**

Alle balancierbaren Parameter leben in `game-params_v1.9.yaml` unter `production:`.

---

## 2. Einheitensystem

Alle Güter werden in abstrakten **Einheiten (E)** gehandelt und gelagert.
Die physikalische Masse pro Einheit bestimmt Transportkapazitäten und Lagervolumen.

| Güterklasse       | Masse / E   | Beispiele                                              |
|-------------------|-------------|--------------------------------------------------------|
| Bulk              | 10.000 t    | Eisenerz, Nickel, Titan, Silikate, Stahl, Titanstahl  |
| Legierung / Chem. | 100 t       | Vanadium, Chrom, Seltene Erden, Kühlmittel            |
| Präzision         | 1 t         | Platin-Gruppe, Halbleiter-Wafer, Exotische Materie    |
| Komponenten       | 0,1 t (Stk) | Mikrochips, Navigationscomputer, Exotik-Prozessor     |
| Flüssiggas        | 5.000 t     | He-3, Wasserstoff, Fusionstreibstoff                  |

---

## 3. Sensitivitätsklassen

Jede Klasse definiert Lageranforderungen, Transportregeln und Kampfverhalten.
Die Klassen sind nach oben erweiterbar (z. B. Klasse V für exotische Antimaterie).

### Klasse I — Bulk / Inert

| Dimension        | Regel                                                              |
|------------------|--------------------------------------------------------------------|
| **Lager**        | Offen / drucklos. Lagermodul Lv1+ genügt.                        |
| **Transport**    | Jeder Standard-Frachter. Keine Sondercontainer.                  |
| **Kampf**        | Schiff zerstört → Ladung verloren. Strahlung / EMP: kein Verlust.|
| **Degradation**  | Keine.                                                            |

Güter: Eisenerz, Nickel, Titan, Silikate, Wassereis, Chrom, Stahl, Titanstahl,
Chrom-Legierung, Keramik-Komposit, Platin-Gruppe, Waffensystem (kinetisch)

---

### Klasse II — Reguliert / Kontrolliert

| Dimension        | Regel                                                                                   |
|------------------|-----------------------------------------------------------------------------------------|
| **Lager**        | Versiegelt / temperaturkontrolliert. Lagermodul „Sealed" erforderlich.                |
| **Transport**    | Zertifizierter Frachter empfohlen. Ohne: Degradation `wrong_transport_decay_per_tick`. |
| **Kampf**        | Energiewaffen-Treffer: Leck-Risiko (Gasverlust). Detonation je nach Gut (s. u.).      |
| **Degradation**  | Nur bei Falschlagerung: `wrong_storage_degradation_pct` / Tick.                        |

Güter: He-3, Wasserstoff, Fusionstreibstoff, Biosynth-Grundstoff, Organika, Kühlmittel,
Seltene Erden, Fusionsreaktor, Ionentriebwerk, Lebenserhaltung, Waffensystem (Energie)

---

### Klasse III — Sensitiv / Fragil

| Dimension        | Regel                                                                                       |
|------------------|---------------------------------------------------------------------------------------------|
| **Lager**        | Strahlungsgeschirmt, vibrationsdämpfend, temperaturkontrolliert. Lagermodul „Shielded".   |
| **Transport**    | Faraday-abgeschirmter Frachtraum empfohlen. Ohne: hohe Degradation per Tick.              |
| **Kampf**        | Strahlungstreffer / EMP: zufälliger Verlust `[radiation_loss_min … radiation_loss_max]`.  |
| **Degradation**  | Nur bei Falschlagerung: `wrong_storage_degradation_pct` / Tick.                           |

Güter: Halbleiter-Wafer, Mikrochips, Navigationscomputer, Exotik-Prozessor,
Phasen-Schild, Sprung-Antrieb, Basiskomponente (Elektronikanteil)

---

### Klasse IV — Segregiert / Isoliert

| Dimension        | Regel                                                                                                         |
|------------------|---------------------------------------------------------------------------------------------------------------|
| **Lager**        | Eigenes isoliertes Magazin. **Keine Mischung** mit anderen Gütern. Lagermodul „Segregated".                 |
| **Transport**    | Ausschließlich dedizierter Gefahrgutfrachter. Kein Standard-Frachter erlaubt.                               |
| **Kampf**        | Schiffszerstörung → Detonationsrisiko `detonation_chance`. Strahlungstreffer: sekundäre Explosion möglich.  |
| **Degradation**  | Nur bei Falschlagerung: sehr hohe Degradation + Detonationsrisiko pro Tick.                                 |

Güter: Spaltmaterial (Uran/Thorium angereichert), instabile Exotika, Antimateriekapseln

---

## 4. Gütertabelle (vollständig, minimaler Kreislauf)

| Gut                  | Stufe | Klasse | Masse/E   | Produziert durch   |
|----------------------|-------|--------|-----------|--------------------|
| Eisenerz             | 1     | I      | 10.000 t  | Bergwerk           |
| Silikate             | 1     | I      | 10.000 t  | Bergwerk           |
| Titan                | 1     | I      | 10.000 t  | Bergwerk           |
| Seltene Erden        | 1     | II     | 100 t     | Bergwerk           |
| Uran / Thorium (roh) | 1     | II     | 100 t     | Bergwerk           |
| He-3                 | 1     | II     | 5.000 t   | Gas-Harvester      |
| Wasserstoff          | 1     | II     | 5.000 t   | Gas-Harvester      |
| Stahl                | 2     | I      | 10.000 t  | Schmelze           |
| Titanstahl           | 2     | I      | 10.000 t  | Schmelze           |
| Halbleiter-Wafer     | 2     | III    | 1 t       | Raffinerie         |
| Fusionstreibstoff    | 2     | II     | 5.000 t   | Raffinerie         |
| Spaltmaterial        | 2     | IV     | 100 t     | Raffinerie (Klasse IV Modul) |
| Biosynth-Grundstoff  | 2     | II     | 1 t       | Bioreaktor         |
| Basiskomponente      | 3     | III    | 0,1 t     | Präzisionsfabrik   |
| Navigationscomputer  | 3     | III    | 0,1 t     | Präzisionsfabrik   |
| Fusionsreaktor Lv1   | 3     | II     | 0,1 t     | Präzisionsfabrik   |
| Ionentriebwerk       | 3     | II     | 0,1 t     | Präzisionsfabrik   |
| Sprung-Antrieb       | 3     | III    | 0,1 t     | Präzisionsfabrik   |

---

## 5. Rezeptmodell

### Formel

```
output_float  += nominal_output × η_facility
output_actual  = floor(output_float)
output_float  -= output_actual          # Rest akkumuliert bis zum nächsten Batch
```

- `nominal_output`: definierter Rezept-Output (Einheiten)
- `η_facility`: Effizienzfaktor der Anlage (0–1, steigt mit Level und Forschung)
- Ausgabe erfolgt am Ende eines Batches (nicht pro Tick)
- Fraktionen akkumulieren über Batches (kein Verlust, kein Rounding-Fehler)

### Minimale Rezepte (vollständiger Kreislauf)

| Rezept              | Input                                          | Nominal-Output | η Lv1 | Fabrik           | Ticks |
|---------------------|------------------------------------------------|----------------|-------|------------------|-------|
| Stahl               | Eisenerz × 5                                   | Stahl × 5      | 0,90  | Schmelze         | 1     |
| Titanstahl          | Eisenerz × 3 + Titan × 1                       | Titanstahl × 3 | 0,90  | Schmelze         | 1     |
| Halbleiter-Wafer    | Silikate × 1 + Seltene Erden × 1               | Wafer × 2      | 0,85  | Raffinerie       | 3     |
| Fusionstreibstoff   | He-3 × 2 + Wasserstoff × 3                     | Treibstoff × 4 | 0,92  | Raffinerie       | 1     |
| Basiskomponente     | Titanstahl × 2 + Wafer × 1                     | Basis × 3      | 0,88  | Präzisionsfabrik | 4     |
| Navigationscomputer | Wafer × 6 + Platin-Gruppe × 2                  | NavComp × 1    | 0,85  | Präzisionsfabrik | 10    |
| Fusionsreaktor Lv1  | Treibstoff × 5 + Titanstahl × 8 + Wafer × 4   | Reaktor × 1    | 0,82  | Präzisionsfabrik | 12    |
| Ionentriebwerk      | Treibstoff × 3 + Titan × 4 + Wafer × 3        | Triebwerk × 1  | 0,85  | Präzisionsfabrik | 8     |

---

## 6a. Deposit-Modell

### Datenstruktur

Rohstoffvorkommen werden in `planets.resource_deposits` (JSONB) gespeichert.
Dasselbe Format gilt für `moons.resource_deposits`.

```json
{
  "iron_ore": {
    "amount":    40000,
    "quality":   0.80,
    "max_mines": 4
  },
  "silicates": {
    "amount":    28000,
    "quality":   0.55,
    "max_mines": 2
  }
}
```

### Felder

| Feld        | Typ    | Wer schreibt       | Wann ändert es sich            |
|-------------|--------|--------------------|--------------------------------|
| `amount`    | float  | Generator (lazy init bei Survey) | Sinkt mit jedem Mine-Tick |
| `quality`   | float  | Generator          | Nie (planetengebundene Geologie) |
| `max_mines` | int    | Generator          | Nie (geographische Zugänglichkeit) |

### Semantik der Felder

**`amount`** — aktueller Vorrat in Einheiten.
- Startwert: `base_units × quality` (aus `game-params`)
- Mining decrementiert direkt: `amount -= extracted_per_tick`
- Bei `amount ≤ 0`: Deposit erschöpft, Bergwerk pausiert

**`quality`** — geologische Qualität des Vorkommens (0.0–1.0).
- Planetengebundener Modifier für die Förderrate
- Einfluss auf Ertrag: `extracted_per_tick = facility.max_rate × quality`
- Hohe quality → mehr Ertrag pro Mine-Tick; kein Einfluss auf Slot-Anzahl

**`max_mines`** — maximale Anzahl gleichzeitig aktiver Bergwerke auf diesem Deposit.
- Modelliert die **Zugänglichkeit** des Geländes (Topographie, Tiefe, Geologie)
- Unabhängig von `quality` — auch ein reiches Vorkommen kann schwer zugänglich sein
- Normalverteilt, ganzzahlig, clamp [1, 10] (Parameter in `game-params`)
- Bestimmt die Slot-Prüfung in `mineSlotAvailable()`

### Förderformel

```
extracted_per_tick = facility.max_rate × deposit.quality
```

- `facility.max_rate`: Förderrate der Mine in E/Tick, in `econ2_facilities.config` gespeichert
- `deposit.quality`: geologischer Modifier des Planeten (0–1)
- Startwert von `max_rate` bei Lv1: aus `game-params.production.mine.base_max_rate`
- `max_rate` steigerbar durch Tech-Upgrades (spielergebunden, nicht planetengebunden)

### Lazy Initialisierung

`resource_deposits` wird beim **ersten Survey** des Planeten angelegt (nicht beim Generieren).
Für unentdeckte Planeten existiert kein Eintrag — das ist Absicht (→ `untouched`-Flag).

Für Planeten mit Survey aber ohne Eintrag: `EnsureDeposits()` legt den Eintrag
aus `game-params` + `quality` an.

### Deposit-Warnungen

| Zustand        | Schwelle                         | Aktion                              |
|----------------|----------------------------------|-------------------------------------|
| Warnung        | `amount < amount_start × 0.20`   | Gelbes Badge, einmaliger Log        |
| Kritisch       | `amount < amount_start × 0.05`   | Rotes Badge, SSE-Push an Client     |
| Erschöpft      | `amount ≤ 0`                     | Bergwerk pausiert, kein weiterer Abbau |

---

## 6b. Himmelskörper-Status

### `untouched`-Flag

Jeder Planet und jeder Mond hat das Feld `untouched BOOLEAN DEFAULT TRUE`.

| Zustand         | Bedeutung                                                          |
|-----------------|--------------------------------------------------------------------|
| `untouched = TRUE`  | Kein Spieler hat diesen Körper bisher gescannt. Alle Eigenschaften (Deposits, Geologie, Ressourcen) können noch vom System nachträglich angepasst werden. |
| `untouched = FALSE` | Mindestens ein Spieler hat gescannt. Eigenschaften sind fixiert. Änderungen würden zu Inkonsistenz zwischen Spielerwissen und Realität führen. |

### Wann wird `untouched` auf `FALSE` gesetzt?

Beim **ersten Scan** eines Spielers:
```sql
UPDATE planets SET untouched = FALSE WHERE id = $1;
```

Nicht beim bloßen Sehen (FOW-Sichtbarkeit) — erst beim aktiven Survey.

### Bedeutung für den Generator

Solange `untouched = TRUE`:
- Deposits können nachträglich generiert/angepasst werden (z. B. bei Generator-Updates)
- Geologische Parameter können korrigiert werden ohne Spieler zu benachteiligen
- Kein Survey-Ergebnis wurde je an einen Spieler gesendet

---

## 7. Produktionsanlagen

### Übersicht

| # | Anlage           | Ort             | Funktion                            | Levels |
|---|------------------|-----------------|-------------------------------------|--------|
| 1 | Bergwerk         | Oberfläche      | Extraktion eines spezifischen Erzes | Lv1–5  |
| 2 | Aufzug           | Oberfläche↔Orbit| Logistik-Flaschenhals               | Lv1–5  |
| 3 | Schmelze         | Oberfläche/Orbit| Erze → Metalle, Legierungen         | Lv1–5  |
| 4 | Raffinerie       | Orbit (bevorzugt)| Mineralien → Halbleiter, Chemikalien| Lv1–5  |
| 5 | Präzisionsfabrik | **Nur orbital** | Komponenten (Chips, Triebwerke …)   | Lv1–5  |
| 6 | Assembler        | Orbit           | Baut Stationsmodule und Anlagen     | Lv1–3  |
| 7 | Werft            | Orbit           | Baut Schiffe                        | Lv1–5  |

---

### Bergwerk

- **Rohstoffspezifisch**: Jedes Bergwerk fördert genau einen Rohstoff (abhängig vom Planetenvorkommen).
- Pro Deposit kann es maximal `deposit.max_mines` gleichzeitig aktive Bergwerke geben.
- Förderertrag pro Tick: `facility.max_rate × deposit.quality`
- `max_rate` startet bei `base_max_rate` (Lv1) und steigt durch Level-Upgrades und Technologie.
- `max_rate` ist **spielergebunden** — zwei Spieler auf demselben Deposit haben unabhängige Raten.

| Level | base_max_rate (E/Tick) | Baukosten (Assembler) |
|-------|------------------------|-----------------------|
| Lv1   | aus game-params        | Kolonieschiff (Starter) |
| Lv2   | × 2,0                  | 30 E Stahl + 5 E Basiskomponente |
| Lv3   | × 3,5                  | 60 E Titanstahl + 15 E Basiskomponente |
| Lv4   | × 5,5                  | 100 E Titanstahl + 30 E Basiskomponente |
| Lv5   | × 8,0                  | 150 E Titanstahl + 50 E Basiskomponente + 10 E NavComp |

---

### Aufzug

- Kapazität: `level × capacity_per_level` Einheiten / Tick (aus `economy:elevator:`).
- Ohne Aufzug: Shuttle-Fallback (langsam, teuer, konfiguriert in `economy:elevator:shuttle_*`).
- Blockiert alle Bergwerke gemeinsam (geteilter Flaschenhals).

---

### Schmelze

| Level | η    | Output / Tick | Baukosten |
|-------|------|---------------|-----------|
| Lv1   | 0,90 | 5 E / Tick    | Kolonieschiff (Starter) |
| Lv2   | 0,92 | 10 E / Tick   | 20 E Titanstahl + 5 E Basiskomponente |
| Lv3   | 0,94 | 18 E / Tick   | 40 E Titanstahl + 12 E Basiskomponente |
| Lv4   | 0,96 | 30 E / Tick   | 80 E Titanstahl + 25 E Basiskomponente |
| Lv5   | 0,98 | 45 E / Tick   | 150 E Titanstahl + 50 E Basiskomponente |

---

### Raffinerie

| Level | η    | Output / Tick | Baukosten |
|-------|------|---------------|-----------|
| Lv1   | 0,85 | 2 E / Tick    | 15 E Titanstahl + 3 E Basiskomponente |
| Lv2   | 0,87 | 5 E / Tick    | 30 E Titanstahl + 8 E Basiskomponente |
| Lv3   | 0,90 | 10 E / Tick   | 60 E Titanstahl + 18 E Basiskomponente |
| Lv4   | 0,92 | 18 E / Tick   | 120 E Titanstahl + 35 E Basiskomponente |
| Lv5   | 0,95 | 28 E / Tick   | 200 E Titanstahl + 60 E Basiskomponente |

---

### Präzisionsfabrik *(nur orbital)*

- Setzt Mikrogravitation voraus — kann nicht auf Planetenoberfläche gebaut werden.
- Produziert alle Stufe-3-Komponenten.

| Level | η    | Slots | Baukosten |
|-------|------|-------|-----------|
| Lv1   | 0,88 | 1     | 25 E Titanstahl + 10 E Wafer |
| Lv2   | 0,90 | 2     | 50 E Titanstahl + 20 E Wafer + 5 E Basis |
| Lv3   | 0,92 | 3     | 100 E Titanstahl + 40 E Wafer + 15 E Basis |
| Lv4   | 0,94 | 4     | 180 E Titanstahl + 70 E Wafer + 30 E Basis |
| Lv5   | 0,96 | 6     | 300 E Titanstahl + 120 E Wafer + 60 E Basis + 10 E NavComp |

---

### Assembler

- Baut alle Stationsmodule und planetaren Anlagen.
- Lv1 kommt per Kolonieschiff oder wird von der Werft gebaut.

| Level | Slots | Baukosten |
|-------|-------|-----------|
| Lv1   | 1     | Kolonieschiff oder 40 E Stahl + 8 E Basiskomponente (Werft) |
| Lv2   | 2     | 80 E Titanstahl + 20 E Basiskomponente |
| Lv3   | 3     | 160 E Titanstahl + 50 E Basiskomponente + 10 E Fusionsreaktor |

---

### Werft

| Level | Slots | Max Schiffsklasse | Baukosten |
|-------|-------|-------------------|-----------|
| Lv1   | 1     | Fregatte          | 60 E Titanstahl + 20 E Basiskomponente |
| Lv2   | 2     | Kreuzer           | 120 E Titanstahl + 45 E Basis + 5 E NavComp |
| Lv3   | 4     | Schlachtschiff    | 250 E Titanstahl + 100 E Basis + 15 E NavComp + 5 E Reaktor |

---

## 8. Lagermodell

### Gemeinsames Stationslager, getrennt nach Sensitivitätsklasse

Jede Station hat **ein gemeinsames Lager** pro Sensitivitätsklasse (nicht pro Anlage).
Alle Anlagen derselben Station teilen diese Puffer.

```
Station "Kepler-7b Orbit"
  ├── Lager Klasse I    [0 / 500 E]   — Standard-Modul
  ├── Lager Klasse II   [0 / 200 E]   — Sealed-Modul
  ├── Lager Klasse III  [0 / 100 E]   — Shielded-Modul
  └── Lager Klasse IV   [0 / 50 E]    — Segregated-Magazin (separates Gebäude!)
```

- Lagerkapazität pro Klasse wächst durch Lagermodul-Upgrades.
- Produktionsanlage pausiert wenn: Eingangslager leer **oder** Ausgangslager voll (2C-Hybrid).
- Klasse-IV-Magazin ist physisch getrennt (eigenes Stationsmodul, nicht stapelbar mit anderen).

### Falschlagerung

Wird ein Gut in ein Lager niedrigerer Klasse gelegt (z. B. Klasse-III-Gut in Klasse-I-Lager):
- Degradation pro Tick: `wrong_storage_degradation_pct` (aus `game-params`)
- UI-Warnung: sofortige rote Markierung + Tick-Zähler bis Totalverlust

---

## 9. Bootstrap — Kolonieschiff-Starterpaket

Das Kolonieschiff bringt einen funktionsfähigen Grundstock. Spieler müssen die
Produktionskette selbst hochziehen; nichts davon ist im laufenden Spiel kostenlos.

```
Kolonieschiff-Fracht:
  Anlagen (vorgebaut, sofort einsatzbereit):
    ✓ Bergwerk Lv1       × 1   (für Eisenerz-Vorkommen)
    ✓ Aufzug Lv1         × 1
    ✓ Schmelze Lv1       × 1
    ✓ Assembler Lv1      × 1

  Materialreserve:
    ✓ 200 E Stahl        (Klasse I)
    ✓  50 E Titanstahl   (Klasse I)

  NICHT enthalten:
    ✗ Raffinerie         → erster eigener Bau (benötigt Titanstahl)
    ✗ Präzisionsfabrik   → zweiter eigener Bau (benötigt Titanstahl + Wafer)
    ✗ Werft              → dritter eigener Bau
```

### Bootstrapsequenz (optimaler Pfad)

```
Tick 1–3:   Bergwerk → Eisenerz + Titan
            Schmelze → Titanstahl
Tick 4–8:   Bergwerk Lv1 (Silikate + Seltene Erden) bauen
            Assembler → Raffinerie Lv1
Tick 9–15:  Raffinerie → Halbleiter-Wafer
            Assembler → Präzisionsfabrik Lv1
Tick 16–25: Präzisionsfabrik → Basiskomponenten
            Assembler → Werft Lv1 + Bergwerk Upgrades
Tick 26+:   Werft → Systemfrachter → Logistik und Expansion
            KREISLAUF GESCHLOSSEN
```

---

## 10. Vollständiger Kreislauf — Flussdiagramm

```
[Planetoberfläche]
  Bergwerk(Eisenerz) ──────────────────────────────┐
  Bergwerk(Silikate) ─────────────────────┐        │
  Bergwerk(Titan) ─────────────────┐      │        │
  Bergwerk(Seltene Erden) ──┐      │      │        │
                            │      │      │        │
         ═══════════[Aufzug Lv1]═══════════════════╪════
                            │      │      │        │
[Orbit] ════════════════════╪══════╪══════╪════════╪════
                            │      │      │        │
                     Raffinerie   (↓)   Schmelze   │
                      Silikate+SE→Wafer  Eisenerz+Titan→Titanstahl
                            │              │        │
                    Basiskomponente     Stahl  ←──┐ │
                    ← Präzisionsfabrik:            │ │
                      Titanstahl×2 + Wafer×1 → Basis×3
                            │
                    Assembler:
                    ├─ Titanstahl×20 + Basis×5  → Schmelze Lv2   ◄─┐
                    ├─ Titanstahl×15 + Basis×8  → Raffinerie Lv2 ◄─┤ KREISLAUF
                    └─ Titanstahl×40 + Basis×8  → Werft Lv1       ◄─┘
                            │
                    Werft Lv1:
                    └─ Titanstahl×30 + Basis×10 → Systemfrachter
                            │
                    Systemfrachter → interstellarer Transport / Expansion
```

---

## 11. Pipeline-UI (Konzept)

> **Zweck:** Produktionskonfiguration grafisch als Datenflusspipeline darstellen,
> statt Tabellenformulare pro Anlage auszufüllen.

### Kernidee

Jede Anlage ist ein **Node** mit Eingangs- und Ausgangsports (Güter).
Spieler verbinden Ports per Drag & Drop → automatische Lager-Routing-Konfiguration.

```
[Bergwerk: Eisenerz]──────►[Schmelze]──────►[Lager Klasse I]──────►[Assembler]
[Bergwerk: Titan]─────────►[Schmelze]              ▲
                                                    │
[Bergwerk: Silikate]──────►[Raffinerie]─────►[Lager Klasse III]───►[Präzisionsfabrik]
[Bergwerk: Seltene Erden]─►[Raffinerie]
```

### Features

- **Engpass-Visualisierung:** Rot wenn Input-Fluss < Anlage-Bedarf
- **Lagerstand-Anzeige:** Live-Balken an jedem Lager-Node
- **Effizienz-Overlay:** η-Wert je Anlage, grün/gelb/rot
- **Klassen-Farbkodierung:** Kantenfarbe = Sensitivitätsklasse des Guts
- **Automatik-Modus:** System schlägt optimales Routing vor (2C-Hybrid)

### Umsetzungshinweis

- Library-Kandidat: React Flow (MIT-Lizenz ✅) als Basis für Node-Editor
- Speicherformat: Pipeline als JSON im PlayerProfile / StationConfig
- Umsetzung: AP6 (UI) oder eigenes AP nach AP4-Backend

---

## 12. Post-MVP: Pipeline-Graph-Modell

**Konzept (2026-03-22, noch nicht spezifiziert):**

Die Wirtschaft wird als **gerichteter Graph** modelliert:

- **Knoten** = Quellen (Minen, Harvester), Senken (Fabriken), Puffer (Lager), Produktionsanlagen
- **Kanten (Pipelines)** = gerichtete Verbindungen mit Kapazität und prozentualer Güteraufteilung
- **Physische Impl.** von Pipelines = Raumschiffe (interstellar/interplanetar) oder Aufzug (Planet ↔ Orbit)

**Design-Entscheidungen (vorläufig):**

| # | Aspekt | Entscheidung |
|---|---|---|
| P1 | Intra-/Inter | Alle Verbindungen sind Pipelines, auch innerplanetare (Mine → Lager → Schmelze) |
| P2 | Puffer | Lokale Knotenpuffer vorhanden, sehr klein (≈ 1 Runde). Ausbaubar durch Lagermodule. |
| P3 | Pipeline bauen | Spieler zieht Verbindung im GUI. Spiel berechnet benötigte Schiffe/Kapazität. Spieler bringt Schiffe zu Knoten → Route aktiv. |
| P4 | Unterbrechung | Nur inter-stellar und inter-planetar. Intraplanetare Verbindungen sind nicht angreifbar. |
| P5 | Aufzug | Ist ein spezieller Pipeline-Typ Planet ↔ Orbital-Knoten. Kapazität = Aufzug-Level. |
| P6 | GUI | React Flow Node-Graph. Kacheln für Anlagen, Kanten für Pipelines, Kapazitätsanzeige. |

---

## 13. Offene Punkte / Nächste Schritte

| Thema | Priorität | Nächster Schritt |
|---|---|---|
| Deposit-Umbau implementieren | Hoch | Migration 014, Generator, Economy2 (→ Umbauplan in production-mechanics_v1.1) |
| Kampfmechanik für Klasse-III/IV-Verluste | Hoch | In AP6 (Kampf) spezifizieren |
| Transportpenalty-Formel für Klasse II/III ohne Spezialfrachter | Mittel | In `game-params` kalibrieren |
| Pipeline-Graph-Datenmodell | Mittel | Post-MVP, vor AP5 (Flotten) spezifizieren |
| React Flow Pipeline-UI | Niedrig | Post-MVP, AP6 oder eigenes AP8 |
| Klasse-IV-Detonationschain-Mechanik | Niedrig | Nach Kampf-AP spezifizieren |

---

*Erstellt: 2026-03-22 · v1.1: 2026-03-29 — Deposit-Modell neu definiert, untouched-Flag, Förderformel, max_rate in Facility*
