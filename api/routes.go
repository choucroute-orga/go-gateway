package api

import (
	"encoding/json"
	"gateway/services"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("context", "api/routes")

func (api *ApiHandler) getAliveStatus(c echo.Context) error {
	l := logger.WithField("request", "getAliveStatus")
	status := NewHealthResponse(LiveStatus)
	if err := c.Bind(status); err != nil {
		FailOnError(l, err, "Response binding failed")
		return NewInternalServerError(err)
	}
	l.WithFields(logrus.Fields{
		"action": "getStatus",
		"status": status,
	}).Debug("Health Status ping")

	return c.JSON(http.StatusOK, &status)
}

const (
	RECIPE_MS_URL   = "http://localhost:3001"
	CATALOG_MS_URL  = "http://localhost:3002"
	RECIPE_ENDPOINT = "/recipes/"
)

var LIST_MS_URLS = []string{
	RECIPE_MS_URL,
	CATALOG_MS_URL,
}

func (api *ApiHandler) getReadyStatus(c echo.Context) error {
	l := logger.WithField("request", "getReadyStatus")

	// Request the health status of each MS
	for _, msUrl := range LIST_MS_URLS {
		resp, err := http.Get(msUrl + "/health/ready")
		if err != nil {
			FailOnError(l, err, "Error when trying to query recipe MS")
			return c.JSON(http.StatusServiceUnavailable, NewHealthResponse(NotReadyStatus))
		}

		// Otherwise, check if the MS is ready
		if resp.StatusCode != http.StatusOK {
			FailOnError(l, err, "Service on "+msUrl+" is not ready")
			return c.JSON(http.StatusServiceUnavailable, NewHealthResponse(NotReadyStatus))
		}
	}

	return c.JSON(http.StatusOK, NewHealthResponse(ReadyStatus))
}

func (api *ApiHandler) getRecipeByID(c echo.Context) error {
	l := logger.WithField("request", "getRecipe")

	id := c.Param("id")
	// Query the recipe MS to retrieve the recipe with the given ID
	recipeUrl := RECIPE_MS_URL + RECIPE_ENDPOINT + id
	resp, err := http.Get(recipeUrl)
	if err != nil {
		FailOnError(l, err, "Error when trying to query recipe MS")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		FailOnError(l, err, "Error when trying to query recipe MS")
		return NewInternalServerError(err)
	}

	// Parse the response body into a Recipe object
	var recipe services.Recipe
	err = json.NewDecoder(resp.Body).Decode(&recipe)
	if err != nil {
		FailOnError(l, err, "Error when trying to parse recipe response")
		return NewInternalServerError(err)
	}

	// Query the catalog MS to retrieve the corresponding ingredients for the recipe
	ingredients := make([]Ingredient, len(recipe.Ingredients))

	ingredientUrl := CATALOG_MS_URL + "/ingredient/"
	for i, ingredientRecipe := range recipe.Ingredients {

		resp, err = http.Get(ingredientUrl + ingredientRecipe.ID)
		if err != nil {
			FailOnError(l, err, "Error when trying to query catalog MS")
			return NewInternalServerError(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			FailOnError(l, err, "Error when trying to query catalog MS")
			return NewInternalServerError(err)
		}

		// Parse the response body into a slice of IngredientCatalog objects
		var ingredientCatalog services.IngredientCatalog
		err = json.NewDecoder(resp.Body).Decode(&ingredientCatalog)
		if err != nil {
			FailOnError(l, err, "Error when trying to parse ingredient response")
			return NewInternalServerError(err)
		}

		ingredients[i] = Ingredient{
			ID:          ingredientRecipe.ID,
			Name:        ingredientCatalog.Name,
			Description: ingredientCatalog.Description,
			Type:        ingredientCatalog.Type,
			Quantity:    ingredientRecipe.Quantity,
			Units:       ingredientRecipe.Units,
		}
	}

	// Create a new Recipe object with the aggregated ingredients
	recipeResponse := Recipe{
		ID:          recipe.ID,
		Name:        recipe.Name,
		Author:      recipe.Author,
		Description: recipe.Description,
		Dish:        services.GetDish(recipe.Dish),
		Servings:    recipe.Servings,
		Metadata:    recipe.Metadata,
		Timers:      recipe.Timers,
		Steps:       recipe.Steps,
		Ingredients: ingredients,
	}

	// Return the aggregated recipe in the response
	return c.JSON(http.StatusOK, recipeResponse)
}
