package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/frolovme74/Comet-Calc-Backend/internal/models"
)

type MediaUploader func(cometID uint) (photo, video string, err error)

func (r *Repository) CreateDraftCometWithMedia(creatorID uint, cometName string, upload MediaUploader) (models.Comet, error) {
	comet := models.Comet{
		CometName:   cometName,
		CometStatus: models.CometStatusDraft,
		CreatedAt:   time.Now(),
		CreatorID:   creatorID,
	}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&comet).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return ErrDraftExists
			}
			return err
		}
		photo, video, err := upload(comet.ID)
		if err != nil {
			return err
		}
		comet.CometPhoto, comet.CometVideo = photo, video
		return tx.Model(&comet).Updates(map[string]any{"comet_photo": photo, "comet_video": video}).Error
	})
	return comet, err
}

func (r *Repository) ownComet(tx *gorm.DB, id, userID uint) (models.Comet, error) {
	var comet models.Comet
	err := tx.Where("id = ? AND comet_status <> ?", id, models.CometStatusDeleted).Take(&comet).Error
	if err != nil {
		return comet, notFound(err)
	}
	if comet.CreatorID != userID {
		return comet, ErrForbidden
	}
	return comet, nil
}

func (r *Repository) PublishComet(id, userID uint, description string, period, eccentricity float64) (models.Comet, error) {
	var comet models.Comet
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var err error
		comet, err = r.ownComet(tx, id, userID)
		if err != nil {
			return err
		}
		if comet.CometStatus != models.CometStatusDraft {
			return ErrWrongStatus
		}
		now := time.Now()
		res := tx.Model(&comet).Where("comet_status = ?", models.CometStatusDraft).Updates(map[string]any{
			"comet_description":  description,
			"orbital_period":     period,
			"orbit_eccentricity": eccentricity,
			"comet_status":       models.CometStatusPublished,
			"formed_at":          now,
		})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrWrongStatus
		}
		comet.CometDescription, comet.OrbitalPeriod, comet.OrbitEccentricity = description, &period, &eccentricity
		comet.CometStatus, comet.FormedAt = models.CometStatusPublished, &now
		return nil
	})
	return comet, err
}

func (r *Repository) SoftDeleteComet(id, userID uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		comet, err := r.ownComet(tx, id, userID)
		if err != nil {
			return err
		}
		return tx.Model(&comet).Update("comet_status", models.CometStatusDeleted).Error
	})
}

func (r *Repository) SetCometLike(userID, cometID uint, like bool) (int64, error) {
	var count int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var comet models.Comet
		err := tx.Where("id = ? AND comet_status = ?", cometID, models.CometStatusPublished).Take(&comet).Error
		if err != nil {
			return notFound(err)
		}
		record := models.CometLike{UserID: userID, CometID: cometID}
		if like {
			err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&record).Error
		} else {
			err = tx.Delete(&record).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&models.CometLike{}).Where("comet_id = ?", cometID).Count(&count).Error
	})
	return count, err
}

func (r *Repository) CreateUser(login, fullName, passwordHash string) (models.User, error) {
	user := models.User{Login: login, FullName: fullName, PasswordHash: passwordHash}
	err := r.db.Create(&user).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return user, ErrLoginTaken
	}
	return user, err
}
