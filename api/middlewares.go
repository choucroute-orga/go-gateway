package api

import (
	"errors"
	"gateway/db"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type jwtCustomClaims struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	UserID   uint   `json:"userId"`
	jwt.RegisteredClaims
}

func (api *ApiHandler) extractUser(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := c.Get("user").(*jwt.Token)
		claims := user.Claims.(*jwtCustomClaims)
		logger.Debugln(claims.ID)
		t, err := db.GetTokenUser(api.pg, user.Raw, claims.UserID)
		if err != nil {
			logger.WithError(err).Debug("Failed to get token")
			return NewUnauthorizedError(errors.New("You are not authorized to access this resource"))
		}
		if t == nil {
			logger.WithError(err).WithField("token", t).Debug("Failed to get token")
			return NewUnauthorizedError(errors.New("You are not authorized to access this resource"))
		}

		c.Set("user", user) // Set the user into the context
		return next(c)
	}
}
