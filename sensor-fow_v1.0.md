# Sensor-Mechanik & Fog of War – Galaxis v1.0

**Datum:** 2026-03-13
**Referenz:** GDD v1.24 · server-core-map_v1.0.md

---

## 1. Wissenschaftliche Grundlage

### 1.1 Der einzelne Parameter: Sensor Rating (SR)

Der physikalisch korrekte Einzel-Parameter für die Empfindlichkeit eines Teleskops ist
die **effektive Sammelflächе** (Collecting Area):

```
SR = A = π × (D/2)²     [Einheit: m²]

D  = Apertur-Durchmesser des Spiegels/Objektivs
SR = Sensor Rating (direkt dem Spieler angezeigt)
```

Warum SR und nicht Durchmesser?
- SR ist additiv: Zwei 5-m²-Sensoren ergeben SR = 10 m² (Interferometer/Array)
- SR ist linear zur Nachweisempfindlichkeit (doppeltes SR → doppelte Photonen)
- Spieler versteht „30 m²" intuitiv als „größer als 0,5 m²"

### 1.2 Detektionsreichweite

Das Licht einer Quelle mit Leuchtkraft L nimmt mit dem Abstand d nach dem
**Inversen Quadratgesetz** ab:

```
Empfangene Flussdichte:  F = L / (4π d²)

Ein Sensor der Fläche SR kann nachweisen bis zu F_min ∝ 1/SR.
Einsetzen und nach d auflösen:

d_detect = k × √(SR × L)

k = Kalibrierungskonstante (enthält Nachweisschwelle, Wellenlängenbereich,
    Rauschen, Integrationszeit – als Gameplay-Parameter gesetzt)
```

### 1.3 Kalibrierung k für das Spiel

**Ziel:** Ein ausgebautes planetares Observatorium soll M-Zwerge bis ~2.000 ly
detektieren (strategisch sinnvoll, nicht die gesamte Galaxie).

```
Planetares Observatorium: SR = 177 m² (Spiegel D = 15 m)
M-Zwerg:                  L  = 0,01 L☉

d = k × √(177 × 0,01) = k × 1,33

d = 2.000 ly  →  k ≈ 1.500 ly / √(m² · L☉)
```

---

## 2. Sensor-Typen & Referenzwerte

| Sensor-Typ | Apertur D | SR (m²) | Plattform |
|---|---|---|---|
| Scout-Array | 0,30 m | 0,07 | Kleines Schiff, Sonde |
| Taktisches Array | 1,00 m | 0,79 | Fregatte / Zerstörer |
| Survey-Array | 1,60 m | 2,00 | Aufklärungsschiff |
| Planetarer Außenposten | 5,00 m | 19,6 | Kolonie (früh) |
| Planetares Observatorium | 15,00 m | 177 | Entwickelter Planet |
| Deep-Space-Array | 30,00 m | 707 | Megastruktur / Orbitalarr. |
| Sensor-Netzwerk (Array) | Mehrere | bis ~5.000+ | Verbund mehrerer Anlagen |

**Array-Regel:** Mehrere Sensoren desselben Spielers im selben System addieren
ihre SR: `SR_gesamt = Σ SR_i`

---

## 3. Detektionsreichweiten im Spiel

### 3.1 Stellare Objekte (Sternkarte – Galaxie-Ebene)

```
d_detect [ly] = 1.500 × √(SR [m²] × L [L☉])
```

| Objekt | L [L☉] | Scout (SR=0,07) | Observ. (SR=177) | Deep Space (SR=707) |
|---|---|---|---|---|
| O-Stern | 300.000 | 218.000 ly ✓ | 1.098.000 ly ✓ | 2.197.000 ly ✓ |
| B-Stern | 1.000 | 12.600 ly ✓ | 63.200 ly ✓ | 126.500 ly ✓ |
| G-Stern (Sonne) | 1,0 | 397 ly | 20.000 ly ✓ | 39.900 ly ✓ |
| K-Stern | 0,1 | 126 ly | 6.300 ly ✓ | 12.600 ly ✓ |
| M-Zwerg | 0,01 | 40 ly | 2.000 ly ✓ | 3.990 ly ✓ |
| Wolf-Rayet | 500.000 | 281.000 ly ✓ | – | – |
| Pulsar (visuell) | ~0,001 | 13 ly | 632 ly | 1.260 ly |
| Schwarzes Loch | 0 (kein Licht) | – | – | – |

**Schwarze Löcher** sind durch Leuchtkraft allein nicht detektierbar.
Sie hinterlassen jedoch eine **FTLW-Anomalie** (Gravitationslinse, Skalarfeld-Verzerrung)
→ separater Detektionsmechanismus (FTL-Sensor, s. Abschnitt 5).

### 3.2 Planeten & Monde (System-Ebene)

Planeten leuchten nur durch reflektiertes Sternlicht:

```
L_planet ≈ Albedo × (R_planet / 2d_orbit)² × L_Stern

Erdähnlicher Planet (Albedo 0,3, 1 AU, G-Stern):
  L_planet ≈ 5 × 10⁻¹⁰ L☉

Gasriese (Albedo 0,5, 5 AU, G-Stern):
  L_planet ≈ 5 × 10⁻¹⁰ L☉  (ähnliche Größenordnung)
```

Detektionsreichweite für Planeten:

```
Observatorium (SR=177):
  d = 1.500 × √(177 × 5×10⁻¹⁰) = 1.500 × 2,98×10⁻⁴ ≈ 0,45 ly

Deep-Space-Array (SR=707):
  d = 1.500 × √(707 × 5×10⁻¹⁰) ≈ 0,89 ly
```

**Fazit:** Planeten und Monde sind mit keinem realistisch baubaren Sensor
aus interstellarer Distanz detektierbar. Die physikalische Grenze liegt bei
< 1 Lichtjahr – also praktisch nur **innerhalb desselben Sternensystems**.

### 3.3 Schiffe (Thermische Signatur)

Schiffe strahlen Abwärme ab. Die thermische Leuchtkraft folgt dem
**Stefan-Boltzmann-Gesetz:**

```
P_thermal = ε × σ × A_radiator × T⁴

σ = 5,67 × 10⁻⁸ W/(m²K⁴)  (Stefan-Boltzmann-Konstante)
ε = Emissionsgrad (~0,9 für Metall-Radiatoren)
T = Radiator-Temperatur
```

Typische Schiff-Temperaturen und Signaturen:

| Betriebszustand | T_radiator | Rel. Signatur | L_equiv [L☉] |
|---|---|---|---|
| Abgeschaltet / Drift | 50 K | minimal | ~10⁻¹⁷ |
| Standby | 300 K | niedrig | ~10⁻¹⁵ |
| Normalbetrieb | 500 K | mittel | ~10⁻¹³ |
| Vollschub / Kampf | 900 K | hoch | ~10⁻¹¹ |
| FTL-Warp-Initiierung | >1500 K (kurz) | sehr hoch | ~10⁻⁹ |

Detektionsreichweite für Schiffe (Normalbetrieb, L=10⁻¹³ L☉):

```
Scout (SR=0,07):    d = 1.500 × √(0,07 × 10⁻¹³) = 0,013 AU ≈ 2 Mio km
Taktisches Array (SR=0,79):  d = 0,042 AU ≈ 6,3 Mio km
Observatorium (SR=177):      d = 0,63 AU
Deep Space Array (SR=707):   d = 1,26 AU
```

**Fazit:** Schiffe sind selbst mit den stärksten planetaren Sensoren nur
innerhalb weniger AU detektierbar. Im tiefen interstellaren Raum sind sie
unsichtbar – außer beim FTL-Warp-Start.

---

## 4. Fog of War – Spielmechanik

### 4.1 Drei Sichtbarkeits-Ebenen

```
Ebene 1 – STERNKARTE (Galaxie-Zoom)
  Was:   Sterne, Nebel (grob)
  Wie:   Leuchtkraft-basiert, Formel d_detect
  Wann:  Immer (passiv, kein aktiver Scan nötig)

Ebene 2 – SYSTEMKARTE (System-Zoom)
  Was:   Planeten, Monde, Asteroiden, fremde Flotten im System
  Wie:   Erfordert physische Präsenz (Nav-Punkt) oder In-System-Sensor
  Wann:  Nur wenn Schiff/Sonde im System ist

Ebene 3 – OBERFLÄCHE / DETAIL
  Was:   Ressourcen, Terrain, Infrastruktur
  Wie:   Orbital-Survey oder Landung
  Wann:  Nach explizitem Survey-Befehl
```

### 4.2 Ebene 1 – Sternkarte im Detail

**Sichtbarkeitsberechnung (Server, einmal pro Strategietick):**

```
Für jeden Spieler P:
  SR_best = Σ SR aller Sensoren von P im besten System
            (oder max, falls keine Additionalität gewünscht)

  Für jeden Stern S in der Galaxie:
    d = Distanz(P.bester_Sensorstandort, S.position)
    if d ≤ 1.500 × √(SR_best × S.luminosity):
      Stern ist sichtbar für P
```

**Informationsqualität nach Distanz:**

| Distanz / d_detect | Sichtbare Information |
|---|---|
| < 10% | Vollständige Daten: Typ, Masse, Leuchtkraft, Nebeltyp, grobe Planetenanzahl (als Schätzung) |
| 10 – 50% | Typ, Spektralklasse, Leuchtkraft |
| 50 – 90% | Nur: Lichtsignal vorhanden, Spektralklasse unsicher |
| 90 – 100% | Knappes Limit: Lichtsignal, Typ unbekannt ("Unbekanntes Objekt") |
| > 100% | Nicht sichtbar (Fog) |

**Nebel:**
- H-II-Regionen sind durch ihre eigene Emission (ionisiertes Gas) direkt sichtbar
  (hohe effektive Leuchtkraft des Nebels selbst) → sichtbar aus großer Distanz
- SNR: Schwächer, aber noch aus ~10.000 ly sichtbar
- Kugelsternhaufen: Summierte Leuchtkraft → gut sichtbar aus großer Distanz

**Schwarze Löcher auf der Sternkarte:**
- Nicht durch Leuchtkraft sichtbar
- Sichtbar durch **FTLW-Anomalie**: wenn SR des Spielers ausreicht, eine
  FTLW-Verzerrung im Voxelgrid zu detektieren → separater Mechanismus
  (erfordert FTL-Sensor-Upgrade, s. Abschnitt 5)

### 4.3 Ebene 2 – Systemsichtbarkeit

**Voraussetzung:** Mindestens ein Schiff oder eine Sonde hat den Nav-Punkt
des Systems erreicht (Systemeintritt).

**Nach Systemeintritt:**
- Alle Planeten und größeren Monde: automatisch sichtbar (sie sind hell genug
  bei kurzer Distanz)
- Asteroiden und Kleinkörper: erfordern In-System-Scan (Survey-Befehl,
  belegt das Schiff für N Ticks)
- Fremde Infrastruktur auf Planeten: sichtbar bei Orbitalflug
- Fremde Flotten: IR-Sensor und Radar-Mechanik (s. GDD Kap. 9)

**Systemgedächtnis:**
- Einmal besuchte Systeme bleiben in der Karte des Spielers eingetragen
- Planetardaten veralten nicht (Planeten ändern sich nicht)
- Fremde Flotten: veralten nach N Ticks ohne Bestätigung
  ("Last Known Position" – als gestricheltes Symbol)

### 4.4 Ebene 3 – Oberflächendetail

Erst nach einem **Orbital-Survey**:
- Ressourcen-Vorkommen auf dem Planeten sichtbar
- Nutzflächen-Karte
- Vorhandene Gebäude (fremde Infrastruktur)

Nach einer **Landung / Sondenmission:**
- Feinere Ressourcenverteilung
- Atmosphären-Analyse-Bestätigung (war vorher nur Schätzung)

---

## 5. FTL-Sensor (Gravitationslinsen-Detektion)

Schwarze Löcher, Pulsare und das SMBH sind durch ihre **FTLW-Feldverzerrung**
auch ohne Lichtemission detektierbar.

**Mechanik:**
- Der Spieler baut einen **FTL-Sensor** (spezielle Anlage, teurer als optisches Observatorium)
- FTL-Sensoren messen lokale FTLW-Gradienten (Schwankungen im Skalarfeld)
- Detektionsreichweite: `d_FTL = k_FTL × √(SR_FTL × M_objekt)`
  (analog zur optischen Formel, aber mit Objektmasse statt Leuchtkraft)
- Schwarze Löcher mit hoher Masse → gut aus großer Distanz detektierbar
- FTL-Sensor detektiert auch **aktive FTL-Warp-Signaturen** von Schiffen
  (das bereits im GDD spezifizierte Skalar-Interferometer)

---

## 6. Spielmechanische Konsequenzen (Zusammenfassung)

| Situation | Konsequenz |
|---|---|
| Spieler hat nur Schiffs-Sensoren | Sternkarte zeigt nur helle Sterne in Reichweite; M-Zwerge nur in der Nähe |
| Spieler baut planetares Observatorium | Massiv größere Reichweite; kann M-Zwerge bis 2.000 ly kartieren |
| Spieler baut Sensor-Netzwerk (mehrere Anlagen) | SR addiert sich; noch größere Reichweite |
| Spieler will Planeten in fernem System entdecken | Muss physisch einreisen – keine Alternative |
| Schwarzes Loch auf der Karte sehen | Benötigt FTL-Sensor (Gravitationslinsen-Detektion) |
| Feindliche Flotte tief im eigenen System orten | In-System-Sensoren nötig; ohne diese: Überraschungsangriff möglich |
| Feindliche Flotte im interstellaren Raum sehen | Praktisch unmöglich ohne FTL-Sensor (Warp-Signatur bei Sprungstart) |

---

## 7. Offene Punkte (für nächste Iteration)

| Thema | Priorität |
|---|---|
| Sensorupgrade-Pfade (Technologiebaum-Anbindung) | Hoch (bei AP7) |
| Sensor-Netzwerk: additiv oder Maximum? | Mittel (Design-Entscheidung) |
| Veraltungsrate von Last-Known-Position-Daten | Mittel |
| FTL-Sensor SR-Werte und k_FTL Kalibrierung | Mittel |
| Survey-Dauer in Ticks | Niedrig (Balancing) |
| Oberflächen-Survey: Sonde vs. Landungsschiff | Niedrig |
