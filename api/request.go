package api

type postIngredientCatalogRequest struct {
	ID       string `json:"id" validate:"omitempty"`
	Name     string `json:"name" validate:"required"`
	ImageURL string `json:"image_url" validate:"required"`
	Type     string `json:"type" validate:"required,oneof=vegetable fruit meat fish dairy spice sugar cereals nuts other"`
}

type postIngredientInventoryRequest struct {
	ID     string  `json:"id" validate:"required"`
	Name   string  `json:"name" validate:"omitempty"`
	Amount float64 `json:"amount" validate:"required,min=0.1"`
	Unit   string  `json:"unit" validate:"oneof=i is cs tbsp tsp g kg"`
}
