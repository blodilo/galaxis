package planet

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

type rawBiochemYAML struct {
	Archetypes map[string]Archetype `yaml:"archetypes"`
	Balancing  Balancing            `yaml:"balancing"`
}

// LoadBiochem parses biochemistry_archetypes_v*.yaml and returns a BiochemConfig.
// Only enabled archetypes are included.
func LoadBiochem(path string) (*BiochemConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("biochem: read %s: %w", path, err)
	}

	var raw rawBiochemYAML
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("biochem: parse: %w", err)
	}

	cfg := &BiochemConfig{
		Archetypes: make(map[string]*Archetype, len(raw.Archetypes)),
		Balancing:  raw.Balancing,
	}

	for id, a := range raw.Archetypes {
		if !a.Enabled {
			continue
		}
		ac := a // copy to avoid map reference sharing
		cfg.Archetypes[id] = &ac
	}

	// Sort IDs for deterministic CDF (map iteration order is random in Go).
	cfg.SortedIDs = make([]string, 0, len(cfg.Archetypes))
	for id := range cfg.Archetypes {
		cfg.SortedIDs = append(cfg.SortedIDs, id)
	}
	sort.Strings(cfg.SortedIDs)

	return cfg, nil
}
