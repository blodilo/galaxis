# Prompt: Galaxis-Techtree – Erweiterung des graph-Projekts

**Datum:** 2026-03-13
**Adressat:** Claude Code, arbeitend im Verzeichnis `/home/creaminds/Dokumente/graph`
**Zweck:** Minimale Anpassung des graph-Projekts, um den Galaxis-Techtree importieren und exportieren zu können.

---

## Kontext

Das Galaxis-Projekt nutzt das graph-Projekt als Visualisierungs- und Verwaltungswerkzeug
für seinen Technologiebaum. Der Techtree liegt als JSON-LD-Datei vor:

```
/home/creaminds/Dokumente/galaxis/tech-tree_v1.0.jsonld
```

Diese Datei enthält 32 Technologie-Knoten, 8 NodeTypes und 3 RelationTypes.
Jeder Knoten enthält ein `extra_data`-Feld mit einem `galaxis`-Unter-Objekt, das
spielspezifische Daten trägt (Forschungsdauer, Risiko, Ressourcenkosten, Blueprint).

**Beispiel eines Knotens (gekürzt):**

```json
{
  "@id": "kg:node/tech-basic-optics",
  "@type": "kg:Node",
  "label": "Basisoptik",
  "description": "Grundlegende Teleskoptechnologie...",
  "node_type": "kg:node-type/tech-sensors",
  "tags": ["sensors", "tier-1"],
  "business_unit": null,
  "icon_override": null,
  "position_x": null,
  "position_y": null,
  "extra_data": {
    "galaxis": {
      "tier": 1,
      "risk_per_tick": 0.05,
      "risk_category": "Gesichert",
      "research_inputs": {
        "labs_required": 1,
        "scientists_required": 1,
        "credits_per_tick": 50,
        "materials_per_tick": {}
      },
      "research_duration_ticks": 10,
      "blueprint": {
        "unlocks": ["ship-scout-array"],
        "production_inputs": { "credits": 200, "silicon": 50, "aluminum": 30 },
        "production_time_ticks": 3,
        "ship_integration": {
          "mass_tons": 2, "energy_mw": 0.1, "heat_kw": 0.5,
          "voxel_count": 1, "mount_type": "sensor"
        },
        "planetary_integration": null,
        "effects": [
          { "type": "sensor_sr_bonus", "value": 0.07, "description": "Scout-Array SR 0,07 m²" }
        ]
      }
    }
  }
}
```

---

## Analysebefund

Beide relevanten Services ignorieren `extra_data` aktuell vollständig:

**`backend/app/services/graph_import.py`**, Zeile 99:
```python
extra_data={},   # ← hartkodiert leer, ignoriert extra_data aus JSON-LD
```

**`backend/app/services/graph_export.py`**, Zeilen 87–98:
```python
nodes.append({
    "@id": ...,
    "label": ...,
    # ... andere Felder
    "position_y": n.position_y,
    # ← extra_data fehlt komplett
})
```

Datenbankschema und Pydantic-Schemas sind bereits korrekt:
- `nodes.extra_data` (JSON-Spalte) existiert
- `NodeCreate.extra_data: dict` und `NodeRead.extra_data: dict` sind definiert
- **Kein Datenbankmigrationsschritt nötig**

---

## Aufgabe: 4 minimale Änderungen

### Änderung 1 – Import: `extra_data` aus JSON-LD lesen

**Datei:** `backend/app/services/graph_import.py`

Ersetze in der Node-Erstellung (aktuell Zeile 99):
```python
extra_data={},
```
durch:
```python
extra_data=raw.get("extra_data") or {},
```

### Änderung 2 – Import: `properties` von Edges lesen

**Datei:** `backend/app/services/graph_import.py`

Ersetze in der Edge-Erstellung (aktuell Zeile 139):
```python
properties={},
```
durch:
```python
properties=raw.get("properties") or {},
```

### Änderung 3 – Export: `extra_data` in Nodes ausgeben

**Datei:** `backend/app/services/graph_export.py`

Im Node-Dict (aktuell Zeilen 87–98), füge nach `"position_y": n.position_y,` hinzu:
```python
"extra_data": n.extra_data,
```

### Änderung 4 – Export: `properties` in Edges ausgeben

**Datei:** `backend/app/services/graph_export.py`

Im Edge-Dict (aktuell Zeilen 113–120), füge nach `"label": e.label,` hinzu:
```python
"properties": e.properties,
```

---

## Verifikation

Nach den Änderungen muss gelten:

1. **Import-Roundtrip:** Eine JSON-LD-Datei mit `extra_data`-Knoten kann importiert
   und anschließend ohne Datenverlust wieder exportiert werden.

2. **Bestehende Tests laufen durch:** Keine bestehenden Tests dürfen brechen.
   Die Änderungen sind additiv – Nodes ohne `extra_data` in JSON-LD erhalten
   weiterhin `{}` als Standardwert.

3. **Galaxis-Import funktioniert:** Nach den Änderungen kann der Techtree importiert
   werden:

   ```
   POST /api/v1/projects/{id}/import
   Body: Inhalt von /home/creaminds/Dokumente/galaxis/tech-tree_v1.0.jsonld
   ```

   Erwartung: 32 Nodes, 8 NodeTypes, 3 RelationTypes, 33 Edges werden angelegt.
   Jeder Node hat `extra_data.galaxis` mit den Forschungsparametern.

---

## Hinweise

- **Kein Refactoring, keine weiteren Verbesserungen** über diese 4 Zeilen hinaus.
- Keine Änderungen an Pydantic-Schemas, TypeScript-Interfaces oder Datenbankmigrationen.
- Die UNIQUE-Constraint auf Edges (`project_id, relation_type_id, source_id, target_id`)
  erlaubt keine doppelten Kanten — das ist beim Galaxis-Techtree kein Problem
  (alle Kanten sind eindeutig).
- `display_label` auf RelationTypes wird bereits importiert (Zeile 71 im Import-Service)
  und muss nicht geändert werden.
- `description` auf RelationTypes wird im Import **nicht** gespeichert (kein DB-Feld),
  ist aber in der JSON-LD als Kommentar enthalten — das ist korrekt so.
