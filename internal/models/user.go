package models

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Login    string `gorm:"type:varchar(50);not null;uniqueIndex"`
	FullName string `gorm:"type:varchar(100);not null"`
}
