package services

// Here are saved the models of the MS API

// ---           --- //
// *** RECIPE MS *** //
// ---           --- //

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

type Timer struct {
	Name     string `json:"name" validate:"required"`
	Quantity int    `json:"amount" validate:"required,min=1"`
	Units    string `json:"unit" validate:"oneof=seconds minutes hours"`
}

type IngredientRecipe struct {
	ID       string  `json:"id" validate:"omitempty"`
	Quantity float64 `json:"amount" validate:"required,min=0"`
	Units    string  `json:"unit" validate:"oneof=i is cs tbsp tsp g kg"`
}

type Recipe struct {
	ID          string             `json:"id"`
	Name        string             `json:"name" validate:"required"`
	Author      string             `json:"author" validate:"required"` // TODO See If w do a MS for that
	Description string             `json:"description" validate:"required"`
	Dish        Dish               `json:"dish" validate:"oneof=starter main dessert"`
	Servings    int                `json:"servings" validate:"required,min=1"`
	Metadata    map[string]string  `json:"metadata" validate:"omitempty"`
	Timers      []Timer            `json:"timers" validate:"omitempty,dive,required"`
	Steps       []string           `json:"steps" validate:"required"`
	Ingredients []IngredientRecipe `json:"ingredients" validate:"required,dive,required"`
}

// ---           --- //
// *** CATALOG MS *** //
// ---           --- //
type IngredientCatalog struct {
	ID   string `json:"id" validate:"omitempty"`
	Name string `json:"name" validate:"required"`
	Type string `json:"type" validate:"required,oneof=vegetable fruit meat fish dairy spice sugar cereals nuts other"`
}

func GetDish(dish Dish) string {
	switch dish {
	case Starter:
		return "starter"
	case Main:
		return "main"
	case Dessert:
		return "dessert"
	}
	return ""
}

type IngredientShoppingList struct {
	ID     string  `json:"id" validate:"omitempty"`
	Amount float64 `json:"amount" validate:"required,min=0"`
	Unit   string  `json:"unit" validate:"oneof=i is cs tbsp tsp g kg"`
}

type AddRecipeShoppingList struct {
	ID          string                   `json:"id" validate:"required,dive,required"`
	Ingredients []IngredientShoppingList `json:"ingredients" validate:"required,dive,required"`
}

type IngredientInventory struct {
	ID       string  `json:"id" validate:"omitempty"`
	Name     string  `json:"name" validate:"omitempty"`
	Quantity float64 `json:"quantity" validate:"required,min=0"`
	Units    string  `json:"units" validate:"oneof=i is cs tbsp tsp g kg"`
}

type IngredientsShoppingList struct {
	ID         string `json:"id" validate:"omitempty"`
	Quantities []struct {
		Amount   float64 `json:"amount" validate:"required,min=0"`
		Unit     string  `json:"unit" validate:"oneof=i is cs tbsp tsp g kg"`
		RecipeId string  `json:"recipe_id" validate:"omitempty"`
	} `json:"quantities" validate:"required,dive,required"`
}
