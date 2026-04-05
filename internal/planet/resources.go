package planet

import (
	"math"
	"math/rand/v2"
	"sort"

	"galaxis/internal/model"
)

// Resource IDs – 24 Ressourcengruppen (ADR-006, game-params Sektion 6).
const (
	ResIron        = "iron"
	ResNickel      = "nickel"
	ResTitanium    = "titanium"
	ResTungsten    = "tungsten"
	ResMolybdenum  = "molybdenum"
	ResRareEarth   = "rare_earth"
	ResUranium     = "uranium"
	ResThorium     = "thorium"
	ResSilicon     = "silicon"
	ResAluminum    = "aluminum"
	ResPhosphorus  = "phosphorus"
	ResSulfur      = "sulfur"
	ResCarbon      = "carbon"
	ResWaterIce    = "water_ice"
	ResAmmoniaIce  = "ammonia_ice"
	ResMethaneIce  = "methane_ice"
	ResCO2Ice      = "co2_ice"
	ResNitrogen    = "nitrogen"
	ResHydrogen    = "hydrogen"
	ResHelium      = "helium"
	ResHelium3     = "helium_3"
	ResOrganics    = "organics"
	ResAntimatter  = "antimatter"
	ResExoticMatter = "exotic_matter"
)

type resourceWeight struct {
	base        float64
	planetMod   map[string]float64 // planet_type → multiplier
	starTypeMod map[string]float64 // star_type → multiplier
}

// resourceTable defines base weight and modifiers for each resource.
// [BALANCING] – Werte durch Spieltests justieren.
var resourceTable = map[string]resourceWeight{
	ResIron:        {0.80, map[string]float64{"rocky": 2.0, "ice_giant": 0.1, "gas_giant": 0.05, "asteroid_belt": 1.5}, map[string]float64{"O": 0.5, "B": 0.6, "A": 0.8, "F": 1.0, "G": 1.2, "K": 1.3, "M": 0.9, "WR": 0.4, "RStar": 1.1, "SStar": 1.0, "Pulsar": 0.3, "StellarBH": 0.2, "SMBH": 0.1}},
	ResNickel:      {0.60, map[string]float64{"rocky": 1.8, "ice_giant": 0.1, "gas_giant": 0.05, "asteroid_belt": 1.2}, map[string]float64{"O": 0.4, "B": 0.5, "A": 0.7, "F": 0.9, "G": 1.1, "K": 1.2, "M": 0.8, "WR": 0.3, "RStar": 1.0, "SStar": 0.9, "Pulsar": 0.2, "StellarBH": 0.2, "SMBH": 0.1}},
	ResTitanium:    {0.50, map[string]float64{"rocky": 1.5, "ice_giant": 0.1, "gas_giant": 0.0, "asteroid_belt": 1.0}, map[string]float64{"O": 0.6, "B": 0.7, "A": 0.9, "F": 1.1, "G": 1.2, "K": 1.1, "M": 0.7, "WR": 0.5, "RStar": 1.0, "SStar": 0.9, "Pulsar": 0.3, "StellarBH": 0.2, "SMBH": 0.1}},
	ResTungsten:    {0.30, map[string]float64{"rocky": 2.0, "ice_giant": 0.0, "gas_giant": 0.0, "asteroid_belt": 0.8}, map[string]float64{"O": 0.8, "B": 0.9, "A": 0.8, "F": 0.7, "G": 0.9, "K": 1.2, "M": 1.0, "WR": 0.6, "RStar": 0.8, "SStar": 0.7, "Pulsar": 0.2, "StellarBH": 0.1, "SMBH": 0.1}},
	ResMolybdenum:  {0.30, map[string]float64{"rocky": 1.6, "ice_giant": 0.0, "gas_giant": 0.0, "asteroid_belt": 0.9}, map[string]float64{"O": 0.5, "B": 0.6, "A": 0.8, "F": 1.0, "G": 1.3, "K": 1.2, "M": 0.8, "WR": 0.4, "RStar": 0.9, "SStar": 0.8, "Pulsar": 0.2, "StellarBH": 0.1, "SMBH": 0.1}},
	ResRareEarth:   {0.25, map[string]float64{"rocky": 1.8, "ice_giant": 0.0, "gas_giant": 0.0, "asteroid_belt": 0.5}, map[string]float64{"O": 0.5, "B": 0.6, "A": 0.8, "F": 1.0, "G": 1.2, "K": 1.3, "M": 0.9, "WR": 0.4, "RStar": 1.0, "SStar": 0.9, "Pulsar": 0.3, "StellarBH": 0.2, "SMBH": 0.1}},
	ResUranium:     {0.20, map[string]float64{"rocky": 1.5, "ice_giant": 0.0, "gas_giant": 0.0, "asteroid_belt": 0.4}, map[string]float64{"O": 0.3, "B": 0.4, "A": 0.6, "F": 0.8, "G": 1.0, "K": 1.4, "M": 1.2, "WR": 0.3, "RStar": 1.3, "SStar": 1.2, "Pulsar": 0.5, "StellarBH": 0.3, "SMBH": 0.2}},
	ResThorium:     {0.20, map[string]float64{"rocky": 1.4, "ice_giant": 0.0, "gas_giant": 0.0, "asteroid_belt": 0.4}, map[string]float64{"O": 0.3, "B": 0.4, "A": 0.6, "F": 0.8, "G": 1.0, "K": 1.4, "M": 1.2, "WR": 0.3, "RStar": 1.3, "SStar": 1.2, "Pulsar": 0.5, "StellarBH": 0.3, "SMBH": 0.2}},
	ResSilicon:     {0.70, map[string]float64{"rocky": 2.0, "ice_giant": 0.1, "gas_giant": 0.0, "asteroid_belt": 1.5}, map[string]float64{"O": 0.6, "B": 0.7, "A": 0.9, "F": 1.0, "G": 1.1, "K": 1.1, "M": 0.9, "WR": 0.5, "RStar": 1.0, "SStar": 0.9, "Pulsar": 0.3, "StellarBH": 0.2, "SMBH": 0.1}},
	ResAluminum:    {0.60, map[string]float64{"rocky": 1.8, "ice_giant": 0.1, "gas_giant": 0.0, "asteroid_belt": 1.3}, map[string]float64{"O": 0.6, "B": 0.7, "A": 0.9, "F": 1.0, "G": 1.1, "K": 1.1, "M": 0.9, "WR": 0.5, "RStar": 1.0, "SStar": 0.9, "Pulsar": 0.3, "StellarBH": 0.2, "SMBH": 0.1}},
	ResPhosphorus:  {0.40, map[string]float64{"rocky": 1.6, "ice_giant": 0.2, "gas_giant": 0.1, "asteroid_belt": 0.8}, map[string]float64{"O": 0.4, "B": 0.5, "A": 0.7, "F": 0.9, "G": 1.2, "K": 1.3, "M": 1.0, "WR": 0.3, "RStar": 1.0, "SStar": 0.9, "Pulsar": 0.2, "StellarBH": 0.1, "SMBH": 0.1}},
	ResSulfur:      {0.50, map[string]float64{"rocky": 2.0, "ice_giant": 0.2, "gas_giant": 0.3, "asteroid_belt": 0.6}, map[string]float64{"O": 0.7, "B": 0.8, "A": 0.9, "F": 1.0, "G": 1.0, "K": 0.9, "M": 0.8, "WR": 1.2, "RStar": 0.9, "SStar": 0.8, "Pulsar": 0.4, "StellarBH": 0.3, "SMBH": 0.2}},
	ResCarbon:      {0.50, map[string]float64{"rocky": 1.5, "ice_giant": 0.4, "gas_giant": 0.5, "asteroid_belt": 0.9}, map[string]float64{"O": 0.5, "B": 0.6, "A": 0.8, "F": 0.9, "G": 1.1, "K": 1.0, "M": 0.9, "WR": 0.8, "RStar": 0.9, "SStar": 0.9, "Pulsar": 0.3, "StellarBH": 0.2, "SMBH": 0.1}},
	ResWaterIce:    {0.60, map[string]float64{"rocky": 0.5, "ice_giant": 2.0, "gas_giant": 0.8, "asteroid_belt": 1.5}, map[string]float64{"O": 0.3, "B": 0.4, "A": 0.7, "F": 0.9, "G": 1.2, "K": 1.4, "M": 1.5, "WR": 0.2, "RStar": 1.2, "SStar": 1.0, "Pulsar": 0.8, "StellarBH": 1.0, "SMBH": 0.5}},
	ResAmmoniaIce:  {0.30, map[string]float64{"rocky": 0.2, "ice_giant": 1.8, "gas_giant": 0.6, "asteroid_belt": 0.8}, map[string]float64{"O": 0.2, "B": 0.3, "A": 0.5, "F": 0.7, "G": 0.9, "K": 1.2, "M": 1.5, "WR": 0.1, "RStar": 1.0, "SStar": 0.9, "Pulsar": 0.6, "StellarBH": 0.8, "SMBH": 0.4}},
	ResMethaneIce:  {0.25, map[string]float64{"rocky": 0.1, "ice_giant": 2.0, "gas_giant": 0.4, "asteroid_belt": 0.5}, map[string]float64{"O": 0.1, "B": 0.2, "A": 0.4, "F": 0.6, "G": 0.8, "K": 1.1, "M": 1.4, "WR": 0.1, "RStar": 0.9, "SStar": 0.8, "Pulsar": 0.5, "StellarBH": 0.7, "SMBH": 0.4}},
	ResCO2Ice:      {0.25, map[string]float64{"rocky": 0.6, "ice_giant": 1.2, "gas_giant": 0.3, "asteroid_belt": 0.7}, map[string]float64{"O": 0.3, "B": 0.4, "A": 0.6, "F": 0.8, "G": 1.0, "K": 1.2, "M": 1.3, "WR": 0.2, "RStar": 1.0, "SStar": 0.9, "Pulsar": 0.5, "StellarBH": 0.7, "SMBH": 0.3}},
	ResNitrogen:    {0.40, map[string]float64{"rocky": 1.3, "ice_giant": 0.8, "gas_giant": 0.5, "asteroid_belt": 0.2}, map[string]float64{"O": 0.5, "B": 0.6, "A": 0.8, "F": 0.9, "G": 1.1, "K": 1.1, "M": 1.0, "WR": 0.4, "RStar": 0.9, "SStar": 0.9, "Pulsar": 0.2, "StellarBH": 0.1, "SMBH": 0.1}},
	ResHydrogen:    {0.50, map[string]float64{"rocky": 0.1, "ice_giant": 1.5, "gas_giant": 2.5, "asteroid_belt": 0.1}, map[string]float64{"O": 1.2, "B": 1.1, "A": 1.0, "F": 0.9, "G": 0.9, "K": 0.8, "M": 0.7, "WR": 1.3, "RStar": 0.6, "SStar": 0.6, "Pulsar": 1.0, "StellarBH": 1.5, "SMBH": 2.0}},
	ResHelium:      {0.40, map[string]float64{"rocky": 0.0, "ice_giant": 1.3, "gas_giant": 2.2, "asteroid_belt": 0.0}, map[string]float64{"O": 1.5, "B": 1.4, "A": 1.2, "F": 1.0, "G": 0.9, "K": 0.8, "M": 0.6, "WR": 1.5, "RStar": 0.6, "SStar": 0.6, "Pulsar": 1.2, "StellarBH": 1.5, "SMBH": 2.0}},
	ResHelium3:     {0.15, map[string]float64{"rocky": 0.0, "ice_giant": 1.2, "gas_giant": 2.0, "asteroid_belt": 0.0}, map[string]float64{"O": 1.5, "B": 1.4, "A": 1.2, "F": 1.0, "G": 0.9, "K": 0.8, "M": 0.6, "WR": 1.5, "RStar": 0.6, "SStar": 0.6, "Pulsar": 1.5, "StellarBH": 1.8, "SMBH": 2.5}},
	ResOrganics:    {0.35, map[string]float64{"rocky": 1.2, "ice_giant": 0.6, "gas_giant": 0.3, "asteroid_belt": 0.5}, map[string]float64{"O": 0.3, "B": 0.4, "A": 0.6, "F": 0.8, "G": 1.1, "K": 1.2, "M": 1.0, "WR": 0.3, "RStar": 0.9, "SStar": 0.9, "Pulsar": 0.1, "StellarBH": 0.1, "SMBH": 0.1}},
	ResAntimatter:  {0.05, map[string]float64{"rocky": 0.1, "ice_giant": 0.2, "gas_giant": 0.3, "asteroid_belt": 0.1}, map[string]float64{"O": 0.3, "B": 0.3, "A": 0.2, "F": 0.1, "G": 0.1, "K": 0.1, "M": 0.1, "WR": 0.5, "RStar": 0.1, "SStar": 0.1, "Pulsar": 5.0, "StellarBH": 3.0, "SMBH": 2.0}},
	ResExoticMatter: {0.02, map[string]float64{"rocky": 0.1, "ice_giant": 0.1, "gas_giant": 0.2, "asteroid_belt": 0.1}, map[string]float64{"O": 0.1, "B": 0.1, "A": 0.1, "F": 0.1, "G": 0.1, "K": 0.1, "M": 0.1, "WR": 0.2, "RStar": 0.1, "SStar": 0.1, "Pulsar": 1.0, "StellarBH": 2.0, "SMBH": 5.0}},
}

// sortedResourceIDs for deterministic iteration across all resource deposits.
var sortedResourceIDs []string

func init() {
	sortedResourceIDs = make([]string, 0, len(resourceTable))
	for id := range resourceTable {
		sortedResourceIDs = append(sortedResourceIDs, id)
	}
	sort.Strings(sortedResourceIDs)
}

// depositBaseUnits is the default amount at quality=1.0 (matches game-params common_deposit_units).
const depositBaseUnits = 50_000.0

// GenerateDeposits returns resource deposits for a planet or moon.
// isInnerZone: planet is inside the frost line (suppresses ices, boosts heavy metals).
// Each entry contains amount (initial stock), quality (geological modifier 0–1),
// and max_mines (normally distributed accessibility, clamped 1–10).
func GenerateDeposits(rng *rand.Rand, starType, planetType string, isInnerZone bool) map[string]model.DepositEntry {
	deposits := make(map[string]model.DepositEntry, 10)

	for _, resID := range sortedResourceIDs {
		rw := resourceTable[resID]
		base := rw.base

		if pm, ok := rw.planetMod[planetType]; ok {
			base *= pm
		}
		if sm, ok := rw.starTypeMod[starType]; ok {
			base *= sm
		}

		// Inner zone: suppress ices, enhance refractory metals.
		if isInnerZone {
			switch resID {
			case ResWaterIce, ResAmmoniaIce, ResMethaneIce, ResCO2Ice:
				base *= 0.05
			case ResIron, ResTungsten, ResMolybdenum, ResTitanium:
				base *= 1.5
			}
		}

		if base <= 0 {
			continue
		}

		// Log-normal-like distribution: most deposits small, occasional rich deposits.
		quality := math.Min(base*rng.Float64()*(0.4+rng.Float64()*0.6), 1.0)
		if quality <= 0.02 {
			continue // skip trace deposits
		}

		// max_mines: normally distributed N(4, 2), clamped to [1, 10]. [BALANCING]
		maxMines := int(math.Round(4.0 + rng.NormFloat64()*2.0))
		if maxMines < 1 {
			maxMines = 1
		}
		if maxMines > 10 {
			maxMines = 10
		}

		deposits[resID] = model.DepositEntry{
			Amount:   quality * depositBaseUnits,
			Quality:  quality,
			MaxMines: maxMines,
		}
	}

	return deposits
}
