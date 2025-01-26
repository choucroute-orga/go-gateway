package db

import (
	"strconv"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var loger = logrus.WithFields(logrus.Fields{
	"context": "db/query",
})

type PostgresHandler struct {
	db *gorm.DB
}

func (PostgresHandler) NewToken(t TokenRequest) TokenDTO {
	userId, err := strconv.ParseUint(t.UserID, 10, 64)
	if err != nil {
		loger.WithError(err).Error("Error when trying to convert userID to uint")
	}
	return &Token{
		Value:          t.Value,
		ExpirationDate: t.ExpirationDate,
		UserID:         uint(userId),
	}
}

func (PostgresHandler) LogAndReturnError(l *logrus.Entry, result *gorm.DB, action string, modelType string) error {
	if err := result.Error; err != nil {
		l.WithError(err).Error("Error when trying to query database to " + action + " " + modelType)
		return err
	}
	return nil
}

func (ph PostgresHandler) Ping() error {
	sqlDB, err := ph.db.DB()
	if err != nil {
		loger.WithError(err).Error("Error when trying to get database connection")
	}
	err = sqlDB.Ping()
	if err != nil {
		loger.WithError(err).Error("Error when trying to ping database")
	}
	return err
}

func (ph PostgresHandler) CreateUser(userRequest *UserRequest) (UserDTO, error) {

	uuid, err := uuid.NewV4()
	if err != nil {
		loger.WithError(err).Error("Error when trying to generate UUID")
	}

	user := User{
		Username:      userRequest.Username,
		Email:         userRequest.Email,
		Password:      userRequest.Password,
		FirstName:     userRequest.FirstName,
		LastName:      userRequest.LastName,
		EncryptionKey: EncryptionKey{SecretKey: userRequest.EncryptionKey},
		UUID:          uuid.String(),
	}
	result := ph.db.Where("username = ? OR email = ?", userRequest.Username, userRequest.Email).FirstOrCreate(&user)
	loger.Info(result.RowsAffected)
	// User already exists so we throw an error
	if result.RowsAffected == 0 {
		return nil, gorm.ErrDuplicatedKey
	}
	ph.db.Create(&user)
	err = ph.LogAndReturnError(loger, result, "create", "user")
	return &user, err
}

func (ph PostgresHandler) GetUsername(username string) (UserDTO, error) {
	user := new(User)
	result := ph.db.Where("username = ?", username).First(user)
	err := ph.LogAndReturnError(loger, result, "get", "username")
	return user, err
}

func (ph PostgresHandler) UpsertToken(token *TokenRequest) (TokenDTO, error) {

	userId, err := strconv.ParseUint(token.UserID, 10, 64)
	if err != nil {
		loger.WithError(err).Error("Error when trying to convert userID to uint")
	}
	tokenR := Token{
		UserID: uint(userId),
	}
	result := ph.db.Where("user_id = ?", userId).Assign(Token{Value: token.Value, ExpirationDate: token.ExpirationDate}).FirstOrCreate(&tokenR)
	err = ph.LogAndReturnError(loger, result, "upsert", "token")
	return &tokenR, err
}

func (ph PostgresHandler) GetTokenUser(value string, userID string) (TokenDTO, error) {
	// Convert the userID to uint
	userId, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		loger.WithError(err).Error("Error when trying to convert userID to uint")
		return nil, err
	}
	tokenR := new(Token)
	result := ph.db.Where("user_id = ? and value = ?", userId, value).First(tokenR)
	err = ph.LogAndReturnError(loger, result, "get", "token username")
	return tokenR, err
}

func (ph PostgresHandler) DeleteToken(userID string) error {
	// Convert the userID to uint
	userId, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		loger.WithError(err).Error("Error when trying to convert userID to uint")
	}
	result := ph.db.Where("user_id = ?", userId).Delete(&Token{})
	err = ph.LogAndReturnError(loger, result, "delete", "token")
	return err
}
