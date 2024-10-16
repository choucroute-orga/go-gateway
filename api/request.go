package api

import (
	"gateway/messages"
	"time"
)

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
	Unit   string  `json:"unit" validate:"oneof=i is cup tbsp tsp g kg ml l"`
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
	UnitCup   UnitRequest = "cup"
)

type putIngredientInventoryRequest struct {
	ID     string      `param:"id" validate:"required"`
	UserID string      `json:"userId" validate:"required"`
	Name   string      `json:"name" validate:"omitempty"`
	Amount float64     `json:"amount" validate:"required,min=0.1"`
	Unit   UnitRequest `json:"unit" validate:"oneof=i is cup tbsp tsp g kg ml l"`
}

type deleteIngredientInventoryRequest struct {
	ID     string `param:"id" validate:"required"`
	UserID string `param:"userId" validate:"required"`
}

type postIngredientShoppingListRequest struct {
	ID     string      `param:"id" validate:"required"`
	UserID string      `json:"userId" validate:"required"`
	Amount float64     `json:"amount" validate:"required,min=0.1"`
	Unit   UnitRequest `json:"unit" validate:"oneof=i is cup tbsp tsp g kg ml l"`
}

type postPriceCatalogRequest struct {
	ProductID string  `json:"productId" validate:"required"`
	ShopID    string  `json:"shopId" validate:"required"`
	Price     float64 `json:"price" validate:"required,min=0.01"`
	Devise    string  `json:"devise" validate:"required,oneof=EUR USD"`
}

func NewAddPriceMessage(price *postPriceCatalogRequest) *messages.AddPriceCatalog {
	return &messages.AddPriceCatalog{
		ProductID: price.ProductID,
		ShopID:    price.ShopID,
		Price:     price.Price,
		Devise:    price.Devise,
		Date:      time.Now(),
	}
}
