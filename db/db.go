package db

import (
	"fmt"
	"gateway/configuration"

	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func New(conf *configuration.Configuration) (*gorm.DB, error) {

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
		return nil, err
	}
	return db, nil

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
