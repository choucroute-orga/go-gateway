package api

import (
	"gateway/services"
	"time"
)

const (
	LiveStatus     = "OK"
	ReadyStatus    = "READY"
	NotReadyStatus = "NOT READY"
)

type HealthResponse struct {
	Status string `json:"status"`
}

func NewHealthResponse(status string) *HealthResponse {
	return &HealthResponse{
		Status: status,
	}
}

type Dish string

const (
	Starter Dish = "starter"
	Main    Dish = "main"
	Dessert Dish = "dessert"
)

type Metadata struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Ingredient struct {
	ID     string  `json:"id" validate:"omitempty"`
	Amount float64 `json:"amount" validate:"required,min=0.1"`
	Unit   string  `json:"unit" validate:"oneof=i is cs tbsp tsp g kg"`
	Name   string  `json:"name" validate:"required"`
	Type   string  `json:"type" validate:"required,oneof=vegetable fruit meat fish dairy spice sugar cereals nuts other"`
}

type Recipe struct {
	ID          string            `json:"id"`
	Name        string            `json:"name" validate:"required"`
	Author      string            `json:"author" validate:"required"` // TODO See If w do a MS for that
	Description string            `json:"description" validate:"required"`
	Dish        string            `json:"dish" validate:"oneof=starter main dessert"`
	Servings    int               `json:"servings" validate:"required,min=1"`
	Metadata    map[string]string `json:"metadata" validate:"omitempty"`
	Timers      []services.Timer  `json:"timers" validate:"omitempty,dive,required"`
	Ingredients []Ingredient      `json:"ingredients" validate:"required,dive,required"`
	Steps       []string          `json:"steps" validate:"required"`
}

type RecipeShoppingList struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Ingredients []Ingredient `json:"ingredients"`
}

type ShoppingList struct {
	Recipes     []RecipeShoppingList `json:"recipes"`
	Ingredients []Ingredient         `json:"ingredients"`
}

type InventoryQuantity struct {
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

type IngredientInventoryResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	InventoryQuantity
}
