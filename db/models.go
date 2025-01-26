package db

import (
	"fmt"
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

func (u *User) GetId() string {
	return fmt.Sprintf("%d", u.ID)
}

func (u *User) GetUUID() string {
	return u.UUID
}

func (u *User) GetEmail() string {
	return u.Email
}

func (u *User) GetUsername() string {
	return u.Username
}

func (u *User) GetPassword() string {
	return u.Password
}

func (u *User) GetFirstName() string {
	return u.FirstName
}

func (u *User) GetLastName() string {
	return u.LastName
}

func (u *User) GetEncryptionKey() string {
	return u.EncryptionKey.SecretKey
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

func (t *Token) GetId() string {
	return fmt.Sprintf("%d", t.ID)
}

func (t *Token) GetValue() string {
	return t.Value
}

func (t *Token) GetUserID() string {
	return fmt.Sprintf("%d", t.UserID)
}

func (t *Token) GetExpirationDate() time.Time {
	return t.ExpirationDate
}
