package models

import (
	"fmt"
	"math/rand"
)

// AttributeNames defines the 20 named attributes for auction items.
var AttributeNames = [20]string{
	"quality",
	"rarity",
	"condition",
	"age",
	"weight",
	"durability",
	"aesthetics",
	"brand_value",
	"market_demand",
	"authenticity",
	"craftsmanship",
	"material_grade",
	"color_vibrancy",
	"size",
	"portability",
	"warranty_score",
	"eco_friendliness",
	"innovation",
	"cultural_value",
	"resale_potential",
}

// Item represents an auction item with 20 named attributes.
type Item struct {
	ID         int                `json:"id"`
	Name       string             `json:"name"`
	Attributes map[string]float64 `json:"attributes"`
}

// NewItem creates an auction item with randomized attribute values.
func NewItem(id int, rng *rand.Rand) Item {
	attrs := make(map[string]float64, 20)
	for _, name := range AttributeNames {
		attrs[name] = rng.Float64() * 100 // value 0-100
	}
	return Item{
		ID:         id,
		Name:       fmt.Sprintf("Item-%d", id),
		Attributes: attrs,
	}
}

// AttributeValues returns the attribute values as an ordered array (matching AttributeNames order).
// Used by bidders to calculate weighted scores efficiently.
func (item Item) AttributeValues() [20]float64 {
	var values [20]float64
	for i, name := range AttributeNames {
		values[i] = item.Attributes[name]
	}
	return values
}
