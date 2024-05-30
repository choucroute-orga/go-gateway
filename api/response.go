package api

import "gateway/services"

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
	ID          string  `json:"id" validate:"omitempty"`
	Quantity    float64 `json:"quantity" validate:"required,min=0.1"`
	Units       string  `json:"units" validate:"oneof=i is cs tbsp tsp g kg"`
	Name        string  `json:"name" validate:"required"`
	Description string  `json:"description" validate:"required"`
	Type        string  `json:"type" validate:"required,oneof=vegetable fruit meat fish dairy spice sugar cereals nuts other"`
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
	Steps       []string          `json:"steps" validate:"required"`
	Ingredients []Ingredient      `json:"ingredients" validate:"required,dive,required"`
}
