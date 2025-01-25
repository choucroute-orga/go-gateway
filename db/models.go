package db

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID            uint   `gorm:"primaryKey;autoIncrement:true;uniqueIndex;not null"`
	UUID          string `gorm:"unique"`
	Email         string `gorm:"unique"`
	Username      string `gorm:"unique"`
	Password      string
	FirstName     string
	LastName      string
	EncryptionKey EncryptionKey
}

type EncryptionKey struct {
	gorm.Model
	SecretKey string
	UserID    uint
}

type Token struct {
	ID             uint `gorm:"primaryKey;autoIncrement:true;uniqueIndex;not null"`
	Value          string
	ExpirationDate time.Time
	UserID         uint
}
