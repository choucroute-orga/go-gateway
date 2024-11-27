package api

import (
	"gateway/validation"
	"net/http"
	"time"

	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

type CustomValidator struct {
	validator *validator.Validate
}

var trans ut.Translator

func (cv *CustomValidator) Validate(i interface{}) error {

	if err := cv.validator.Struct(i); err != nil {

		errs := err.(validator.ValidationErrors)
		errors := make([]string, len(errs))
		for i, e := range errs {
			errors[i] = e.Translate(trans)
		}
		validationError := &ValidationErrors{
			Error:    err.Error(),
			Code:     http.StatusBadRequest,
			Message:  "Validation error of the Request",
			Errors:   errors,
			IssuedAt: time.Now(),
		}

		return echo.NewHTTPError(http.StatusBadRequest, validationError)
	}
	return nil
}

func New(validation *validation.Validation) *echo.Echo {
	e := echo.New()
	var validate *validator.Validate
	validate, trans = validation.Validate, validation.Trans

	e.Validator = &CustomValidator{validator: validate}

	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowMethods:     []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
		AllowCredentials: true,
	}))
	e.Logger.SetLevel(log.DEBUG)
	e.HideBanner = true
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	return e
}
