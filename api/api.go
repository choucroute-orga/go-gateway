package api

import (
	"gateway/configuration"

	"github.com/labstack/echo/v4"
	amqp "github.com/rabbitmq/amqp091-go"
)

type ApiHandler struct {
	amqp *amqp.Connection
	conf *configuration.Configuration
}

func NewApiHandler(amqp *amqp.Connection, conf *configuration.Configuration) *ApiHandler {
	handler := ApiHandler{
		amqp: amqp,
		conf: conf,
	}
	return &handler
}

func (api *ApiHandler) Register(v1 *echo.Group, conf *configuration.Configuration) {

	health := v1.Group("/health")
	health.GET("/alive", api.getAliveStatus)
	health.GET("/live", api.getAliveStatus)
	health.GET("/ready", api.getReadyStatus)

	recipes := v1.Group("/recipe")
	recipes.GET("", api.getRecipes)
	recipes.GET("/:id", api.getRecipeByID)
	recipes.GET("/ingredient/:id", api.getRecipesByIngredientID)
	recipes.POST("", api.postRecipe)
	recipes.DELETE("/:id", api.deleteRecipe)

	ingredient := v1.Group("/ingredient")
	ingredient.GET("", api.getIngredients)
	ingredient.POST("", api.postIngredientCatalog)
	// recipes.GET("/title/:title", api.getRecipeByTitle)
	// recipes.POST("", api.saveRecipe)
	// recipes.PUT("/:id", api.updateRecipe)
	// recipes.DELETE("/:id", api.deleteRecipe)

	shopping_list := v1.Group("/shopping-list")
	shopping_list.GET("", api.getShoppingList)
	shopping_list.POST("/recipe/:id", api.postIngredientsForRecipeToShoppingList)
	shopping_list.DELETE("/ingredient/:id", api.deleteIngredientForRecipeFromShoppingList)
	shopping_list.DELETE("/recipe/:recipe_id/ingredient/:id", api.deleteIngredientForRecipeFromShoppingList)

	inventory := v1.Group("/inventory/ingredient")
	inventory.GET("", api.getInventory)
	inventory.GET("/:id", api.getIngredientInventory)
	inventory.POST("", api.postInventory)
	inventory.PUT("/:id", api.putInventory)
	inventory.DELETE("/:id/user/:userId", api.deleteInventory)
}
