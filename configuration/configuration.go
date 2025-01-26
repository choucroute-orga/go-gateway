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
	LogLevel            logrus.Level
	DBName              string
	DBUser              string
	DBPassword          string
	DBPort              string
	DBHost              string
	DBTimezone          string
	DBSSLMode           string
	RecipeMSURL         string
	CatalogMSURL        string
	ShoppingListMSURL   string
	InventoryMSURL      string
	RecipeEndpoint      string
	RabbitURL           string
	TranslateValidation bool
	JWTSecret           string
	OtelServiceName     string
	SurrealDBURL        string
	SurrealDBUsername   string
	SurrealDBPassword   string
	SurrealDBDatabase   string
	SurrealDBNamespace  string
}

func New() *Configuration {

	conf := Configuration{}
	var err error

	logLevel := os.Getenv("LOG_LEVEL")
	if len(logLevel) < 1 || logLevel != "debug" && logLevel != "error" && logLevel != "info" && logLevel != "trace" && logLevel != "warn" {
		logrus.WithFields(logrus.Fields{
			"logLevel": logLevel,
		}).Info("logLevel not conform, use `info` ")
		conf.LogLevel = logrus.InfoLevel
	}

	if logLevel == "debug" {
		conf.LogLevel = logrus.DebugLevel
	} else if logLevel == "error" {
		conf.LogLevel = logrus.ErrorLevel
	} else if logLevel == "info" {
		conf.LogLevel = logrus.InfoLevel
	} else if logLevel == "trace" {
		conf.LogLevel = logrus.TraceLevel
	} else if logLevel == "warn" {
		conf.LogLevel = logrus.WarnLevel
	}

	conf.ListenPort = os.Getenv("API_PORT")
	conf.ListenAddress = os.Getenv("API_ADDRESS")
	conf.ListenRoute = os.Getenv("API_ROUTE")

	conf.DBHost = os.Getenv("POSTGRESQL_HOST")
	conf.DBName = os.Getenv("POSTGRESQL_DATABASE")
	conf.DBPort = os.Getenv("POSTGRESQL_PORT")
	conf.DBUser = os.Getenv("POSTGRESQL_USERNAME")
	conf.DBPassword = os.Getenv("POSTGRESQL_PASSWORD")
	conf.DBTimezone = os.Getenv("POSTGRESQL_TIMEZONE")
	conf.DBSSLMode = "disable"

	conf.RecipeMSURL = os.Getenv("RECIPE_MS_URL")
	conf.CatalogMSURL = os.Getenv("CATALOG_MS_URL")
	conf.ShoppingListMSURL = os.Getenv("SHOPPING_LIST_MS_URL")
	conf.InventoryMSURL = os.Getenv("INVENTORY_MS_URL")

	conf.RabbitURL = os.Getenv("RABBITMQ_URL")

	conf.TranslateValidation, err = strconv.ParseBool(os.Getenv("TRANSLATE_VALIDATION"))

	if err != nil {
		logger.Error("Failed to parse bool for TRANSLATE_VALIDATION")
		os.Exit(1)
	}

	conf.JWTSecret = os.Getenv("JWT_SECRET")

	if len(conf.JWTSecret) < 1 {
		logger.Error("JWT_SECRET is required")
		os.Exit(1)
	}

	conf.OtelServiceName = os.Getenv("OTEL_SERVICE_NAME")

	if len(conf.OtelServiceName) < 1 {
		logger.Error("OTEL_SERVICE_NAME is required")
		os.Exit(1)
	}

	conf.SurrealDBDatabase = os.Getenv("SURREALDB_DATABASE")
	conf.SurrealDBNamespace = os.Getenv("SURREALDB_NAMESPACE")
	conf.SurrealDBPassword = os.Getenv("SURREALDB_PASSWORD")
	conf.SurrealDBURL = os.Getenv("SURREALDB_URL")
	conf.SurrealDBUsername = os.Getenv("SURREALDB_USERNAME")

	return &conf
}
