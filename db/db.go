package db

import (
	"fmt"
	"gateway/configuration"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type UserDTO interface {
	GetId() string
	GetUUID() string
	GetEmail() string
	GetUsername() string
	GetPassword() string
	GetFirstName() string
	GetLastName() string
	GetEncryptionKey() string
}

type TokenDTO interface {
	GetId() string
	GetValue() string
	GetUserID() string
	GetExpirationDate() time.Time
}

type TokenRequest struct {
	Value          string
	ExpirationDate time.Time
	UserID         string
}

type UserRequest struct {
	Email         string
	Username      string
	Password      string
	FirstName     string
	LastName      string
	EncryptionKey string
}

type DBHdandler interface {
	CreateUser(*UserRequest) (UserDTO, error)
	GetUsername(username string) (UserDTO, error)
	UpsertToken(*TokenRequest) (TokenDTO, error)
	GetTokenUser(value string, userID string) (TokenDTO, error)
	DeleteToken(userID string) error
	Ping() error
}

func NewPostgresHandler(conf *configuration.Configuration) (PostgresHandler, error) {

	// Database connexion
	dsn := fmt.Sprintf("host=%v port=%v user=%v password=%v dbname=%v sslmode=%v TimeZone=%v ",
		conf.DBHost,
		conf.DBPort,
		conf.DBUser,
		conf.DBPassword,
		conf.DBName,
		conf.DBSSLMode,
		conf.DBTimezone)

	gormLogger := NewGormLogger()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		logrus.Fatal(err)
		return PostgresHandler{db: db}, err
	}
	if err := AutoMigrate(db); err != nil {
		logrus.Fatal(err)
		return PostgresHandler{db: db}, err
	}
	return PostgresHandler{db: db}, nil

}

func AutoMigrate(db *gorm.DB) error {

	err := db.AutoMigrate(
		&User{},
		&Token{},
	)
	if err != nil {
		logrus.Fatal(err)
	}
	return nil
}
