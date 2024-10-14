package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"gateway/messages"
	"gateway/services"
	"net/http"

	"github.com/labstack/echo/v4"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/codes"
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
	var request postIngredientCatalogRequest

	if err := c.Bind(&request); err != nil {
		FailOnError(l, err, "Request binding failed")
		return NewBadRequestError(err)
	}
	if err := c.Validate(&request); err != nil {
		FailOnError(l, err, "Request validation failed")
		return NewBadRequestError(err)
	}
	encodedRequest, err := json.Marshal(request)
	if err != nil {
		FailOnError(l, err, "Error when trying to Marshal request")
		return NewInternalServerError(err)
	}

	// Send the object to the catalog MS
	resp, err := http.Post(api.conf.CatalogMSURL+"/ingredient", "application/json", bytes.NewBuffer(encodedRequest))

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
			break
		}

		ingredients[i] = Ingredient{
			ID:     ingredientRecipe.ID,
			Name:   ingredientCatalog.Name,
			Type:   ingredientCatalog.Type,
			Amount: ingredientRecipe.Amount,
			Unit:   ingredientRecipe.Unit,
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

func (api *ApiHandler) getRecipes(c echo.Context) error {
	l := logger.WithField("request", "getRecipes")

	l.Info("Getting all recipes " + api.conf.RecipeMSURL)
	// Query the recipe MS to retrieve all recipes
	resp, err := http.Get(api.conf.RecipeMSURL + "/recipe")
	if err != nil {
		FailOnError(l, err, "Error when trying to query recipe MS")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		FailOnError(l, err, "Error when trying to query recipe MS")
		return NewInternalServerError(err)
	}

	// Parse the response body into a slice of Recipe objects
	var recipes interface{}
	err = json.NewDecoder(resp.Body).Decode(&recipes)
	if err != nil {
		FailOnError(l, err, "Error when trying to parse recipe response")
		return NewInternalServerError(err)
	}

	return c.JSON(resp.StatusCode, recipes)
}

func (api *ApiHandler) postRecipe(c echo.Context) error {
	l := logger.WithField("request", "postRecipe")
	l.Info("Posting a new recipe")
	var recipe services.Recipe
	if err := c.Bind(&recipe); err != nil {
		FailOnError(l, err, "Request binding failed")
		return NewBadRequestError(err)
	}
	if err := c.Validate(&recipe); err != nil {
		FailOnError(l, err, "Request validation failed")
		return NewBadRequestError(err)
	}

	encodedRecipe, err := json.Marshal(recipe)
	if err != nil {
		FailOnError(l, err, "Error when trying to Marshal request")
		return NewInternalServerError(err)
	}

	// Send the object to the recipe MS
	resp, err := http.Post(api.conf.RecipeMSURL+"/recipe", "application/json", bytes.NewBuffer(encodedRecipe))

	l.WithFields(logrus.Fields{
		"status": resp.StatusCode,
		"body":   resp.Body,
	})

	if err != nil {
		FailOnError(l, err, "Error when trying to post recipe to recipe MS")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()

	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode POST response")
		return NewInternalServerError(err)
	}

	// Return the response from the recipe MS
	return c.JSON(resp.StatusCode, response)

}

func (api *ApiHandler) getIngredients(c echo.Context) error {
	l := logger.WithField("request", "getRecipes")

	l.Info("Getting all recipes " + api.conf.CatalogMSURL)
	// Query the recipe MS to retrieve all recipes
	resp, err := http.Get(api.conf.CatalogMSURL + "/ingredient")
	if err != nil {
		FailOnError(l, err, "Error when trying to query ingredient MS")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		FailOnError(l, err, "Error when trying to query ingredient MS")
		return NewInternalServerError(err)
	}

	// Parse the response body into a slice of Recipe objects
	var recipes interface{}
	err = json.NewDecoder(resp.Body).Decode(&recipes)
	if err != nil {
		FailOnError(l, err, "Error when trying to parse ingredient response")
		return NewInternalServerError(err)
	}

	return c.JSON(resp.StatusCode, recipes)
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

func (api *ApiHandler) postIngredientToShoppingList(c echo.Context) error {
	context, span := api.tracer.Start(c.Request().Context(), "api.postIngredientToShoppingList")
	defer span.End()
	l := logger.WithContext(context).WithField("request", "postIngredientToShoppingList")

	// Bind the request body to a postIngredientShoppingListRequest object
	var request postIngredientShoppingListRequest
	if err := c.Bind(&request); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Request binding failed")
		FailOnError(l, err, "Request binding failed")
		return NewBadRequestError(err)
	}
	if err := c.Validate(&request); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Request validation failed")
		FailOnError(l, err, "Request validation failed")
		return NewBadRequestError(err)
	}
	ingredientInventory := messages.IngredientShoppingList{
		ID:     request.ID,
		UserID: request.UserID,
		Amount: request.Amount,
		Unit:   string(request.Unit),
	}
	publishCtx, publishSpan := api.tracer.Start(context, "messages.PublishInventoryShoppingListQueue")
	l.WithContext(publishCtx).WithField("ingredientInventory", ingredientInventory).Debug("Publishing ingredient to shopping list")
	err := messages.PublishInventoryShoppingListQueue(l, api.amqp, ingredientInventory)
	publishSpan.End()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "Publishing ingredient to shopping list failded")
		FailOnError(l, err, "Error when trying to publish ingredient to shopping list")
		return NewInternalServerError(err)
	}

	return c.JSON(http.StatusCreated, ingredientInventory)
}

func (api *ApiHandler) postIngredientsForRecipeToShoppingList(c echo.Context) error {

	l := logger.WithField("request", "postIngredientsForRecipeToShoppingList")

	id := c.Param("id")

	userId := c.QueryParam("userId")
	if userId == "" {
		return NewBadRequestError(errors.New("userId query param is required"))
	}

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
			Amount: ingredientRecipe.Amount,
			Unit:   ingredientRecipe.Unit,
		}
	}

	recipeSL := services.AddRecipeShoppingList{
		ID:          id,
		UserID:      userId,
		Ingredients: ingredients,
	}

	encodedRecipe, err := json.Marshal(recipeSL)

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
			Body:        encodedRecipe,
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

	var ingSl []services.IngredientsShoppingList
	// Convert the response interface to a Recipes array
	recipeJson, _ := json.Marshal(response)
	err = json.Unmarshal(recipeJson, &ingSl)
	if err != nil {
		FailOnError(l, err, "Error when trying to parse recipe response")
		return NewInternalServerError(err)
	}

	// Navigate through the ingredients and get the recipe ID
	rSL := []RecipeShoppingList{}

	// First convert the array of Ing to an ingredient response

	ings := []Ingredient{}

	for _, ing := range ingSl {

		// check each quantities in the recipe

		for _, quantity := range ing.Quantities {
			if quantity.RecipeId != "" {

				// Check if the recipe is already in the list
				found := false
				for _, r := range rSL {
					if r.ID == quantity.RecipeId {
						found = true
						break
					}
				}
				// If found, skip to next ingredient

				if !found {
					// Query the recipe MS to retrieve the recipe with the given ID
					recipeUrl := api.conf.RecipeMSURL + "/recipe/" + quantity.RecipeId
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
						break
					}

					recipeResponse := RecipeShoppingList{
						ID:          recipe.ID,
						Name:        recipe.Name,
						Ingredients: []Ingredient{},
					}

					rSL = append(rSL, recipeResponse)
				}

			}
		}

		// Query the Catalog MS to the corresponding ingredient for the recipe
		ingredientUrl := api.conf.CatalogMSURL + "/ingredient/" + ing.ID
		resp, err := http.Get(ingredientUrl)
		if err != nil {
			FailOnError(l, err, "Error when trying to query catalog MS")
			return NewInternalServerError(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			FailOnError(l, err, "Error when requesting the ingredient from catalog MS")
			break
		}

		var ingredientCatalog services.IngredientCatalog
		err = json.NewDecoder(resp.Body).Decode(&ingredientCatalog)
		if err != nil {
			FailOnError(l, err, "Error when trying to parse ingredient response")
			break
		}

		ingredient := Ingredient{
			ID:   ingredientCatalog.ID,
			Name: ingredientCatalog.Name,
			Type: ingredientCatalog.Type,
		}

		// For each quantity, add the ingredient to the recipe
		for _, quantity := range ing.Quantities {
			ingredient.Amount = quantity.Amount
			ingredient.Unit = quantity.Unit
			if quantity.RecipeId != "" {
				// Search for the recipe in the list
				for i, r := range rSL {
					if r.ID == quantity.RecipeId {
						rSL[i].Ingredients = append(rSL[i].Ingredients, ingredient)
					}
				}
			} else {
				ings = append(ings, ingredient)
			}
		}

	}

	// Navigate trou the list to have the recipeIDs
	sL := ShoppingList{
		Recipes:     rSL,
		Ingredients: ings,
	}
	return c.JSON(resp.StatusCode, sL)
}

func (api *ApiHandler) deleteIngredientForRecipeFromShoppingList(c echo.Context) error {
	ingredientId := c.Param("id")
	recipeId := c.Param("recipe_id")
	// allQuantities := c.QueryParam("all")

	l := logger.WithField("request", "deleteIngredientForRecipeFromShoppingList")

	slUrl := api.conf.ShoppingListMSURL + "/ingredient/" + ingredientId
	if recipeId != "" {
		slUrl = api.conf.ShoppingListMSURL + "/recipe/" + recipeId + "/ingredient/" + ingredientId
	}
	req, err := http.NewRequest(http.MethodDelete, slUrl, nil)
	if err != nil {
		FailOnError(l, err, "Error when trying to create DELETE request")
		return NewInternalServerError(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		FailOnError(l, err, "Error when trying to delete ingredient in shopping list")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return c.JSON(http.StatusNoContent, nil)
	}

	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode DELETE response")
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(resp.StatusCode, response)
}

func (api *ApiHandler) getIngredientInventory(c echo.Context) error {
	l := logger.WithField("request", "getIngredientInventory")
	userId := c.QueryParam("userId")
	if userId == "" {
		return NewBadRequestError(errors.New("userId is required"))
	}
	id := c.Param("id")
	invUrl := fmt.Sprintf("%s/inventory/ingredient/%s?userId=%s", api.conf.InventoryMSURL, id, userId)

	resp, err := http.Get(invUrl)
	if err != nil {
		return NewInternalServerError(err)
	}
	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode GET response")
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(resp.StatusCode, response)
}

func (api *ApiHandler) getInventory(c echo.Context) error {
	l := logger.WithField("request", "getInventory")
	userId := c.QueryParam("userId")
	if userId == "" {
		return NewBadRequestError(errors.New("userId is required"))
	}
	invUrl := fmt.Sprintf("%s/inventory/ingredient?userId=%s", api.conf.InventoryMSURL, userId)

	resp, err := http.Get(invUrl)
	if err != nil {
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()
	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode DELETE response")
		return c.JSON(http.StatusInternalServerError, err.Error())
	}
	if resp.StatusCode != http.StatusOK {
		return c.JSON(resp.StatusCode, response)
	}

	// TODO Add information from the catalog
	var inventoryResponse []IngredientInventoryResponse
	// Convert the response interface to a Recipes array
	recipeJson, _ := json.Marshal(response)
	err = json.Unmarshal(recipeJson, &inventoryResponse)
	if err != nil {
		FailOnError(l, err, "Error when trying to parse recipe response")
		return NewInternalServerError(err)
	}
	// Return
	return c.JSON(http.StatusOK, inventoryResponse)

}

func (api *ApiHandler) postInventory(c echo.Context) error {
	l := logger.WithField("request", "postInventory")

	invUrl := fmt.Sprintf("%s/inventory/ingredient", api.conf.InventoryMSURL)

	var request postIngredientInventoryRequest
	if err := c.Bind(&request); err != nil {
		return NewBadRequestError(err)
	}
	if err := c.Validate(request); err != nil {
		return NewBadRequestError(err)
	}

	json_marshal, err := json.Marshal(request)
	if err != nil {
		FailOnError(l, err, "Error when trying to Marshal request")
		return NewInternalServerError(err)
	}

	// Send the object to the catalog MS
	resp, err := http.Post(invUrl, "application/json", bytes.NewBuffer(json_marshal))

	if err != nil {
		FailOnError(l, err, "Error when trying to post ingredient to catalog MS")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()
	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode POST response")
		return NewInternalServerError(err)
	}

	return c.JSON(resp.StatusCode, response)

}

func (api *ApiHandler) putInventory(c echo.Context) error {
	l := logger.WithField("request", "putInventory")

	var request putIngredientInventoryRequest

	if err := c.Bind(&request); err != nil {
		return NewBadRequestError(err)
	}
	if err := c.Validate(request); err != nil {
		// TODO Change to UnprocessableEntityError
		return NewBadRequestError(err)
	}

	invUrl := fmt.Sprintf("%s/inventory/ingredient/%s?userId=%s", api.conf.InventoryMSURL, request.ID, request.UserID)

	encodedRequest, err := json.Marshal(request)
	if err != nil {
		FailOnError(l, err, "Error when trying to Marshal request")
		return NewInternalServerError(err)
	}

	// Send the object to the catalog MS
	req, err := http.NewRequest(http.MethodPut, invUrl, bytes.NewBuffer(encodedRequest))
	// Change the request Header to application/json
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		FailOnError(l, err, "Error when trying to create PUT request")
		return NewInternalServerError(err)
	}
	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		FailOnError(l, err, "Error when trying to post ingredient to catalog MS")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()
	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode POST response")
		return NewInternalServerError(err)
	}
	return c.JSON(resp.StatusCode, response)

}

func (api *ApiHandler) deleteInventory(c echo.Context) error {
	l := logger.WithField("request", "deleteIngredientInventory")
	var delete deleteIngredientInventoryRequest
	if err := c.Bind(&delete); err != nil {
		return NewBadRequestError(err)
	}
	if err := c.Validate(delete); err != nil {
		return NewBadRequestError(err)
	}
	invUrl := fmt.Sprintf("%s/inventory/ingredient/%s/%s", api.conf.InventoryMSURL, delete.ID, delete.UserID)

	req, err := http.NewRequest(http.MethodDelete, invUrl, nil)
	if err != nil {
		FailOnError(l, err, "Error when trying to create DELETE request")
		return NewInternalServerError(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		FailOnError(l, err, "Error when trying to delete ingredient in shopping list")
		return NewInternalServerError(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return c.JSON(http.StatusNoContent, nil)
	}

	var response interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		FailOnError(l, err, "Error when trying to decode DELETE response")
		return c.JSON(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(resp.StatusCode, response)
}
