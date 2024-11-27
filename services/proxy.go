package services

import (
	"encoding/json"
	"errors"
	"gateway/graph/model"
	"net/http"

	"github.com/sirupsen/logrus"
)

type RecipeServiceInterface interface {
	GetRecipe(id string) (*Recipe, error)
	GetRecipes() ([]*Recipe, error)
}

type RecipesService struct {
	host string
}

var logger = logrus.WithField("file", "service/proxy")

func NewRecipesService(host string) *RecipesService {
	return &RecipesService{
		host: host,
	}
}

func (r *RecipesService) GetRecipe(id string) (*model.Recipe, error) {

	// Query the recipe MS to retrieve the recipe with the given ID
	recipeUrl := r.host + "/recipe/" + id
	resp, err := http.Get(recipeUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("Recipe not found")
	}

	// Parse the response body into a Recipe object
	var recipe model.Recipe

	// Convert the response interface to a Recipe object
	recipeJson, _ := json.Marshal(response)
	err = json.Unmarshal(recipeJson, &recipe)

	if err != nil {
		return nil, err
	}

	// Query the catalog MS to retrieve the corresponding ingredients for the recipe
	// ingredients, err := api.getIngredientForRecipe(recipe)
	// if err != nil {
	// 	return nil, err
	// }

	// Create a new Recipe object with the aggregated ingredients
	// return Recipe{
	// 	ID:          recipe.ID,
	// 	Name:        recipe.Name,
	// 	Author:      recipe.Author,
	// 	Description: recipe.Description,
	// 	dish:        GetDish(recipe.Dish),
	// 	Servings:    recipe.Servings,
	// 	Metadata:    recipe.Metadata,
	// 	Timers:      recipe.Timers,
	// 	Steps:       recipe.Steps,
	// 	Ingredients: *ingredients,
	// }
	// return &Recipe{}, nil
	return &recipe, nil
}

func (r *RecipesService) GetRecipes() ([]*Recipe, error) {
	return []*Recipe{}, nil
}
