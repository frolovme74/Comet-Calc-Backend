package models

import "time"

const (
	CometStatusDraft     = "draft"
	CometStatusPublished = "published"
	CometStatusDeleted   = "deleted"
)

type Comet struct {
	ID                uint       `gorm:"primaryKey"`
	CometName         string     `gorm:"type:varchar(100);not null"`
	CometDescription  string     `gorm:"type:varchar(500);not null;default:''"`
	CometStatus       string     `gorm:"type:varchar(20);not null;default:'draft';check:chk_comet_status,comet_status IN ('draft','published','deleted')"`
	CometPhoto        string     `gorm:"type:varchar(255);not null;default:''"`
	CometVideo        string     `gorm:"type:varchar(255);not null;default:''"`
	OrbitalPeriod     *float64   `gorm:"type:numeric(8,2);check:chk_orbital_period,orbital_period > 0"`
	OrbitEccentricity *float64   `gorm:"type:numeric(4,3);check:chk_orbit_eccentricity,orbit_eccentricity >= 0 AND orbit_eccentricity < 1"`
	CreatedAt         time.Time  `gorm:"type:timestamp;not null"`
	CreatorID         uint       `gorm:"not null;uniqueIndex:idx_comets_one_draft_per_creator,where:comet_status = 'draft'"`
	FormedAt          *time.Time `gorm:"type:timestamp"`

	Creator User `gorm:"foreignKey:CreatorID;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT"`
}
