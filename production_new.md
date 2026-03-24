# Produktions- & Logistik-Modell

**Status:** In Entwicklung  
**Stand:** 2026-03-24

---

## 1. Grundprinzip

Der Spieler denkt in **Flüssen und Kapazitäten**, nicht in Einzelaktionen. Das System übernimmt Buchhaltung und Verteilung automatisch.

---

## 2. Lager-Hierarchie

Material durchläuft eine Pipeline mit klar definierten Zuständen:

```
Remote Lager → [Route/Transit] → Fabriklager → [Allokiert] → Produktion
```

| Zustand | Verfügbar für Auftrag? |
|---|---|
| `stored_remote` | Nein – kein Transportauftrag |
| `in_transit` | Nein – committet, unterwegs |
| `stored_local` | Ja – kann allokiert werden |
| `allocated` | Nein – gebunden an Auftrag |

### Datenmodell Storage

```go
type ItemStock struct {
    Total     float64  // physisch im Fabriklager
    Allocated float64  // in Aufträgen gebunden
}

func (s *ItemStock) Available() float64 {
    return s.Total - s.Allocated
}
```

Spieler-UI zeigt: `Lager: 500 | Gebunden: 300 | Frei: 200`

---

## 3. Routen & Schiffe

### Schicht-Trennung

```
Spieler-Ebene:   Route A→B, 500t/h       (abstrakt, konfigurierbar)
Implementierung: N Schiffe pendeln A↔B   (real, simuliert)
```

Die Route-Kapazität ergibt sich aus: `Anzahl Schiffe × Ladekapazität / Umlaufzeit`

### Datenmodell Route

```go
type Route struct {
    ID              string
    FromLocationID  string
    ToLocationID    string
    CapacityPerTick float64  // ergibt sich aus zugewiesenen Schiffen
    Status          string   // "active", "suspended"
}
```

### Datenmodell Schiff

Ein Schiff pendelt kontinuierlich durch vier Zustände:

```go
type ShipState string

const (
    ShipStateLoading     ShipState = "loading"      // lädt in FromLocation
    ShipStateTransitTo   ShipState = "transit_to"   // unterwegs zur Ziellocation
    ShipStateUnloading   ShipState = "unloading"    // entlädt in ToLocation
    ShipStateTransitBack ShipState = "transit_back" // leer zurück
)

type Ship struct {
    ID       string
    RouteID  string
    State    ShipState
    Cargo    map[string]float64  // aktuelle Ladung
    CargoMax float64
    ETA      int64               // Tick der nächsten Zustandsänderung
}
```

### Ankunftszeit

```
Menge / Routenkapazität + Transitzeit = früheste Verfügbarkeit (EarliestArrival)
```

Material im Remote-Lager ist erst planbar, wenn eine Route mit freier Kapazität existiert.

---

## 4. Auftragstypen

| Typ | Verhalten | Transport-Priorität |
|---|---|---|
| `batch` | Einmalig, zeitkritisch | FIFO, exklusiv |
| `continuous` | Dauerhaft, laufende Produktion | Proportional zum Bedarf |

### Datenmodell Auftrag

```go
type OrderType string

const (
    OrderTypeBatch      OrderType = "batch"
    OrderTypeContinuous OrderType = "continuous"
)

type OrderInputStatus struct {
    Required        float64
    Allocated       float64  // im Fabriklager gesichert
    InTransit       float64  // auf aktiver Route
    EarliestArrival int64    // Tick, ab dem vollständig verfügbar
    Missing         float64  // keine Route vorhanden
}
```

### Voraussetzung bei Auftragseinstellung

- **Material:** muss NICHT vollständig vorhanden sein
- **Fabriken:** müssen vorhanden sein (harte Voraussetzung)

---

## 5. Routenkapazität: Aufteilung

### Grundregel

Continuous-Produktion bekommt immer mindestens **20% der Routenkapazität**:

```go
const MinContinuousShare = 0.20
```

Batch belegt maximal 80%. Wenn Batch-Bedarf größer → Batch wird **verzögert**, Continuous wird **nicht** gedrosselt.

### Algorithmus

```
Gesamtkapazität = Batch-Pool (80%) + Continuous-Pool (20%)
```

1. **Batch:** FIFO nach Priorität, bis Batch-Pool erschöpft
2. **Ungenutzter Batch-Slot** fließt automatisch zu Continuous
3. **Continuous:** proportional auf verfügbare Kapazität aufgeteilt

```go
const MinContinuousShare = 0.20

func (r *Route) AllocateCapacity(batches, continuous []*ProductionOrder) []Warning {
    var warnings []Warning

    batchCap      := r.CapacityPerTick * (1 - MinContinuousShare)
    continuousCap := r.CapacityPerTick * MinContinuousShare

    // Batch: FIFO bis batchCap
    remainingBatch := batchCap
    for _, order := range batches {
        needed := order.TransportNeedPerTick()
        take := math.Min(needed, remainingBatch)
        order.AssignedTransport = take
        remainingBatch -= take

        if take < needed {
            order.RecalculateETA()
            warnings = append(warnings, Warning{
                Type:    "batch_delayed",
                OrderID: order.OrderID,
                Message: fmt.Sprintf("Route %s überlastet – Fertigstellung verzögert", r.ID),
            })
        }
    }

    // Ungenutzter Batch-Slot zu Continuous
    continuousCap += remainingBatch

    // Continuous: proportional
    totalNeed := 0.0
    for _, o := range continuous { totalNeed += o.TransportNeedPerTick() }

    for _, order := range continuous {
        share := (order.TransportNeedPerTick() / totalNeed) * continuousCap
        order.AssignedTransport = share

        if share < order.TransportNeedPerTick() {
            warnings = append(warnings, Warning{
                Type:    "continuous_throttled",
                OrderID: order.OrderID,
                Message: fmt.Sprintf("%.0f%% Kapazität – Produktion gedrosselt",
                    (share/order.TransportNeedPerTick())*100),
            })
        }
    }

    return warnings
}
```

---

## 6. Warnungen & Gegensteuern

| Situation | Typ | Spieler-Aktion |
|---|---|---|
| Batch verdrängt Continuous | `continuous_throttled` | Neue Route / Batch-Prio senken |
| Batch selbst gedrosselt | `batch_delayed` | ETA aktualisiert, Spieler entscheidet |
| Continuous auf 0 | `continuous_starved` | Rot markiert, Fabrik läuft leer |

---

## 7. Rezepte

Rezepte sind durch `ProductID` und `FactoryType` eindeutig. Bessere Technologie = anderes Rezept mit günstigeren Inputs oder höherem Yield – kein Techlevel-Feld.

```go
type RecipeInput struct {
    ItemID string
    Amount float64
}

type Recipe struct {
    RecipeID    string
    ProductID   string
    FactoryType string
    Inputs      []RecipeInput  // mehrere Inputs, explizit als Slice
    BaseYield   float64
}

// Lookup-Key
type RecipeKey struct {
    ProductID   string
    FactoryType string
}

var RecipeBook map[RecipeKey]Recipe
```

### Inputs werden bei Auftragseinstellung kopiert

```go
type ProductionOrder struct {
    OrderID   string
    FactoryID string
    OrderType OrderType

    // Snapshot – unveränderlich nach Erstellung
    RecipeID  string
    Inputs    []RecipeInput        // kopiert aus Rezept, nicht referenziert
    TargetQty float64

    // Laufende Allokation
    AllocatedInputs map[string]float64
}
```

**Vorteile:**
- Rezept kann sich ändern (Forschung), laufende Aufträge bleiben stabil
- Continuous-Produktion rechnet pro Tick direkt mit `order.Inputs` – kein Lookup

---

## 8. Top-Down Bedarfsauflösung (MRP)

Zwei-Phasen-Ansatz um Lager-Mutation bei Teilfehlern zu verhindern:

### Phase 1: Dry Run (keine Mutation)

```go
func ResolveDemand(productID string, amount float64, factory *Factory,
    recipes map[RecipeKey]Recipe, totals map[string]float64, visiting map[string]bool) error {

    if visiting[productID] {
        return fmt.Errorf("cycle detected at: %s", productID)
    }
    visiting[productID] = true
    defer delete(visiting, productID)

    key := RecipeKey{productID, factory.Type}
    recipe, exists := recipes[key]
    if !exists {
        totals[productID] += amount  // Basisrohstoff
        return nil
    }

    runs := amount / recipe.BaseYield
    for _, input := range recipe.Inputs {
        if err := ResolveDemand(input.ItemID, input.Amount*runs, factory, recipes, totals, visiting); err != nil {
            return err
        }
    }
    return nil
}
```

### Phase 2: Commit (atomare Allokation, nur wenn Phase 1 erfolgreich)

```go
func AllocateOrder(factory *Factory, order *ProductionOrder, totals map[string]float64) error {
    factory.mu.Lock()
    defer factory.mu.Unlock()

    for itemID, needed := range totals {
        if factory.Storage[itemID].Available() < needed {
            order.Status = OrderStatusWaiting
            return nil
        }
    }

    for itemID, needed := range totals {
        factory.Storage[itemID].Allocated += needed
        order.AllocatedInputs[itemID] += needed
    }
    order.Status = OrderStatusReady
    return nil
}
```

---

## 9. Fabrikzerstörung

### Grundregel

Wird eine Fabrik zerstört, ist **alles verloren** – kein Teilrückbuch, keine Ausnahmen:

- Fabriklager (frei + allokiert) → vernichtet
- Ladung auf Schiffen, die gerade entladen oder zur Fabrik unterwegs sind → vernichtet
- Alle Aufträge → `cancelled`
- Routen zur Fabrik → `suspended` (Schiffe überleben, Route kann neu zugewiesen werden)

```go
func (f *Factory) Destroy() {
    f.mu.Lock()
    defer f.mu.Unlock()

    for _, order := range f.OrderQueue {
        order.Status = OrderStatusCancelled
    }
    f.OrderQueue = nil
    f.Storage    = nil
}
```

### Schiff-Verhalten bei Fabrikzerstörung

| Schiff-Zustand | Konsequenz |
|---|---|
| Lädt in Remote-Lager | Ladevorgang abbrechen, Schiff frei |
| Transit → Fabrik | Landet, Ladung vernichtet |
| Entlädt in Fabrik | Ladung vernichtet |
| Transit → Remote | Nicht betroffen |

Schiffe überleben stets – nur Ladung und Lagerinhalt gehen verloren.

---

## 10. Entscheidungen & Offene Punkte

### Entschieden

- **Distributed Locking:** nicht nötig – ein Spieler wird immer durch eine Server-Instanz abgedeckt
- **Umallokation:** nicht vorgesehen – einmal allokiert bleibt gebunden
- **Priorität:** Spieler vergibt explizite Prioritäten (1, 2, 3, ...) auf Aufträge; Queue wird vor jedem `DistributeMaterials`-Lauf danach sortiert
- **MinContinuousShare:** spielerseitig einstellbar pro Route
- **Fabrikzerstörung:** alles verloren, kein Teilrückbuch; Schiffe überleben, Routen werden suspendiert

### Offen

- [ ] Tear-Down / Call-Home von Transportflotten (anderer Kontext, noch nicht spezifiziert)
