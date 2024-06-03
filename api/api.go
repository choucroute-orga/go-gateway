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
	recipes.GET("/:id", api.getRecipeByID)
	recipes.GET("/ingredient/:id", api.getRecipesByIngredientID)
	recipes.DELETE("/:id", api.deleteRecipe)

	ingredient := v1.Group("/ingredient")
	ingredient.POST("", api.postIngredientCatalog)
	// recipes.GET("/title/:title", api.getRecipeByTitle)
	// recipes.POST("", api.saveRecipe)
	// recipes.PUT("/:id", api.updateRecipe)
	// recipes.DELETE("/:id", api.deleteRecipe)

	shopping_list := v1.Group("/shopping-list")
	shopping_list.GET("", api.getShoppingList)
	shopping_list.POST("/recipe/:id", api.postIngredientsForRecipeToShoppingList)
	
}
