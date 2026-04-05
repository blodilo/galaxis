package economy2

import (
	"encoding/json"
	"fmt"
	"os"
)

// DeployableItemDef describes what a deployable factory item becomes when deployed.
type DeployableItemDef struct {
	FactoryType   string  `json:"factory_type"`
	DepositGoodID string  `json:"deposit_good_id,omitempty"`
	Level         int     `json:"level"`
	MaxRate       float64 `json:"max_rate,omitempty"`
}

// ItemCatalog maps item_id → deployment definition.
// Only deployable items are listed; regular goods are absent.
type ItemCatalog map[string]DeployableItemDef

// LoadItemCatalog reads the items JSON file and returns an ItemCatalog.
func LoadItemCatalog(path string) (ItemCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("economy2: read item catalog %s: %w", path, err)
	}
	var catalog ItemCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("economy2: parse item catalog: %w", err)
	}
	return catalog, nil
}
