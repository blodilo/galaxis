# Spielmechanik: Forschung – Galaxis v1.0

**Datum:** 2026-03-13
**Referenz:** GDD v1.24 · tech-tree_v1.0.jsonld · game-params_v1.0.yaml

---

## 1. Konzept

Forschung ist ein stochastischer Prozess: Jede Runde (Tick) kann scheitern.
Das erzeugt realistische Varianz – kein Spieler kann den genauen Fertigstellungstermin
exakt vorhersagen, aber mehr Investition verkürzt die Erwartungszeit.

**Ergebnis einer abgeschlossenen Forschung ist immer ein Bauplan
(Konstruktionsunterlagen)** – kein direktes Objekt. Der Bauplan enthält alle
notwendigen Produktionsinputs und Integrationsdaten.

---

## 2. Inputs pro Tick

Jeder Forschungsauftrag verbraucht pro Tick die folgenden Ressourcen,
solange er aktiv ist:

| Input | Beschreibung |
|---|---|
| **Labore** | Anzahl Laborgebäude, die dem Auftrag zugewiesen sind (Infrastruktur, planetar) |
| **Wissenschaftler** | Anzahl ausgebildeter Wissenschaftler (aus Akademie, Pyramide C) |
| **Credits** | Laufende Betriebskosten pro Tick |
| **Materialien** | Kleine Mengen spezifischer Elemente (Experimentierproben, werden verbraucht) |

Sind nicht alle Inputs erfüllt (z. B. Materialien fehlen), ruht die Forschung –
kein Fortschritt, keine Materialkosten, Credits laufen weiter (Personalkosten).

---

## 3. Fortschrittsmodell

### 3.1 Tick-Ablauf

```
Jeden Strategie-Tick, wenn alle Inputs vorhanden:

1. Materialien abbuchen (werden konsumiert, egal ob Erfolg oder Misserfolg)
2. Credits abbuchen
3. Würfeln: random(0,1) < risk_per_tick?
   → JA  (Fehlschlag): kein Fortschritt diesen Tick
   → NEIN (Erfolg):    Fortschritt += 1 Tick
4. Wenn Fortschritt >= required_progress_ticks → Forschung abgeschlossen
```

### 3.2 Parameter

| Parameter | Beschreibung | Ort |
|---|---|---|
| `required_progress_ticks` | Wie viele Erfolgs-Ticks benötigt werden | `tech-tree.jsonld` → `research_duration_ticks` |
| `risk_per_tick` | Wahrscheinlichkeit eines Fehlschlags pro Tick | `tech-tree.jsonld` → `risk_per_tick` |
| `base_research_speed` | Globaler Multiplikator (Servereinstellung) | `game-params.yaml` |

### 3.3 Erwartete Abschlusszeit

```
E[Ticks bis Abschluss] = required_progress_ticks / (1 - risk_per_tick)

Beispiele:
  risk = 0,05 (sicher):  E = base × 1,05  →  5% länger
  risk = 0,20 (normal):  E = base × 1,25  → 25% länger
  risk = 0,40 (riskant): E = base × 1,67  → 67% länger
  risk = 0,60 (gefährlich): E = base × 2,50 → 150% länger
```

### 3.4 Wissenschaftler-Bonus

Zusätzliche Wissenschaftler senken das Risiko:

```
risk_effective = risk_base × max(0, 1 - scientists_assigned × scientist_risk_reduction)

scientist_risk_reduction = game-params.yaml → research.scientist_risk_reduction
(Standardwert: 0,05 pro zusätzlichem Wissenschaftler über Minimum)
```

Mehr Wissenschaftler → zuverlässigere Forschung, aber höhere Personalkosten.
Diminishing returns: Das Risiko kann nicht unter 0 sinken.

### 3.5 Labor-Bonus

Mehr zugewiesene Labore als das Minimum können den Fortschritt pro
Erfolgs-Tick erhöhen (parallele Versuchsreihen):

```
progress_per_success = 1 + max(0, labs_assigned - labs_required) × lab_bonus_factor

lab_bonus_factor = game-params.yaml → research.lab_bonus_factor
(Standardwert: 0,25 – jedes zusätzliche Labor erhöht Fortschritt um 25%)
```

---

## 4. Risikokategorien

Der Risikofaktor gibt zugleich eine narrative Einschätzung der Forschungsschwierigkeit:

| risk_per_tick | Kategorie | Farbe (UI) | Typische Technologien |
|---|---|---|---|
| 0,00–0,10 | Gesichert | Grün | Basisoptik, einfacher Bergbau |
| 0,11–0,25 | Normal | Gelb | Hochleistungsoptik, Fusionsantrieb |
| 0,26–0,45 | Riskant | Orange | FTL-Sensorik, Verbundpanzerung |
| 0,46–0,65 | Gefährlich | Rot | Skalarfeld-Manipulation, Graser |
| 0,66–0,85 | Experimentell | Dunkelrot | Antimaterie, Sprungtor-Theorie |

---

## 5. Output: Bauplan (Konstruktionsunterlagen)

Nach Abschluss der Forschung erhält der Spieler einen **Bauplan**.
Dieser enthält alle Daten, die für Produktion und Integration notwendig sind.

### 5.1 Produktionsrezept

Ressourcen und Zeit, die benötigt werden, um **eine Einheit** herzustellen:

```yaml
blueprint:
  production_inputs:
    credits: 1000
    iron: 500
    silicon: 200
    neodymium: 10
  production_time_ticks: 10
  required_facility: "basic-shipyard"   # Oder "advanced-shipyard", "planetary-factory"
```

### 5.2 Schiffsintegration (optional)

Falls die Technologie ein Schiffsmodul ergibt:

```yaml
ship_integration:
  mass_tons: 50
  energy_consumption_mw: 5.0
  heat_output_kw: 20.0
  voxel_count: 4
  mount_type: "hull"         # hull | weapon | sensor | engine | utility
  size_class: "frigate+"     # Mindest-Schiffsklasse für dieses Modul
```

### 5.3 Planetare Integration (optional)

Falls die Technologie ein Gebäude oder eine Anlage ergibt:

```yaml
planetary_integration:
  footprint_sectors: 2
  power_draw_mw: 15.0
  workforce_required: 50
  build_time_ticks: 20
  terrain_requirements: ["flat", "no-ocean"]
```

### 5.4 Spezialeffekte

Freie Effekteliste für Boni/Mali die nicht in die obigen Kategorien passen:

```yaml
effects:
  - type: "sensor_sr_bonus"
    value: 177
    description: "Fügt SR 177 m² zum planetaren Sensornetzwerk hinzu"
  - type: "ftlw_multiplier"
    value: 0.7
    target: "voxel_range_100ly"
    description: "Reduziert FTLW in 100 ly Umkreis auf 70%"
```

---

## 6. Spielerinteraktion

### Forschung starten

1. Spieler wählt Technologie aus dem Baum (Voraussetzungen müssen erfüllt sein)
2. Spieler weist zu: Labore (wählt Planet + Anzahl), Wissenschaftler (Anzahl)
3. System berechnet und zeigt: Erwartete Dauer, Risikokategorie, Kosten/Tick
4. Spieler bestätigt → Forschungsauftrag in Action Queue

### Laufende Forschung

- Jederzeit pausierbar (Ressourcenverbrauch stoppt, kein Fortschrittsverlust)
- Wissenschaftler können umgeschichtet werden (Risiko ändert sich sofort)
- Materialien müssen in lokalem Lager verfügbar sein (kein automatischer Import)

### Abschluss

- Notification: „Forschung abgeschlossen: [Technologiename]"
- Bauplan erscheint in der Bauplandatenbank des Spielers
- Technologie im Baum als abgeschlossen markiert (Folgetechnologien freigeschaltet)

---

## 7. Verknüpfung mit game-params.yaml

Folgende Parameter aus dieser Mechanik sind in game-params einzutragen:

```yaml
research:
  scientist_risk_reduction: 0.05   # [BALANCING] Risikoredukion pro zusätzlichem Wissenschaftler
  lab_bonus_factor: 0.25           # [BALANCING] Zusätzlicher Fortschritt pro Extra-Labor
  base_research_speed: 1.0         # [BALANCING] Globaler Zeitstrecker/-staffer
  parallel_research_slots: 1       # [BALANCING] Gleichzeitige Forschungsaufträge
  scientist_research_bonus: 0.10   # [BALANCING] Bonus auf research_speed pro Wissenschaftler
```
