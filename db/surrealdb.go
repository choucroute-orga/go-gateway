package db

import (
	"fmt"
	"gateway/configuration"
	"time"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	surrealdb "github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

var logger = logrus.WithFields(logrus.Fields{
	"context": "surrealdb",
})

type SurrealDBHandler struct {
	db   *surrealdb.DB
	conf *configuration.Configuration
}

func NewSurrealDBHandler(conf *configuration.Configuration) (SurrealDBHandler, error) {

	db, err := surrealdb.New(conf.SurrealDBURL)
	if err != nil {
		return SurrealDBHandler{}, err
	}

	err = connectAndPing(conf, db)
	if err != nil {
		return SurrealDBHandler{}, err
	}

	logger.Info("Connected to SurrealDB with url ", conf.SurrealDBURL)
	return SurrealDBHandler{
		db:   db,
		conf: conf,
	}, nil
}

type SurrealUser struct {
	ID            *models.RecordID `json:"id,omitempty"`
	Username      string           `json:"username"`
	UUID          *models.UUID     `json:"uuid,omitempty"`
	Email         string           `json:"email"`
	Password      string           `json:"password"`
	FirstName     string           `json:"firstName"`
	LastName      string           `json:"lastName"`
	EncryptionKey string           `json:"encryptionKey,omitempty"`
}

func (su *SurrealUser) GetId() string {
	return su.Username
}

func (su *SurrealUser) GetUUID() string {
	return su.UUID.String()
}

func (su *SurrealUser) GetEmail() string {
	return su.Email
}

func (su *SurrealUser) GetUsername() string {
	return su.Username
}

func (su *SurrealUser) GetPassword() string {
	return su.Password
}

func (su *SurrealUser) GetFirstName() string {
	return su.FirstName
}

func (su *SurrealUser) GetLastName() string {
	return su.LastName
}

// TODO See how to make the link here
func (su *SurrealUser) GetEncryptionKey() string {
	return su.EncryptionKey
}

type SurrealEncryptionKey struct {
	ID        *models.RecordID `json:"id,omitempty"`
	SecretKey string
}

type SurrealToken struct {
	ID             *models.RecordID `json:"id,omitempty"`
	Value          string
	ExpirationDate string
	//UserId         *models.RecordID `json:"userId,omitempty"`
}

func (st *SurrealToken) GetId() string {
	return st.ID.String()
}

func (st *SurrealToken) GetValue() string {
	return st.Value
}

func (st *SurrealToken) GetExpirationDate() time.Time {
	t, _ := time.Parse(time.RFC3339, st.ExpirationDate)
	return t
}

func (st *SurrealToken) GetUserID() string {
	return st.ID.String()
}

func (sdh SurrealDBHandler) NewUserDTO(email string, username string, password string, firstName string, lastName string, encryptionKey string) SurrealUser {

	r := models.NewRecordID("users", username)
	return SurrealUser{
		ID:        &r,
		Username:  username,
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
	}
}

func (sdh SurrealDBHandler) Ping() error {
	return connectAndPing(sdh.conf, sdh.db)
}

func (sdh SurrealDBHandler) CreateUser(userRequest *UserRequest) (UserDTO, error) {
	// r := models.NewRecordID("users", userRequest.Username)
	uuid, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	u := &SurrealUser{
		ID:        &models.RecordID{ID: userRequest.Username},
		UUID:      &models.UUID{UUID: uuid},
		Username:  userRequest.Username,
		Email:     userRequest.Email,
		Password:  userRequest.Password,
		FirstName: userRequest.FirstName,
		LastName:  userRequest.LastName,
		//EncryptionKey: userRequest.EncryptionKey,
	}

	user, err := surrealdb.Select[SurrealUser](sdh.db, *u.ID)
	if err != nil {
		panic(err)
	}
	logger.Infof("User: %v", user)
	// TODO Check here why the select doesn't work
	if user.ID != nil {
		return nil, fmt.Errorf("user already exists")
	}
	user, err = surrealdb.Create[SurrealUser](sdh.db, models.Table("users"), u)
	if err != nil {
		return nil, fmt.Errorf("user insertion failed")
	}
	return user, nil
}

func (sdh SurrealDBHandler) GetUsername(username string) (UserDTO, error) {
	r := models.NewRecordID("users", username)
	user, err := surrealdb.Select[SurrealUser, models.RecordID](sdh.db, r)
	if err != nil {
		return nil, err
	}
	return user, nil
	// vars := map[string]interface{}{"username": username}
	// res, err := surrealdb.Query[[]SurrealUser](sdh.db, "SELECT * FROM users WHERE username = gridexx", vars)
	// if err != nil {
	// 	return nil, err
	// }
	// if res == nil {
	// 	return nil, fmt.Errorf("user not found")
	// }
	// if len(*res) == 0 {
	// 	return nil, fmt.Errorf("user not found")
	// }
	// for _, u := range *res {
	// 	logger.Infof("User: %v", u)
	// }
	// // u := (*res)[0].Result
	// return nil, err
}

func (sdh SurrealDBHandler) UpsertToken(token *TokenRequest) (TokenDTO, error) {
	s := SurrealToken{
		ID:             &models.RecordID{ID: token.UserID},
		Value:          token.Value,
		ExpirationDate: token.ExpirationDate.Format(time.RFC3339),
	}
	res, err := surrealdb.Upsert[SurrealToken](sdh.db, models.Table("tokens"), s)
	return res, err
}
func (sdh SurrealDBHandler) GetTokenUser(value string, userID string) (TokenDTO, error) {
	r := models.NewRecordID("tokens", userID)
	token, err := surrealdb.Select[SurrealToken, models.RecordID](sdh.db, r)
	if err != nil {
		return nil, err
	}
	if token.Value != value {
		return nil, fmt.Errorf("token not found")
	}
	return token, nil
}
func (sdh SurrealDBHandler) DeleteToken(userID string) error {
	r := models.NewRecordID("tokens", userID)
	_, err := surrealdb.Delete[map[string]interface{}, models.RecordID](sdh.db, r)
	if err != nil {
		return err
	}
	return nil
}

func connectAndPing(conf *configuration.Configuration, db *surrealdb.DB) error {

	// Set the namespace and database
	if err := db.Use(conf.SurrealDBNamespace, conf.SurrealDBDatabase); err != nil {
		return err
	}

	// Create the connection to the database
	// Sign in to authentication `db`
	authData := &surrealdb.Auth{
		Username: conf.SurrealDBUsername, // use your setup username
		Password: conf.SurrealDBPassword, // use your setup password
	}
	token, err := db.SignIn(authData)
	if err != nil {
		return err
	}

	// Check token validity. This is not necessary if you called `SignIn` before. This authenticates the `db` instance too if sign in was
	// not previously called
	if err := db.Authenticate(token); err != nil {
		return err
	}

	// And we can later on invalidate the token if desired
	// defer func(token string) {
	// 	if err := db.Invalidate(); err != nil {
	// 		panic(err)
	// 	}
	// }(token)
	return err
}
