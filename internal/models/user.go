package models

type User struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Login        string `gorm:"type:varchar(50);not null;uniqueIndex" json:"login"`
	FullName     string `gorm:"type:varchar(100);not null" json:"full_name"`
	PasswordHash string `gorm:"type:varchar(100);not null;default:''" json:"-"`
}
