package models

type CometLike struct {
	ID      uint `gorm:"primaryKey" json:"id"`
	UserID  uint `gorm:"not null;uniqueIndex:idx_comet_likes_user_comet" json:"user_id"`
	CometID uint `gorm:"not null;uniqueIndex:idx_comet_likes_user_comet" json:"comet_id"`

	User  User  `gorm:"foreignKey:UserID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
	Comet Comet `gorm:"foreignKey:CometID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}
