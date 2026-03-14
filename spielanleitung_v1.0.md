# Galaxis – Spielanleitung v1.0

**Version:** 1.0 · **Datum:** 2026-03-13
*Dieses Dokument erklärt die Spielmechaniken aus Spielerperspektive.*

---

## 1. Die Galaxie

Du spielst eine Spezies in einer prozedural generierten Balkenspiralgalaxie.
Die Karte zeigt Sternensysteme, Nebel und die Routen deiner Flotten.

Die Galaxie besteht aus drei Zonen:

- **Kern (Bulge):** Sterne liegen sehr dicht beieinander. Ressourcenreich,
  aber Reisen sind langsam – der FTL-Widerstand ist hier am stärksten.
- **Spiralarme:** Die bevölkerten Gebiete. Sterne liegen in vertretbarem
  Abstand, Reisen sind möglich.
- **Galaktischer Rand:** Sterne liegen weit auseinander. Reisen sind schnell,
  aber Ressourcen sind seltener.

---

## 2. Sichtbarkeit & Fog of War

Du siehst nicht automatisch die gesamte Galaxie. Was du siehst, hängt von
deinen **Sensoren** ab.

### 2.1 Sensorreichweite

Jeder Sensor hat einen einzigen Kennwert: den **Sensor Rating (SR)** in m².
Er beschreibt die effektive Sammelfläche des Sensors – je größer, desto
empfindlicher.

Die Detektionsreichweite für einen Stern hängt von SR und der Leuchtkraft
des Sterns ab:

> **Je heller der Stern und je größer dein Sensor, desto weiter kannst du sehen.**

Konkrete Beispiele (Richtwerte):

| Sensor | SR | G-Stern sichtbar bis | M-Zwerg sichtbar bis |
|---|---|---|---|
| Schiffs-Scout-Array | 0,07 m² | ~400 ly | ~40 ly |
| Taktisches Schiffsarray | 0,79 m² | ~1.300 ly | ~130 ly |
| Planetares Observatorium | 177 m² | ~20.000 ly | ~2.000 ly |
| Deep-Space-Array | 707 m² | ~40.000 ly | ~4.000 ly |

**Wichtig:** Planeten, Monde und feindliche Schiffe sind von weitem
grundsätzlich nicht detektierbar – dafür sind sie zu lichtschwach.
Sie erfordern einen Systembesuch (→ Abschnitt 3).

### 2.2 Was du siehst (und was nicht)

| Objekt | Sichtbar aus der Ferne? | Voraussetzung |
|---|---|---|
| Helle Sterne (O, B, A) | Ja – aus sehr großer Distanz | Beliebiger Sensor |
| Sonnenähnliche Sterne (G, K) | Ja – mittlere Distanz | Planetarer Sensor empfohlen |
| Rote Zwerge (M) | Ja – nur in der Nähe | Starker Sensor nötig |
| Schwarze Löcher | Nein (kein Licht) | FTL-Sensor erforderlich |
| Pulsare | Kaum – sehr schwach | Starker Sensor + Nähe |
| Nebel (H-II, Supernova-Überreste) | Ja – leuchten selbst | Beliebiger Sensor |
| Planeten & Monde | Nein | Systembesuch erforderlich |
| Feindliche Schiffe | Nein (Weltraum) | Im System: IR/Radar-Sensor |

### 2.3 Informationsqualität

Sterne am Rand deiner Sichtweite sind unsicher identifiziert. Je näher
ein Stern an deinen Sensoren liegt, desto mehr Details siehst du:

- **Nahe (< 10% der Maximalreichweite):** Vollständige Daten – Typ, Masse,
  Leuchtkraft, Nebeltyp, grobe Planetenanzahl (als Schätzung).
- **Mittel (10–50%):** Spektralklasse und Leuchtkraft sichtbar.
- **Weit (50–90%):** Nur: „Ein Lichtsignal unbekannten Typs."
- **Grenze (90–100%):** Das Objekt ist gerade eben sichtbar – Typ unbekannt.

---

## 3. Exploration – Systeme entdecken

### 3.1 Systembesuch

Um ein Sternensystem vollständig zu erkunden, muss eines deiner Schiffe
den **Nav-Punkt** des Systems erreichen – den Eintrittspunkt am Rand des
Gravitationsfelds des Sterns.

Nach dem Eintritt werden automatisch sichtbar:
- Alle Planeten und größeren Monde
- Asteroiden (nach kurzer Scan-Zeit)
- Sichtbare feindliche Infrastruktur und Flotten (mit Sensoren)

### 3.2 Detailscan (Orbital-Survey)

Ein **Orbital-Survey** liefert:
- Genaue Ressourcen-Vorkommen auf Planeten
- Detaillierte Nutzflächenverteilung
- Bestätigung der Atmosphärenzusammensetzung

Der Survey belegt dein Schiff für mehrere Ticks. Spezielle Survey-Schiffe
(mit großem Survey-Array) erledigen das deutlich schneller.

### 3.3 Systemgedächtnis

Einmal besuchte Systeme bleiben dauerhaft in deiner Karte eingetragen.
Feindliche Flottenpositionen veralten jedoch nach einigen Ticks:
sie werden als **„Letzte bekannte Position"** angezeigt (gestricheltes Symbol),
bis du sie erneut bestätigst.

---

## 4. Sensor-Netzwerke

### 4.1 Additive Sensoren (Stationäre Anlagen)

Mehrere stationäre Sensoranlagen auf verschiedenen Planeten und Stationen
können zu einem **Sensor-Netzwerk** kombiniert werden.

> Befinden sich zwei Stationen im gegenseitigen Sichtbereich, addieren sich
> ihre Sensor Ratings: `SR_netzwerk = SR₁ + SR₂ + ... + SR_n`

Ein Netzwerk aus fünf planetaren Observatorien (je 177 m²) erreicht
SR = 885 m² – deutlich mehr als ein einzelnes Deep-Space-Array.

**Spielstrategie:** Baue Sensornetzwerke entlang deiner Grenzen, um früh
zu erkennen, wenn feindliche Flotten auf dich zureisen.

### 4.2 Netzwerk-Aktualisierung

Wenn du eine **neue Sensoranlage** errichtest, wird die effektive Reichweite
des gesamten Netzwerks einmalig neu berechnet – für alle Stationen, deren
Sichtbereich sich mit der neuen Anlage überschneidet.

Du siehst die aktualisierte Karte unmittelbar nach Fertigstellung der neuen Station.

### 4.3 Schiffssensoren

Schiffssensoren sind **nicht additiv** und werden nicht ins Netzwerk eingebunden.
Sie gelten nur für das jeweilige Schiff und seinen aktuellen Standort.
Ein gut ausgerüstetes Aufklärungsschiff kann dennoch tief in feindliches
oder unerforschtes Gebiet vordringen und Daten nach Hause funken.

---

## 5. FTL-Sensoren (Gravitationslinsen-Detektion)

Schwarze Löcher und Pulsare emittieren kein Licht und sind mit optischen
Sensoren nicht direkt sichtbar. Sie verzerren jedoch das FTL-Feld der
Umgebung messbar.

Mit einem **FTL-Sensor** (eine spezialisierte, teure Anlage) kannst du:
- Schwarze Löcher und Pulsare auf der Karte lokalisieren
- Aktive FTL-Warp-Signaturen feindlicher Schiffe erkennen
  (für kurze Zeit nach dem Warp-Start sichtbar)

FTL-Sensoren haben ebenfalls einen SR-Wert – größer bedeutet größere Reichweite.
Sie sind von optischen Observatorien getrennt und müssen separat gebaut werden.

---

---

## 6. Deine Spezies – Biochemie und Besiedlung

### 6.1 Die Biochemie-Wahl

Zu Spielbeginn wählst du die **Biochemie deiner Spezies** – die chemische Grundlage
ihres Stoffwechsels. Diese Entscheidung ist dauerhaft und definiert:

- Welche Planeten du **nativ besiedeln** kannst (ohne Habitat)
- Welche Planeten **mit Habitat** erschlossen werden können (Technologie nötig)
- Auf welchen Welten du **Biomasse** erzeugen kannst (Ernährungsgrundlage)

Es gibt fünf Biochemie-Archetypen. Alle sind **wissenschaftlich fundiert** –
entweder als bekannte Erdbiochemie oder als peer-reviewed hypothetische Alternative.

---

### 6.2 Die fünf Archetypen

#### Oxisch – Oxidative Phosphorylierung

> **Reaktion:** C₆H₁₂O₆ + 6 O₂ → 6 CO₂ + 6 H₂O + ATP

Deine Spezies atmet Sauerstoff. Deine Welten sind warm, feucht, mit blauem Himmel.
Das ist die Biochemie der Erde – erprobt, effizient, und am meisten umkämpft,
da alle oxischen Spezies dieselben Welten wollen.

**Ideale Welt:** 250–330 K, 0,5–3 atm, O₂/N₂-Atmosphäre.

*Wissenschaftliche Basis: Aerobe Zellatmung – einziger Archetyp mit direkter
Evidenz auf der Erde. Höchste ATP-Ausbeute (36–38 ATP/Glukose) aller bekannten Pfade.*

---

#### Reduktiv – Methanogenese

> **Reaktion:** CO₂ + 4 H₂ → CH₄ + 2 H₂O (−131 kJ/mol)

Wasserstoff ist dein Treibstoff, Methan dein Ausatemgas. Deine Welten sind kalt
und haben einen rotbraunen Dunstschleier. Gasriesen sind dein strategisches Rückgrat –
sie liefern endlos H₂ und Deuterium.

**Ideale Welt:** 150–280 K, 0,3–8 atm, H₂/CH₄-Atmosphäre.

*Wissenschaftliche Basis: Hydrogenotrophe Methanogenese. Auf der Erde nachgewiesen
bei Archaeen (Methanobacterium, Methanococcus) in anaeroben Hydrothermalsystemen.
Quelle: Seager et al. (2020), arXiv:2009.10247.*

*Treibhauseffekt: H₂ erzeugt durch Kollisions-Absorption (CIA) signifikante Wärme –
+20–60 K pro bar H₂ bei 1 AU. Quelle: Pierrehumbert & Gaidos (2011), ApJL 734(1).*

---

#### Carbonisch – CO₂-Chemolithotrophe

> **Reaktion:** CO₂ + H₂S → CH₂O + S (vereinfacht)

Deine Spezies lebt von CO₂ und nutzt anorganische Verbindungen als Energiequelle.
Kein Sonnenlicht erforderlich – du kannst in totaler Dunkelheit gedeihen. Deine Welten
haben dicke, schweferige Atmosphären wie eine gemäßigte Venus.

**Ideale Welt:** 220–420 K, 1–20 atm, CO₂/N₂/SO₂-Atmosphäre.

*Wissenschaftliche Basis: Chemolithotrophe Mikroorganismen (Thiobacillus, Acidithiobacillus)
sind auf der Erde in Tiefseehydrothermalfeldern und Bergbaugewässern nachgewiesen.
Quelle: Bains (2004), Astrobiology 4(2).*

---

#### Ammonisch – NH₃-Biochemie

> **Reaktion:** N₂ + 3 H₂ → 2 NH₃ (Stickstoff-Fixierung als Energiepfad)

Flüssiges Ammoniak statt Wasser – deine Biochemie löst organische Moleküle in
NH₃ statt H₂O. Deine Welten sind bitterkalt. Deine Atmosphäre ist biologisch fragil:
UV-Strahlung zerstört NH₃ in 10–40 Jahren ohne ständige biologische Nachlieferung.

**Ideale Welt:** 160–270 K, 0,5–4 atm, NH₃/N₂-Atmosphäre.

*Wissenschaftliche Basis: Hypothetisch. NH₃ hat ein Dipolmoment (1,47 D) ähnlich
H₂O (1,85 D) und erlaubt analoge Säure-Base-Reaktionen (NH₄⁺/NH₂⁻ statt H₃O⁺/OH⁻).
Quelle: Bains (2004); Schulze-Makuch & Irwin (2008). Photolytische Instabilität:
Kuhn & Atreya (1979), Icarus 37(1).*

---

#### Chlorisch – Chlor-Reduktase-Metabolismus

> **Reaktion:** Cl₂ + 2 e⁻ → 2 Cl⁻ (Cl₂ als terminaler Elektronenakzeptor)

Das korrosivste Atemgas im Periodensystem. Deine heißen, schwefligen Welten töten
alle anderen sofort. Cl₂ ist kein Treibhausgas – deine Planeten sind wegen
CO₂/SO₂-Hüllen warm, nicht wegen Cl₂ selbst. Als biologisches Ausscheidungsgas
wäre Cl₂ spektroskopisch aus großer Distanz detektierbar – ein zweischneidiges Schwert.

**Ideale Welt:** 300–500 K, 2–30 atm, Cl₂/SO₂/CO₂-Atmosphäre.

*Wissenschaftliche Basis: Hochspekulativ. Erdanalogon: Perchlorate-reduzierende Bakterien
nutzen ClO₄⁻ als Elektronenakzeptor. Cl₂ als Atemgas: keine Primärquelle.
Als Biosignaturgas: Seager et al. (2016), Astrobiology 16(6).*

---

### 6.3 Bewohnbarkeit – Wie wird ein Planet klassifiziert?

Jeder Planet wird bei der Generierung einem Atmosphären-Archetyp zugeordnet.
Beim Spielstart vergleicht das System deinen Biochemie-Typ mit jedem Planetentyp:

| Status | Bedeutung |
|---|---|
| **Nativ bewohnbar** | Archetyp passt + Temperatur + Druck + Gravitation im Toleranzbereich |
| **Bewohnbar mit Habitat** | Archetyp passt nicht, aber Technologie ermöglicht Besiedlung |
| **Ressourcen-Kolonie** | Nicht bewohnbar, aber abbaubare Rohstoffe vorhanden |
| **Unbewohnbar** | Keine wirtschaftliche oder biologische Nutzung ohne Extremtechnologie |

### 6.4 Biomasse

**Biomasse** (Nahrung, biologische Rohstoffe) kannst du nur auf **nativ bewohnbaren**
oder technologisch erschlossenen Welten erzeugen. Jeder Archetyp hat ein eigenes
maximales Biomasse-Potential (0–1 Skala):

| Archetyp | Biomasse-Potential | Grund |
|---|---|---|
| Oxisch | 1,0 | Höchste ATP-Ausbeute; vielzelliges Leben möglich |
| Reduktiv | 0,7 | Gute Energiedichte, aber anaerob begrenzt |
| Carbonisch | 0,5 | Chemolithotrophe Energie < aerobe Energie |
| Ammonisch | 0,4 | NH₃ als Lösungsmittel: langsamere Reaktionskinetik |
| Chlorisch | 0,3 | Hochspekulativ; geringste Evidenz |

---

*Weitere Kapitel werden ergänzt: Flotten & FTL-Reise, Wirtschaft & Produktion,
Schiffsdesign, Kampf, Technologiebaum.*
