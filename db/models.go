package db

import "time"

type User struct {
	ID        uint   `gorm:"primaryKey;autoIncrement:true;uniqueIndex;not null"`
	Email     string `gorm:"unique"`
	Username  string `gorm:"unique"`
	Password  string
	FirstName string
	LastName  string
}

type Token struct {
	ID             uint `gorm:"primaryKey;autoIncrement:true;uniqueIndex;not null"`
	Value          string
	ExpirationDate time.Time
	UserID         uint
}
