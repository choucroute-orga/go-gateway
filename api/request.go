package api

type postIngredientCatalogRequest struct {
	ID       string `json:"id" validate:"omitempty"`
	Name     string `json:"name" validate:"required"`
	ImageURL string `json:"image_url" validate:"required"`
	Type     string `json:"type" validate:"required,oneof=vegetable fruit meat fish dairy spice sugar cereals nuts other"`
}

type postIngredientInventoryRequest struct {
	ID     string  `json:"id" validate:"required"`
	UserID string  `json:"userId" validate:"required"`
	Name   string  `json:"name" validate:"omitempty"`
	Amount float64 `json:"amount" validate:"required,min=0.1"`
	Unit   string  `json:"unit" validate:"oneof=i is cs tbsp tsp g kg ml l"`
}

type UnitRequest string

const (
	UnitItem  UnitRequest = "i"
	UnitItems UnitRequest = "is"
	UnitG     UnitRequest = "g"
	UnitKg    UnitRequest = "kg"
	UnitMl    UnitRequest = "ml"
	UnitL     UnitRequest = "l"
	UnitTsp   UnitRequest = "tsp"
	UnitTbsp  UnitRequest = "tbsp"
	UnitCs    UnitRequest = "cs"
)

type putIngredientInventoryRequest struct {
	ID     string      `param:"id" validate:"required"`
	UserID string      `json:"userId" validate:"required"`
	Name   string      `json:"name" validate:"omitempty"`
	Amount float64     `json:"amount" validate:"required,min=0.1"`
	Unit   UnitRequest `json:"unit" validate:"oneof=i is cs tbsp tsp g kg ml l"`
}

type deleteIngredientInventoryRequest struct {
	ID     string `param:"id" validate:"required"`
	UserID string `param:"userId" validate:"required"`
}
