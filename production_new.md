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

## 3. Routen

Routen sind **dauerhafte Infrastruktur** mit Durchsatz – keine Einzelfahrten.

```go
type Route struct {
    ID              string
    FromLocationID  string
    ToLocationID    string
    CapacityPerTick float64  // maximaler Durchsatz
}
```

Material im Remote-Lager ist erst planbar, wenn eine Route mit freier Kapazität existiert.

**Ankunftszeit:**
```
Menge / Routenkapazität + Transitzeit = früheste Verfügbarkeit (EarliestArrival)
```

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

## 7. Top-Down Bedarfsauflösung (MRP)

Zwei-Phasen-Ansatz um Lager-Mutation bei Teilfehlern zu verhindern:

### Phase 1: Dry Run (keine Mutation)

```go
func ResolveDemand(itemID string, amount float64, recipes map[string]Recipe,
    totals map[string]float64, visiting map[string]bool) error {

    if visiting[itemID] {
        return fmt.Errorf("cycle detected at: %s", itemID)
    }
    visiting[itemID] = true
    defer delete(visiting, itemID)

    recipe, exists := recipes[itemID]
    if !exists {
        totals[itemID] += amount  // Basisrohstoff
        return nil
    }

    runs := (amount / recipe.BaseYield) / recipe.ProdEfficiency
    for inputID, inputAmount := range recipe.Inputs {
        if err := ResolveDemand(inputID, inputAmount*runs, recipes, totals, visiting); err != nil {
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
            // Nicht abbrechen – Auftrag geht in "waiting_for_mats"
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

## 8. Offene Punkte

- [ ] Distributed Locking (mehrere Server-Instanzen)
- [ ] Umallokation von in-transit Material (Frachter umlenken)
- [ ] Spieler-seitige Prioritätsvergabe zwischen Aufträgen
- [ ] `MinContinuousShare` als spielerseitige Einstellung pro Route?
