package configuration

import (
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

var logger = logrus.WithFields(logrus.Fields{
	"context": "configuration/configuration",
})

type Configuration struct {
	ListenPort          string
	ListenAddress       string
	ListenRoute         string
	LogLevel            string
	RecipeMSURL         string
	CatalogMSURL        string
	ShoppingListMSURL   string
	InventoryMSURL      string
	RecipeEndpoint      string
	TranslateValidation bool
	JWTSecret           string
}

func New() *Configuration {

	conf := Configuration{}
	var err error

	logLevel := os.Getenv("LOG_LEVEL")
	if len(logLevel) < 1 || logLevel != "debug" && logLevel != "error" && logLevel != "info" && logLevel != "trace" && logLevel != "warn" {
		logrus.WithFields(logrus.Fields{
			"logLevel": logLevel,
		}).Info("logLevel not conform, use `info` ")
		conf.LogLevel = "info"
	} else {
		conf.LogLevel = logLevel
	}

	conf.ListenPort = os.Getenv("API_PORT")
	conf.ListenAddress = os.Getenv("API_ADDRESS")
	conf.ListenRoute = os.Getenv("API_ROUTE")

	conf.RecipeMSURL = os.Getenv("RECIPE_MS_URL")
	conf.CatalogMSURL = os.Getenv("CATALOG_MS_URL")
	conf.ShoppingListMSURL = os.Getenv("SHOPPING_LIST_MS_URL")
	conf.InventoryMSURL = os.Getenv("INVENTORY_MS_URL")

	conf.TranslateValidation, err = strconv.ParseBool(os.Getenv("TRANSLATE_VALIDATION"))

	if err != nil {
		logger.Error("Failed to parse bool for TRANSLATE_VALIDATION")
		os.Exit(1)
	}

	conf.JWTSecret = os.Getenv("JWT_SECRET")

	return &conf
}
