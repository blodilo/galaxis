package economy2

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// RecipeInput is a single ingredient in a recipe.
type RecipeInput struct {
	ItemID string  `json:"item_id" yaml:"item_id"`
	Amount float64 `json:"amount"  yaml:"amount"`
}

// Recipe describes a single production recipe.
type Recipe struct {
	RecipeID    string        `yaml:"recipe_id"`
	ProductID   string        `yaml:"product_id"`
	FactoryType string        `yaml:"factory_type"`
	Inputs      []RecipeInput `yaml:"inputs"`
	BaseYield   float64       `yaml:"base_yield"`
	Ticks       int           `yaml:"ticks"`
	Efficiency  float64       `yaml:"efficiency"` // base η (0–1)
}

// RecipeKey identifies a recipe by what it produces and which factory makes it.
type RecipeKey struct {
	ProductID   string
	FactoryType string
}

// RecipeBook is the in-memory lookup table for all recipes.
type RecipeBook map[RecipeKey]*Recipe

// LoadRecipes parses the YAML recipe file and returns a RecipeBook.
func LoadRecipes(path string) (RecipeBook, error) {
	if !filepath.IsAbs(path) {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("economy2: resolve path: %w", err)
		}
		path = abs
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("economy2: read recipes %s: %w", path, err)
	}

	var rf struct {
		Recipes []Recipe `yaml:"recipes"`
	}
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("economy2: parse recipes: %w", err)
	}

	book := make(RecipeBook, len(rf.Recipes))
	for i := range rf.Recipes {
		r := &rf.Recipes[i]
		key := RecipeKey{r.ProductID, r.FactoryType}
		if _, dup := book[key]; dup {
			return nil, fmt.Errorf("economy2: duplicate recipe (%s, %s)", r.ProductID, r.FactoryType)
		}
		book[key] = r
	}
	return book, nil
}
