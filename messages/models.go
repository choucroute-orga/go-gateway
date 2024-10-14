package messages

type IngredientShoppingList struct {
	ID     string  `json:"id"`
	UserID string  `json:"userId"`
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}
