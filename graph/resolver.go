package graph

import "gateway/services"

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

type Resolver struct {
	recipeService *services.RecipesService
}

func NewResolver(recipeHost string) *Resolver {
	recipeService := services.NewRecipesService(recipeHost)
	return &Resolver{
		recipeService: recipeService,
	}
}
