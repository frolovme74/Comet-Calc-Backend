package models

import "time"

const (
	CometStatusDraft     = "draft"
	CometStatusPublished = "published"
	CometStatusDeleted   = "deleted"
)

type Comet struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	CometName         string     `gorm:"type:varchar(100);not null" json:"comet_name"`
	CometDescription  string     `gorm:"type:varchar(500);not null;default:''" json:"comet_description"`
	CometStatus       string     `gorm:"type:varchar(20);not null;default:'draft';check:chk_comet_status,comet_status IN ('draft','published','deleted')" json:"comet_status"`
	CometPhoto        string     `gorm:"type:varchar(255);not null;default:''" json:"comet_photo"`
	CometVideo        string     `gorm:"type:varchar(255);not null;default:''" json:"comet_video"`
	OrbitalPeriod     *float64   `gorm:"type:numeric(8,2);check:chk_orbital_period,orbital_period > 0" json:"orbital_period"`
	OrbitEccentricity *float64   `gorm:"type:numeric(4,3);check:chk_orbit_eccentricity,orbit_eccentricity >= 0 AND orbit_eccentricity < 1" json:"orbit_eccentricity"`
	CreatedAt         time.Time  `gorm:"type:timestamp;not null" json:"created_at"`
	CreatorID         uint       `gorm:"not null;uniqueIndex:idx_comets_one_draft_per_creator,where:comet_status = 'draft'" json:"creator_id"`
	FormedAt          *time.Time `gorm:"type:timestamp" json:"formed_at"`

	Creator User `gorm:"foreignKey:CreatorID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT" json:"-"`
}
