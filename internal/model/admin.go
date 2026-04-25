package model

type Admin struct {
	Base
	Username     string `gorm:"column:username;size:128;not null;uniqueIndex"`
	PasswordHash string `gorm:"column:password_hash;size:255;not null"`
}

func (Admin) TableName() string {
	return "admins"
}
