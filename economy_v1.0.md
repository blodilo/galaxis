# Galaxis — Wirtschaftsmodell v1.0

**Status:** Design finalisiert · Alle Designentscheidungen getroffen · Implementation offen
**Abhängigkeiten:** Planetengenerierung, Fraktionssystem, Flottenmechanik, Tech-Tree

---

## Grundprinzipien

| Entscheidung | Wahl | Begründung |
|---|---|---|
| Marktmodell | **1A — Vollständig spielergetrieben** | Emergente Preise, kein NPC-Boden |
| Automatisierung | **2C — Hybrid** | Basis läuft auto, aktives Management bringt Vorteil |
| Zeitmodell | **3B — Tick-basiert** | Wie Flottenbefehle — Befehl → N Ticks Ausführung |
| Ressourcenverteilung | **4A — Regional** | Erzwingt interstellaren Handel |
| Forschung | **5 — Beides** | Effizienz UND neue Rezepte |

---

## Produktionskette (5 Stufen)

```
PLANET / ASTEROIDEN / GASRIESE
         │
         ▼  [Bergwerk / Bohrturm / Gas-Harvester]
   ═══ STUFE 1: ROHSTOFFE ══════════════════════════════
         │
         ▼  [Schmelze / Raffinerie / Bioreaktor]
   ═══ STUFE 2: HALBZEUG ═══════════════════════════════
         │
         ▼  [Präzisionsfabrik — ausschließlich orbital]
   ═══ STUFE 3: KOMPONENTEN ════════════════════════════
         │
         ▼  [Werft / Assembler]
   ═══ STUFE 4: ENDPRODUKTE ════════════════════════════
         │
         ▼  [Frachter · Handelsstation · Lager]
   ═══ STUFE 5: LOGISTIK / MARKT ══════════════════════
```

---

## Stufe 1 — Rohstoffe

### Ressourcenverteilung (4A: regional)

Welche Rohstoffe ein Planet liefert, ergibt sich aus **Planetentyp × Sternspektralklasse**.
Das schafft natürliche wirtschaftliche Geographie — keine Selbstversorgung möglich.

| Rohstoff | Primärquelle | Spektralklasse | Besonderheit |
|---|---|---|---|
| **Eisenerz** | Felsplanet | G, K, M | häufig, Basismetall |
| **Nickel** | Felsplanet, Asteroidengürtel | K, M | häufig |
| **Titan** | Heißer Felsplanet | F, A | mittelhäufig |
| **Chrom** | Heißer Felsplanet | F, A | mittelhäufig |
| **Seltene Erden** | Alter Felsplanet | K, M (alt) | selten, Technologieschlüssel |
| **Uran / Thorium** | Radioaktiver Felsplanet | — | selten, strategisch |
| **He-3** | Gasriese | beliebig | Fusionsschlüssel |
| **Wasserstoff** | Gasriese, Eismond | beliebig | Treibstoff-Vorläufer |
| **Wassereis** | Eismond, äußere Zone | — | Lebenserhaltung, Kühlmittel |
| **Organika** | Habitable Zone + Biosphäre | G, K | selten, Biotech |
| **Platin-Gruppe** | Asteroidengürtel | — | selten, Elektronik |
| **Exotische Materie** | Pulsar-Umgebung, NS | Pulsar | sehr selten, Hightech |

### Extraktionsgebäude

| Gebäude | Quelle | Output-Tick | Bemerkung |
|---|---|---|---|
| Bergwerk Lv1–5 | Felsplanet | 10–50 Einh./Tick | Oberfläche, Kapazität durch Aufzug begrenzt |
| Orbitalbohrer | Felsplanet (tief) | 5–20 Einh./Tick | orbital, seltenere Erze |
| Gas-Harvester | Gasriese | 20–80 Einh./Tick | orbital, nur Gas-Typen |
| Asteroidenschürfer | Asteroidengürtel | variabel | mobile Plattform |
| Bio-Extraktor | Habitable Planet | 5–15 Einh./Tick | benötigt Biosphäre Level ≥ 2 |

**Aufzug-Engpass:** Planetenoberfläche → Orbit ist der erste Engpass.
Kapazität = `Aufzug-Level × 100 Einh./Tick`. Ohne Aufzug: Shuttles (langsam, teuer).

---

## Stufe 2 — Halbzeug

Einfache Rezepte (1–2 Inputs). Verarbeitungsanlagen können orbital **oder** bodengebunden sein.

| Halbzeug | Input A | Input B | Fabrik | Ticks |
|---|---|---|---|---|
| **Titanstahl** | Eisenerz ×3 | Titan ×1 | Schmelze | 1 |
| **Chrom-Legierung** | Chrom ×2 | Eisenerz ×1 | Schmelze | 1 |
| **Keramik-Komposit** | Silikate ×2 | Chrom ×1 | Schmelze | 2 |
| **Halbleiter-Wafer** | Silikate ×1 | Seltene Erden ×1 | Raffinerie | 3 |
| **Fusionstreibstoff** | He-3 ×2 | Wasserstoff ×3 | Raffinerie | 1 |
| **Spaltmaterial** | Uran ×2 | — | Raffinerie (abgeschirmt) | 4 |
| **Biosynth-Grundstoff** | Organika ×2 | Wassereis ×1 | Bioreaktor | 2 |
| **Kühlmittel** | Wassereis ×3 | — | Raffinerie | 1 |

---

## Stufe 3 — Komponenten

Komplexe Rezepte (2–4 Inputs). Ausschließlich **orbital** — Mikrogravitation ist Voraussetzung
für Präzisionsfertigung dieser Klasse. Längere Produktionszeiten.

| Komponente | Inputs | Ticks | Verwendung |
|---|---|---|---|
| **Fusionsreaktor Lv1** | Fusionstreibstoff ×5, Titanstahl ×8, Halbleiter ×4 | 12 | Schiffe, Stationen |
| **Ionentriebwerk** | Fusionstreibstoff ×3, Titan ×4, Halbleiter ×3 | 8 | alle Schiffe |
| **Sprung-Antrieb** | Fusionsreaktor ×1, Spaltmaterial ×4, Seltene Erden ×6 | 24 | FTL-fähige Schiffe |
| **Phasen-Schild** | Halbleiter ×6, Seltene Erden ×4, Kühlmittel ×3 | 10 | Schiffe |
| **Waffensystem (kinetisch)** | Titanstahl ×6, Halbleiter ×2 | 6 | Schiffe |
| **Waffensystem (Energie)** | Spaltmaterial ×2, Halbleiter ×8, Seltene Erden ×2 | 14 | Schiffe |
| **Lebenserhaltung** | Biosynth ×4, Halbleiter ×3, Wassereis ×2 | 8 | bemannte Schiffe, Stationen |
| **Navigationscomputer** | Halbleiter-Wafer ×6, Platin-Gruppe ×2 | 10 | alle Schiffe |
| **Exotik-Prozessor** | Exotische Materie ×1, Halbleiter ×8, Platin-Gruppe ×4 | 40 | Hightech-Endprodukte |

---

## Stufe 4 — Endprodukte

### Schiffe (Werft-Produktion)

Werften haben **Slots** — jeder Slot kann einen Bauauftrag gleichzeitig abarbeiten.
Größere Schiffe blockieren mehr Slots und brauchen mehr Ticks.

| Schiff | Kernkomponenten | Slots | Ticks | Funktion |
|---|---|---|---|---|
| **Jäger** | Triebwerk ×1, Waffe kin. ×2, Lebenserhaltung (klein) ×1 | 1 | 20 | Kampf, Eskorte |
| **Bomber** | Triebwerk ×1, Waffe Energie ×1, Spaltmaterial ×2 | 1 | 28 | Schwere Angriffe |
| **Systemfrachter** | Triebwerk ×2, Lebenserhaltung ×1 | 1 | 30 | Intra-System-Logistik |
| **FTL-Frachter** | Sprung-Antrieb ×1, Triebwerk ×2, Lebenserhaltung ×1 | 2 | 60 | Inter-System-Logistik |
| **Kreuzer** | Fusionsreaktor ×2, alle Systeme ×2–4 | 4 | 200 | Kampfkapitalschiff |
| **Trägerschiff** | Fusionsreaktor ×3, alle ×4–8, Exotik-Prozessor ×1 | 8 | 500 | Flotten-Flaggschiff |
| **Kolonieschiff** | Lebenserhaltung (groß) ×4, Sprung-Antrieb ×1, alle ×2 | 4 | 300 | Expansion |

### Stationsmodule (Assembler-Produktion)

| Modul | Kernkomponenten | Ticks | Funktion |
|---|---|---|---|
| **Lagermodul** | Titanstahl ×20 | 15 | +500 Lagerkapazität |
| **Werft-Erweiterung** | Titanstahl ×30, Halbleiter ×10 | 40 | +2 Werft-Slots |
| **Marktmodul** | Navigationscomputer ×2, Halbleiter ×15 | 30 | Handelszugang |
| **Forschungsmodul** | Exotik-Prozessor ×1, Halbleiter ×20 | 60 | Forschungs-Ticks |
| **Verteidigungsplattform** | Waffensystem ×4, Schild ×2, Fusionsreaktor ×1 | 80 | Stationsschutz |

---

## Stufe 5 — Logistik

### Drei Transportschichten

```
[Planetenoberfläche]
    ↕  Aufzug / Shuttle  (Flaschenhals, kapazitätsbegrenzt)
[Orbitale Station / Lager]
    ↕  Systemfrachter  (kein FTL, schnell, N Ticks pro Route)
[Andere Station im selben System]
    ↕  FTL-Frachter  (Sprungantrieb, M Ticks, Treibstoffkosten)
[Station in anderem Sternsystem]
```

### Handelsrouten (Tick-basiert wie Flottenbefehle)

Ein Spieler definiert eine Route als Befehlssequenz:

```
ROUTE: "Titan-Export Alpha"
  1. Lade: Station Kepler-7b-Orbit → 200× Titan
  2. Reise: → Station Sol-Handelsstation  (14 Ticks)
  3. Entlade: → Lager Sol-HS
  4. Lade: → 150× Fusionstreibstoff
  5. Reise: → Station Kepler-7b-Orbit  (14 Ticks)
  6. Entlade: → Lager Kepler-7b
  [Wiederhole]
```

- **Befehl erteilen → Ticks laufen → Befehl abgeschlossen** — kein Echtzeit-Monitoring nötig
- Routen können **abgebrochen** werden (Frachter kehrt ins Heimatsystem zurück, N/2 Ticks)
- **Warteschlange:** mehrere Routen pro Frachter sequenziell oder mehrere Frachter parallel
- **Offline-Sicherheit:** Frachter führen laufende Befehle zu Ende, neue Befehle warten auf Login

### Lager & Puffer

- Jede Station hat **Lager** (ausbaubar via Modul)
- Überlaufschutz: Produktionsanlage pausiert automatisch wenn Lager voll (2C-Hybrid)
- **Prioritätswarteschlange:** Spieler definiert welche Rezepte bevorzugt laufen wenn Inputs knapp

### Markt (1A — vollständig spielergetrieben)

- Nur an Stationen mit **Marktmodul** handelbar
- **Order Book:** Kauf- und Verkaufsaufträge, Matching bei Preisüberschneidung
- **Keine NPC-Preise** — Preise entstehen rein durch Angebot/Nachfrage
- **Kommunikationsverzögerung (optional):** Preisdaten aus fernen Systemen sind N Ticks alt
  → Hard-Sci-Fi-Authentizität, verhindert perfekte Arbitrage-Bots
- **Marktgebühr:** kleiner Prozentsatz pro Trade → Geldbecken, verhindert Spam-Orders
- **Sichtbarkeit:** nur Systeme mit eigener Infrastruktur oder Scout-Präsenz zeigen Live-Preise

---

## Forschung (Entscheidung 5: Effizienz + neue Rezepte)

### Zwei parallele Forschungsstränge

```
EFFIZIENZ-BAUM                    ENTDECKUNGS-BAUM
────────────────                   ─────────────────
Schmelze Lv2                       Halbleiter Lv2
 → -15% Rohstoffverbrauch           → schaltet Exotik-Wafer frei
Raffinerie Lv3
 → +20% Output/Tick                Sprungantrieb Lv2
Werft-Optimierung                   → FTL-Effizienz +30%
 → -10% Bauzeit alle Schiffe
                                   Exotik-Prozessor Lv2
                                    → schaltet Psi-Komponenten frei (?)
```

### Forschungs-Mechanik

- Forschungsmodule auf Stationen produzieren **Forschungs-Ticks**
- Jede Forschung kostet X Ticks + spezifische Komponenten als Materialkosten
- Forschungen sind **spielergebunden** (kein automatischer Technologietransfer)
- **Wissenshandel:** Baupläne können als Item gehandelt werden (lizensiert oder raubkopiert)

---

## Finalisierte Designentscheidungen

| # | Frage | Entscheidung |
|---|---|---|
| 1 | **Ressourcenmengen** | **Erschöpfbar** — Vorkommen sind endlich; mehr Förderung erfordert logistischen Mehraufwand (tiefere Minen, Aufzug-Upgrades). Erzwingt Expansion und interstellaren Handel. |
| 2 | **Tick-Länge** | **Partieparameter** — Basis: 360 Minuten Echtzeit (6 h) pro Strategie-Tick. Konfigurierbar in `game-params.yaml`. |
| 3 | **Frachter-Risiko** | **Ja** — Frachter auf aktiven Routen können angegriffen und abgefangen werden. Erzwingt Eskortenlogistik und strategische Routenwahl. |
| 4 | **Steuern/Pacht** | **Nein** — Keine Planetenpacht an Fraktion/Imperium. Planeten gehören dem Spieler, der sie kolonisiert. |
| 5 | **Sabotage** | **Ja** — Feindliche Spieler können Produktionsanlagen sabotieren. Details: Mechanik in AP6 spezifizieren (Agenten/Spezialoperationen). |
| 6 | **NPC-Nachfrage** | **Nein** — Kein NPC-Markt als Absatzgarantie. Preise entstehen rein durch Spieler-Angebot/-Nachfrage (konsequent 1A). |
| 7 | **Blaupausen** | **Forschung** — Keine Rezepte sind zu Spielbeginn bekannt. Alle Produktionsstufen müssen über den Forschungsbaum freigeschaltet werden. |
| 8 | **Produktionsverlust** | **Nein explizit** — Keine zufälligen Produktionsausfälle/Unfälle. Verlust nur durch Sabotage (→ Entscheidung 5) oder Ressourcenmangel. |

---

## Datenbankschema-Skizze (konzeptionell)

```
Planet
  ├── resource_deposits[]: { type, quantity, extraction_rate }
  └── infrastructure[]: { building_type, level, slots_used }

ProductionOrder
  ├── facility_id
  ├── recipe_id
  ├── quantity_ordered
  ├── ticks_remaining
  └── status: queued | running | completed | cancelled

TradeRoute
  ├── ship_id
  ├── steps[]: { action: load|travel|unload, target, commodity, quantity }
  ├── current_step
  ├── ticks_remaining_step
  └── repeat: boolean

MarketOrder
  ├── station_id
  ├── player_id
  ├── commodity_type
  ├── price_per_unit
  ├── quantity
  └── type: buy | sell

ResearchProject
  ├── player_id
  ├── tech_id
  ├── ticks_remaining
  └── material_cost_paid: boolean
```

---

*Erstellt: 2026-03-21 · Alle Designentscheidungen fixiert 2026-03-21 · Nächster Schritt: DB-Schema finalisieren → BL-Tickets erstellen (AP4)*
