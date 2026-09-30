package models

type CometLike struct {
	UserID  uint `gorm:"primaryKey"`
	CometID uint `gorm:"primaryKey"`

	User  User  `gorm:"foreignKey:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
	Comet Comet `gorm:"foreignKey:CometID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}
