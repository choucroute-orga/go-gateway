package messages

import "time"

type IngredientShoppingList struct {
	ID     string  `json:"id"`
	UserID string  `json:"userId"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

type AddPriceCatalog struct {
	ProductID string    `json:"productId"`
	ShopID    string    `json:"shopId"`
	Price     float64   `json:"price"`
	Devise    string    `json:"devise"`
	Date      time.Time `json:"date"`
}
