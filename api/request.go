package api

type postIngredientCatalogRequest struct {
	ID       string `json:"id" validate:"omitempty"`
	Name     string `json:"name" validate:"required"`
	ImageURL string `json:"image_url" validate:"required"`
	Type     string `json:"type" validate:"required,oneof=vegetable fruit meat fish dairy spice sugar cereals nuts other"`
}
