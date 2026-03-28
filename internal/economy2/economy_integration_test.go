// Integration-Tests für das Economy2-System.
// Erfordern eine echte PostgreSQL-Datenbank.
//
// Ausführen:
//
//	DATABASE_URL="postgres://galaxis:galaxis_dev@localhost:5432/galaxis?sslmode=disable" \
//	  go test ./internal/economy2/... -v -run TestEconomy
//
// Ohne DATABASE_URL werden alle Tests automatisch übersprungen.
package economy2

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ── Test-Globals ──────────────────────────────────────────────────────────────

var (
	itDB           *pgxpool.Pool
	itCtx          = context.Background()
	itSkip         bool
	itRecipes      RecipeBook
	itMineParams   = MineParams{BaseRate: 5.0, LevelMultiplier: []float64{1.0, 1.5, 2.0}}
	itBootstrapCfg = BootstrapConfig{
		Stock: map[string]float64{
			"iron":     200.0,
			"silicon":  100.0,
			"helium_3": 40.0,
			"steel":    150.0,
		},
		Facilities: []BootstrapFacility{
			{FactoryType: "mine", DepositGoodID: "iron"},
			{FactoryType: "mine", DepositGoodID: "silicon"},
			{FactoryType: "smelter"},
		},
	}
)

func TestMain(m *testing.M) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://galaxis:galaxis_dev@localhost:5432/galaxis?sslmode=disable"
	}

	var err error
	itDB, err = pgxpool.New(itCtx, dbURL)
	if err != nil {
		fmt.Println("economy2 integration: DB connect failed, skipping all tests:", err)
		itSkip = true
		os.Exit(m.Run())
	}
	if err := itDB.Ping(itCtx); err != nil {
		fmt.Println("economy2 integration: DB ping failed, skipping all tests:", err)
		itSkip = true
		itDB.Close()
		os.Exit(m.Run())
	}
	defer itDB.Close()

	// Rezepte laden — Pfad relativ zu diesem Paket (intern/economy2 → ../../)
	itRecipes, err = LoadRecipes("../../econ2_recipes_v1.0.yaml")
	if err != nil {
		fmt.Println("economy2 integration: LoadRecipes failed:", err)
		// kein itSkip — einzelne Tests können Rezepte bei Bedarf überspringen
	}

	os.Exit(m.Run())
}

func requireDB(t *testing.T) {
	t.Helper()
	if itSkip {
		t.Skip("Keine Datenbank verfügbar")
	}
}

// ── Test-Fixtures ─────────────────────────────────────────────────────────────

type itFixtures struct {
	galaxyID uuid.UUID
	starID   uuid.UUID
	planetID uuid.UUID
	playerID uuid.UUID
}

// setupFixtures legt Galaxy → Star → Planet an und räumt nach dem Test auf.
// Der Planet hat resource_deposits: {iron_ore:0.8, silicates:0.5, he3:0.3}.
func setupFixtures(t *testing.T) itFixtures {
	t.Helper()
	fx := itFixtures{playerID: uuid.New()}

	if err := itDB.QueryRow(itCtx,
		`INSERT INTO galaxies (name, seed, config, status) VALUES ($1, 1, '{}', 'ready') RETURNING id`,
		"test-econ2-"+t.Name(),
	).Scan(&fx.galaxyID); err != nil {
		t.Fatalf("Fixture galaxy: %v", err)
	}

	if err := itDB.QueryRow(itCtx,
		`INSERT INTO stars (galaxy_id, x, y, z, star_type, planet_seed) VALUES ($1, 0, 0, 0, 'G', 1) RETURNING id`,
		fx.galaxyID,
	).Scan(&fx.starID); err != nil {
		t.Fatalf("Fixture star: %v", err)
	}

	resDeposits, _ := json.Marshal(map[string]float64{
		"iron":     0.8,
		"silicon":  0.5,
		"helium_3": 0.3,
	})
	if err := itDB.QueryRow(itCtx,
		`INSERT INTO planets (star_id, orbit_index, planet_type, orbit_distance_au, resource_deposits)
		 VALUES ($1, 0, 'rocky', 1.0, $2) RETURNING id`,
		fx.starID, resDeposits,
	).Scan(&fx.planetID); err != nil {
		t.Fatalf("Fixture planet: %v", err)
	}

	t.Cleanup(func() {
		// ON DELETE CASCADE zieht stars, planets, econ2_nodes, econ2_facilities,
		// econ2_orders, econ2_item_stock, planet_deposits mit hoch.
		itDB.Exec(itCtx, `DELETE FROM galaxies WHERE id = $1`, fx.galaxyID)
	})

	return fx
}

// ── Tests ─────────────────────────────────────────────────────────────────────

// TestEconomyBootstrapCreatesDeposits prüft, dass nach RunBootstrap in planet_deposits
// eine Zeile mit Vorkommen-Daten für das Heimat-Planeten existiert.
func TestEconomyBootstrapCreatesDeposits(t *testing.T) {
	requireDB(t)
	fx := setupFixtures(t)

	if _, err := RunBootstrap(itCtx, itDB, fx.playerID, fx.starID, itBootstrapCfg, itRecipes); err != nil {
		t.Fatalf("RunBootstrap: %v", err)
	}

	// planet_deposits muss existieren
	var stateRaw []byte
	if err := itDB.QueryRow(itCtx,
		`SELECT state FROM planet_deposits WHERE planet_id = $1`, fx.planetID,
	).Scan(&stateRaw); err != nil {
		t.Fatalf("planet_deposits nicht angelegt: %v", err)
	}

	var state map[string]depositState
	if err := json.Unmarshal(stateRaw, &state); err != nil {
		t.Fatalf("planet_deposits state parsen: %v", err)
	}

	// iron (quality=0.8): remaining = 0.8 × 50 000 = 40 000
	ds, ok := state["iron"]
	if !ok {
		t.Fatal("iron fehlt in planet_deposits")
	}
	want := 0.8 * 50_000.0
	if ds.Remaining != want {
		t.Errorf("iron remaining = %.1f, want %.1f", ds.Remaining, want)
	}
	if ds.MaxRate <= 0 {
		t.Errorf("iron max_rate = %.2f, want > 0", ds.MaxRate)
	}

	// silicon und helium_3 müssen ebenfalls vorhanden sein
	for _, good := range []string{"silicon", "helium_3"} {
		if _, ok := state[good]; !ok {
			t.Errorf("%s fehlt in planet_deposits", good)
		}
	}

	t.Logf("Deposits: iron=%.0f (max_rate=%.2f), silicon=%.0f, helium_3=%.0f",
		state["iron"].Remaining, state["iron"].MaxRate,
		state["silicon"].Remaining, state["helium_3"].Remaining)
}

// TestEconomyBootstrapPlanetInAssets prüft, dass nach RunBootstrap der Knoten
// auf Planeten-Ebene angelegt wurde und in der Assets-Liste erscheint.
func TestEconomyBootstrapPlanetInAssets(t *testing.T) {
	requireDB(t)
	fx := setupFixtures(t)

	result, err := RunBootstrap(itCtx, itDB, fx.playerID, fx.starID, itBootstrapCfg, itRecipes)
	if err != nil {
		t.Fatalf("RunBootstrap: %v", err)
	}

	// Node muss planetary sein und auf den Heimat-Planeten zeigen
	var level string
	var gotPlanetID *uuid.UUID
	if err := itDB.QueryRow(itCtx,
		`SELECT level, planet_id FROM econ2_nodes WHERE id = $1`, result.NodeID,
	).Scan(&level, &gotPlanetID); err != nil {
		t.Fatalf("Node nicht gefunden: %v", err)
	}
	if level != "planetary" {
		t.Errorf("node.level = %q, want %q", level, "planetary")
	}
	if gotPlanetID == nil || *gotPlanetID != fx.planetID {
		t.Errorf("node.planet_id = %v, want %v", gotPlanetID, fx.planetID)
	}

	// In der Assets-Liste (my-nodes) muss der Node erscheinen
	var count int
	if err := itDB.QueryRow(itCtx,
		`SELECT COUNT(*) FROM econ2_nodes
		 WHERE player_id = $1 AND star_id = $2 AND planet_id = $3`,
		fx.playerID, fx.starID, fx.planetID,
	).Scan(&count); err != nil {
		t.Fatalf("my-nodes query: %v", err)
	}
	if count != 1 {
		t.Errorf("my-nodes: %d Einträge, want 1", count)
	}
	t.Logf("Node %s auf Planet %s (level=%s)", result.NodeID, fx.planetID, level)
}

// TestEconomyBootstrapStarterKitPresent prüft, dass Lager und Anlagen nach
// RunBootstrap vollständig vorhanden sind.
func TestEconomyBootstrapStarterKitPresent(t *testing.T) {
	requireDB(t)
	fx := setupFixtures(t)

	result, err := RunBootstrap(itCtx, itDB, fx.playerID, fx.starID, itBootstrapCfg, itRecipes)
	if err != nil {
		t.Fatalf("RunBootstrap: %v", err)
	}

	// Lager prüfen
	stock, err := NodeStock(itCtx, itDB, result.NodeID)
	if err != nil {
		t.Fatalf("NodeStock: %v", err)
	}
	for itemID, wantQty := range itBootstrapCfg.Stock {
		got := stock[itemID].Total
		if got != wantQty {
			t.Errorf("Lager[%s] = %.1f, want %.1f", itemID, got, wantQty)
		}
	}

	// Anzahl Anlagen prüfen
	var facilityCount int
	if err := itDB.QueryRow(itCtx,
		`SELECT COUNT(*) FROM econ2_facilities
		 WHERE player_id=$1 AND star_id=$2 AND status!='destroyed'`,
		fx.playerID, fx.starID,
	).Scan(&facilityCount); err != nil {
		t.Fatalf("Facilities count: %v", err)
	}
	if facilityCount != len(itBootstrapCfg.Facilities) {
		t.Errorf("Facilities = %d, want %d", facilityCount, len(itBootstrapCfg.Facilities))
	}

	// Mine: deposit_good_id muss korrekt sein
	var cfgRaw []byte
	if err := itDB.QueryRow(itCtx,
		`SELECT config FROM econ2_facilities
		 WHERE player_id=$1 AND factory_type='mine' AND config->>'deposit_good_id'='iron'`,
		fx.playerID,
	).Scan(&cfgRaw); err != nil {
		t.Fatalf("Eisenerz-Mine nicht gefunden: %v", err)
	}
	var cfg FacilityConfig
	_ = json.Unmarshal(cfgRaw, &cfg)
	if cfg.Level != 1 {
		t.Errorf("Mine level = %d, want 1", cfg.Level)
	}

	t.Logf("Lager OK (%d Güter), Anlagen OK (%d)", len(stock), facilityCount)
}

// TestEconomyBuildOrderConsumesResources prüft, dass ein abgeschlossener
// Bauauftrag die Lagerbestände korrekt verringert und eine Anlage erstellt.
func TestEconomyBuildOrderConsumesResources(t *testing.T) {
	requireDB(t)
	if itRecipes == nil {
		t.Skip("Rezepte nicht geladen")
	}
	fx := setupFixtures(t)

	// Schmelze-Baurezept ermitteln
	key := RecipeKey{ProductID: "facility_smelter", FactoryType: "construction"}
	recipe, ok := itRecipes[key]
	if !ok {
		t.Skip("facility_smelter construction recipe nicht gefunden")
	}

	// Lager mit genug Ressourcen befüllen (mind. das Dreifache der Baukosten)
	startStock := map[string]float64{}
	for _, inp := range recipe.Inputs {
		startStock[inp.ItemID] = inp.Amount * 3
	}
	bootstrapCfg := BootstrapConfig{Stock: startStock}
	result, err := RunBootstrap(itCtx, itDB, fx.playerID, fx.starID, bootstrapCfg, itRecipes)
	if err != nil {
		t.Fatalf("RunBootstrap: %v", err)
	}
	nodeID := result.NodeID

	// Inputs allozieren (simuliert MRP)
	for _, inp := range recipe.Inputs {
		if _, err := itDB.Exec(itCtx,
			`UPDATE econ2_item_stock
			 SET allocated = allocated + $1, updated_at = now()
			 WHERE node_id = $2 AND item_id = $3`,
			inp.Amount, nodeID, inp.ItemID,
		); err != nil {
			t.Fatalf("allocate %s: %v", inp.ItemID, err)
		}
	}

	// Bauauftrag anlegen (ready → wird in runBuildTick auf running gesetzt)
	allocMap := map[string]float64{}
	for _, inp := range recipe.Inputs {
		allocMap[inp.ItemID] = inp.Amount
	}
	order := &ProductionOrder{
		PlayerID:        fx.playerID,
		StarID:          fx.starID,
		NodeID:          nodeID,
		OrderType:       OrderTypeBuild,
		Status:          OrderStatusReady,
		RecipeID:        recipe.RecipeID,
		ProductID:       recipe.ProductID,
		FactoryType:     recipe.FactoryType,
		Inputs:          recipe.Inputs,
		BaseYield:       recipe.BaseYield,
		RecipeTicks:     1, // Fertig nach 1 Tick
		Efficiency:      recipe.Efficiency,
		TargetQty:       1,
		AllocatedInputs: allocMap,
		Priority:        5,
	}
	if err := CreateOrder(itCtx, itDB, order); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	// Lager vor dem Tick
	stockBefore, _ := NodeStock(itCtx, itDB, nodeID)

	// Build-Tick ausführen (setzt ready→running, dann fertig weil produced_qty=0 ≥ recipe_ticks=1 → nein)
	// Erster Tick: ready → running, produced_qty 0→1 → 1 ≥ 1 → finishBuildOrder
	if err := runBuildTick(itCtx, itDB, itRecipes); err != nil {
		t.Fatalf("runBuildTick: %v", err)
	}

	// Auftragsstatus prüfen
	var status string
	itDB.QueryRow(itCtx, `SELECT status FROM econ2_orders WHERE id=$1`, order.ID).Scan(&status)
	if status != "completed" {
		t.Errorf("Auftragsstatus = %q, want completed", status)
	}

	// Anlage muss erstellt worden sein
	var facilityCount int
	itDB.QueryRow(itCtx,
		`SELECT COUNT(*) FROM econ2_facilities
		 WHERE player_id=$1 AND factory_type='smelter' AND node_id=$2`,
		fx.playerID, nodeID,
	).Scan(&facilityCount)
	if facilityCount < 1 {
		t.Error("Keine Schmelze nach Abschluss des Bauauftrags gefunden")
	}

	// Lager muss kleiner sein (Inputs verbraucht)
	stockAfter, _ := NodeStock(itCtx, itDB, nodeID)
	for _, inp := range recipe.Inputs {
		before := stockBefore[inp.ItemID].Total
		after := stockAfter[inp.ItemID].Total
		if after >= before {
			t.Errorf("Lager[%s]: vor=%.1f, nach=%.1f — hätte abnehmen müssen",
				inp.ItemID, before, after)
		}
		t.Logf("Lager[%s]: %.1f → %.1f (verbraucht %.1f)", inp.ItemID, before, after, before-after)
	}
}

// TestEconomyMineIncreasesResources prüft, dass eine laufende Mine nach einem
// Tick das Lager auffüllt und das Vorkommen entsprechend abnimmt.
func TestEconomyMineIncreasesResources(t *testing.T) {
	requireDB(t)
	if itRecipes == nil {
		t.Skip("Rezepte nicht geladen")
	}
	fx := setupFixtures(t)

	// Bootstrap: Mine anlegen + Deposits initialisieren
	bootstrapCfg := BootstrapConfig{
		Stock: map[string]float64{"iron_ore": 0},
		Facilities: []BootstrapFacility{
			{FactoryType: "mine", DepositGoodID: "iron_ore"},
		},
	}
	result, err := RunBootstrap(itCtx, itDB, fx.playerID, fx.starID, bootstrapCfg, itRecipes)
	if err != nil {
		t.Fatalf("RunBootstrap: %v", err)
	}
	nodeID := result.NodeID

	// Mine-Anlage ermitteln
	var mineID uuid.UUID
	if err := itDB.QueryRow(itCtx,
		`SELECT id FROM econ2_facilities
		 WHERE player_id=$1 AND factory_type='mine' AND config->>'deposit_good_id'='iron'`,
		fx.playerID,
	).Scan(&mineID); err != nil {
		t.Fatalf("Mine nicht gefunden: %v", err)
	}

	// Mine-Rezept
	recipe := itRecipes[RecipeKey{ProductID: "iron_ore", FactoryType: "mine"}]
	if recipe == nil {
		t.Skip("mine_iron_ore recipe nicht gefunden")
	}

	// Produktionsauftrag anlegen
	order := &ProductionOrder{
		PlayerID:        fx.playerID,
		StarID:          fx.starID,
		NodeID:          nodeID,
		OrderType:       OrderTypeContinuous,
		Status:          OrderStatusRunning,
		RecipeID:        recipe.RecipeID,
		ProductID:       recipe.ProductID,
		FactoryType:     recipe.FactoryType,
		Inputs:          recipe.Inputs,
		BaseYield:       recipe.BaseYield,
		RecipeTicks:     recipe.Ticks,
		Efficiency:      recipe.Efficiency,
		TargetQty:       0,
		AllocatedInputs: map[string]float64{},
		Priority:        5,
	}
	if err := CreateOrder(itCtx, itDB, order); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}

	// Mine-Anlage auf running setzen + Auftrag verknüpfen
	cfgJSON, _ := json.Marshal(FacilityConfig{
		Level:          1,
		DepositGoodID:  "iron_ore",
		TicksRemaining: 1, // produziert beim nächsten Tick
	})
	if _, err := itDB.Exec(itCtx,
		`UPDATE econ2_facilities
		 SET status='running', current_order_id=$1, config=$2, updated_at=now()
		 WHERE id=$3`,
		order.ID, cfgJSON, mineID,
	); err != nil {
		t.Fatalf("Mine auf running setzen: %v", err)
	}

	// Zustand vor dem Tick
	depositsBefore, _ := ReadAllDeposits(itCtx, itDB, fx.planetID)
	stockBefore, _ := NodeStock(itCtx, itDB, nodeID)
	t.Logf("Vor Tick: Vorkommen=%.1f, Lager=%.1f",
		depositsBefore["iron_ore"].Remaining, stockBefore["iron_ore"].Total)

	// Produktions-Tick ausführen
	if err := runProductionTick(itCtx, itDB, itRecipes, itMineParams); err != nil {
		t.Fatalf("runProductionTick: %v", err)
	}

	// Lager muss gestiegen sein
	stockAfter, _ := NodeStock(itCtx, itDB, nodeID)
	if stockAfter["iron_ore"].Total <= stockBefore["iron_ore"].Total {
		t.Errorf("Lager stieg nicht: vor=%.1f, nach=%.1f",
			stockBefore["iron_ore"].Total, stockAfter["iron_ore"].Total)
	}

	// Vorkommen muss gesunken sein
	depositsAfter, _ := ReadAllDeposits(itCtx, itDB, fx.planetID)
	if depositsAfter["iron_ore"].Remaining >= depositsBefore["iron_ore"].Remaining {
		t.Errorf("Vorkommen sank nicht: vor=%.1f, nach=%.1f",
			depositsBefore["iron_ore"].Remaining, depositsAfter["iron_ore"].Remaining)
	}

	extracted := depositsBefore["iron_ore"].Remaining - depositsAfter["iron_ore"].Remaining
	gained := stockAfter["iron_ore"].Total - stockBefore["iron_ore"].Total
	t.Logf("Nach Tick: Vorkommen=%.1f (−%.2f), Lager=%.1f (+%.2f)",
		depositsAfter["iron_ore"].Remaining, extracted,
		stockAfter["iron_ore"].Total, gained)
}
