package api

import (
	"bytes"
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

func (api *ApiHandler) getServicesList() []string {
	return []string{
		api.conf.RecipeMSURL,
		api.conf.CatalogMSURL,
		api.conf.ShoppingListMSURL,
		api.conf.InventoryMSURL,
	}
}

func (api *ApiHandler) getReadyStatus(c echo.Context) error {
	l := logger.WithField("request", "getReadyStatus")

	// Request the health status of each MS
	for _, msUrl := range api.getServicesList() {
		resp, err := http.Get(msUrl + "/health/ready")
		if err != nil {
			FailOnError(l, err, "Error when trying to query MS "+msUrl)
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

func (api *ApiHandler) postIngredientCatalog(c echo.Context) error {

	l := logger.WithField("request", "postIngredientCatalog")

	// Bind the request body to a postIngredientCatalogRequest object
	var request interface{}

	if err := c.Bind(&request); err != nil {
		FailOnError(l, err, "Request binding failed")
		return NewBadRequestError(err)
	}
	if err := c.Validate(&request); err != nil {
		FailOnError(l, err, "Request validation failed")
		return NewBadRequestError(err)
	}
	json_marshal, err := json.Marshal(request)
	if err != nil {
		FailOnError(l, err, "Error when trying to Marshal request")
		return NewInternalServerError(err)
	}

	// Send the object to the catalog MS
	resp, err := http.Post(api.conf.CatalogMSURL+"/ingredient", "application/json", bytes.NewBuffer(json_marshal))

	// Debug log the response
	l.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   resp.Body,
	})

	if err != nil {
		FailOnError(l, err, "Error when trying to post ingredient to catalog MS")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()

	// Parse the response body into an interface
	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode POST response")
		return NewInternalServerError(err)
	}

	return c.JSON(resp.StatusCode, response)
}

// func (api *Apihandler) getRecipes(c echo.Context) error {
// 	l := logger.WithField("request", "getRecipes")

// 	// Query the recipe MS to retrieve all recipes
// 	resp, err := http.Get(api.conf.RecipeMSURL + "/recipe")
// 	if err != nil {
// 		FailOnError(l, err, "Error when trying to query recipe MS")
// 		return NewInternalServerError(err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		FailOnError(l, err, "Error when trying to query recipe MS")
// 		return NewInternalServerError(err)
// 	}

// 	// Parse the response body into a slice of Recipe objects
// 	var recipes []services.Recipe
// 	err = json.NewDecoder(resp.Body).Decode(&recipes)
// 	if err != nil {
// 		FailOnError(l, err, "Error when trying to parse recipe response")
// 		return NewInternalServerError(err)
// 	}

// 	// Create a slice of Recipe objects to return
// 	recipeResponse := make([]Recipe, len(recipes))
// 	for i, recipe := range recipes {
// 		recipeResponse[i] = Recipe{
// 			ID:          recipe.ID,
// 			Name:        recipe.Name,
// }

func (api *ApiHandler) getRecipesByIngredientID(c echo.Context) error {

	l := logger.WithField("request", "getRecipesByIngredientID")

	// Query the recipe MS to retrieve all recipes with ingredient
	resp, err := http.Get(api.conf.RecipeMSURL + "/recipe/ingredient/" + c.Param("id"))
	if err != nil {
		FailOnError(l, err, "Error when trying to query recipe MS")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()

	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode GET response")
		return c.JSON(resp.StatusCode, response)
	}

	if resp.StatusCode != http.StatusOK {
		return c.JSON(resp.StatusCode, response)
	}

	// Parse the response body into a Recipe object
	var recipes []services.Recipe

	// Convert the response interface to a Recipes array
	recipeJson, _ := json.Marshal(response)
	err = json.Unmarshal(recipeJson, &recipes)

	if err != nil {
		FailOnError(l, err, "Error when trying to parse recipe response")
		return NewInternalServerError(err)
	}

	// Create a slice of Recipe objects to return
	recipeResponse := make([]Recipe, len(recipes))
	for i, recipe := range recipes {
		ingredients, err := api.getIngredientForRecipe(recipe)
		if err != nil {
			return err
		}

		recipeResponse[i] = Recipe{
			ID:          recipe.ID,
			Name:        recipe.Name,
			Author:      recipe.Author,
			Description: recipe.Description,
			Dish:        services.GetDish(recipe.Dish),
			Servings:    recipe.Servings,
			Metadata:    recipe.Metadata,
			Timers:      recipe.Timers,
			Steps:       recipe.Steps,
			Ingredients: *ingredients,
		}

	}

	return c.JSON(http.StatusOK, recipeResponse)
}

func (api *ApiHandler) getIngredientForRecipe(recipe services.Recipe) (*[]Ingredient, error) {

	l := logger.WithField("function", "getIngredientForRecipe")

	ingredients := make([]Ingredient, len(recipe.Ingredients))

	ingredientUrl := api.conf.CatalogMSURL + "/ingredient/"
	for i, ingredientRecipe := range recipe.Ingredients {

		resp, err := http.Get(ingredientUrl + ingredientRecipe.ID)
		if err != nil {
			FailOnError(l, err, "Error when trying to query catalog MS")
			return nil, NewInternalServerError(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			FailOnError(l, err, "Error when requesting the ingredient from catalog MS")
			return nil, NewInternalServerError(err)
		}

		// Parse the response body into a slice of IngredientCatalog objects
		var ingredientCatalog services.IngredientCatalog
		err = json.NewDecoder(resp.Body).Decode(&ingredientCatalog)
		if err != nil {
			FailOnError(l, err, "Error when trying to parse ingredient response")
			return nil, NewInternalServerError(err)
		}

		ingredients[i] = Ingredient{
			ID:       ingredientRecipe.ID,
			Name:     ingredientCatalog.Name,
			Type:     ingredientCatalog.Type,
			Quantity: ingredientRecipe.Quantity,
			Units:    ingredientRecipe.Units,
		}
	}

	return &ingredients, nil
}

func (api *ApiHandler) getRecipeByID(c echo.Context) error {
	l := logger.WithField("request", "getRecipe")

	id := c.Param("id")
	// Query the recipe MS to retrieve the recipe with the given ID
	recipeUrl := api.conf.RecipeMSURL + "/recipe/" + id
	resp, err := http.Get(recipeUrl)
	if err != nil {
		FailOnError(l, err, "Error when trying to query recipe MS")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()

	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode GET response")
		return c.JSON(resp.StatusCode, response)
	}

	if resp.StatusCode != http.StatusOK {
		return c.JSON(resp.StatusCode, response)
	}

	// Parse the response body into a Recipe object
	var recipe services.Recipe

	// Convert the response interface to a Recipe object
	recipeJson, _ := json.Marshal(response)
	err = json.Unmarshal(recipeJson, &recipe)

	if err != nil {
		FailOnError(l, err, "Error when trying to parse recipe response")
		return NewInternalServerError(err)
	}

	// Query the catalog MS to retrieve the corresponding ingredients for the recipe
	ingredients, err := api.getIngredientForRecipe(recipe)
	if err != nil {
		return err
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
		Ingredients: *ingredients,
	}

	// Return the aggregated recipe in the response
	return c.JSON(http.StatusOK, recipeResponse)
}
