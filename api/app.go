package api

import (
	"errors"
	"fmt"
	"gateway/db"
	"gateway/utils"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

func (api *ApiHandler) login(c echo.Context) error {

	l := logger.WithField("request", "login")

	u := new(UserConnectionRequest)
	if err := c.Bind(u); err != nil {
		FailOnError(l, err, "Body param failed")
	}
	if err := c.Validate(u); err != nil {
		return err
	}

	user, err := db.GetUsername(api.pg, u.Username)

	if err != nil {
		return NewInternalServerError(err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(u.Password))

	if err != nil {
		return NewNotFoundError(errors.New("username or password incorrect"))
	}

	expirationDate := time.Now().Add(time.Second * 60 * 24 * 30) // 30 days
	// Set custom claims
	claims := &jwtCustomClaims{
		user.Username,
		user.Email,
		user.ID,
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationDate),
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token and send it as response.
	t, err := token.SignedString([]byte(api.conf.JWTSecret))
	if err != nil {
		return err
	}

	tokenDb := db.Token{
		Value:          t,
		ExpirationDate: expirationDate,
		UserID:         user.ID,
	}
	//Insert the new token in the DB
	db.UpsertToken(api.pg, tokenDb)

	return c.JSON(http.StatusOK, echo.Map{
		"token":      t,
		"email":      user.Email,
		"username":   user.Username,
		"id":         user.ID,
		"expiration": expirationDate,
	})
}

func (api *ApiHandler) logout(c echo.Context) error {
	user := c.Get("user").(*jwt.Token)
	claims := user.Claims.(*jwtCustomClaims)
	userID := claims.UserID
	logger.Info(userID)
	if err := db.DeleteToken(api.pg, userID); err != nil {
		return NewInternalServerError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (api *ApiHandler) restricted(c echo.Context) error {
	// claims := c.Get("user").(*jwtCustomClaims)
	claims := c.Get("user").(*jwt.Token)
	name := claims.Claims.(*jwtCustomClaims).Username

	return c.String(http.StatusOK, "Welcome "+name+"!")
}

func (api *ApiHandler) signup(c echo.Context) error {
	l := logger.WithField("request", "sign-up")

	u := new(UserCreationRequest)
	if err := c.Bind(u); err != nil {
		FailOnError(l, err, "Body param failed")
	}
	if err := c.Validate(u); err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println("Error during hash generation:", err)
		return err
	}

	secretKey, err := utils.GenerateSecretKey()
	if err != nil {
		return NewInternalServerError(err)
	}
	user := db.User{
		Email:     u.Email,
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.FirstName,
		Password:  string(hashedPassword[:]),
		UUID:      uuid.NewString(),
		EncryptionKey: db.EncryptionKey{
			SecretKey: secretKey,
		},
	}

	_, err = db.CreateUser(api.pg, user)
	if err != nil {
		return NewConflictError(err)
	}
	return c.NoContent(http.StatusCreated)
}
