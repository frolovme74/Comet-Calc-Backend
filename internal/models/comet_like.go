package models

type CometLike struct {
	UserID  uint `gorm:"primaryKey" json:"user_id"`
	CometID uint `gorm:"primaryKey" json:"comet_id"`

	User  User  `gorm:"foreignKey:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Comet Comet `gorm:"foreignKey:CometID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}
