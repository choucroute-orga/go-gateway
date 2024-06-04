package api

import (
	"bytes"
	"encoding/json"
	"gateway/messages"
	"gateway/services"
	"net/http"

	"github.com/labstack/echo/v4"
	amqp "github.com/rabbitmq/amqp091-go"
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

func (api *ApiHandler) getRecipeByTitle(c echo.Context) error {
	l := logger.WithField("request", "getRecipeByTitle")

	title := c.Param("title")
	// Query the recipe MS to retrieve the recipe with the given ID
	recipeUrl := api.conf.RecipeMSURL + "/recipe/title/" + title
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

	return c.JSON(http.StatusOK, recipeResponse)
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

func (api *ApiHandler) deleteRecipe(c echo.Context) error {
	l := logger.WithField("request", "deleteRecipe")

	id := c.Param("id")
	// Query the recipe MS to delete the recipe with the given ID
	recipeUrl := api.conf.RecipeMSURL + "/recipe/" + id
	req, err := http.NewRequest(http.MethodDelete, recipeUrl, nil)
	if err != nil {
		FailOnError(l, err, "Error when trying to create DELETE request")
		return NewInternalServerError(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		FailOnError(l, err, "Error when trying to delete recipe")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()

	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode DELETE response")
		return c.JSON(resp.StatusCode, response)
	}

	return c.JSON(resp.StatusCode, response)
}

func (api *ApiHandler) postIngredientsForRecipeToShoppingList(c echo.Context) error {

	l := logger.WithField("request", "postIngredientsForRecipeToShoppingList")

	id := c.Param("id")

	recipe := services.Recipe{}
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
	recipeJson, _ := json.Marshal(response)
	err = json.Unmarshal(recipeJson, &recipe)
	if err != nil {
		FailOnError(l, err, "Error when trying to parse recipe response")
		return NewInternalServerError(err)
	}

	ingredients := make([]services.IngredientShoppingList, len(recipe.Ingredients))

	for i, ingredientRecipe := range recipe.Ingredients {
		ingredients[i] = services.IngredientShoppingList{
			ID:     ingredientRecipe.ID,
			Amount: ingredientRecipe.Quantity,
			Unit:   ingredientRecipe.Units,
		}
	}

	recipeSL := services.AddRecipeShoppingList{
		ID:          id,
		Ingredients: ingredients,
	}

	json_marshal, err := json.Marshal(recipeSL)

	if err != nil {
		FailOnError(l, err, "Error when trying to Marshal request")
		return NewInternalServerError(err)
	}

	// Send the object to the shopping list queue

	ch, err := api.amqp.Channel()
	if err != nil {
		l.WithError(err).Error("Failed to open a channel")
	}

	defer ch.Close()

	q, err := messages.GetInventoryShoppingListQueue(api.amqp)

	if err != nil {
		l.WithError(err).Error("Failed to declare a queue")
		return NewInternalServerError(err)
	}

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        json_marshal,
		})

	if err != nil {
		l.WithError(err).Error("Failed to publish a message")

	}

	return c.JSON(http.StatusOK, recipeSL)

}

func (api *ApiHandler) getShoppingList(c echo.Context) error {

	l := logger.WithField("request", "getShoppingList")

	// Query the shopping list MS to retrieve all recipes
	resp, err := http.Get(api.conf.ShoppingListMSURL + "/shopping-list")
	if err != nil {
		FailOnError(l, err, "Error when trying to query shopping list MS")
		return NewInternalServerError(err)
	}

	defer resp.Body.Close()

	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode GET response")
		return c.JSON(resp.StatusCode, response)
	}

	return c.JSON(resp.StatusCode, response)
}
