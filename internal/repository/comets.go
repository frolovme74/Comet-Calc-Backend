package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/frolovme74/Comet-Calc-Backend/internal/models"
)

type CometCard struct {
	models.Comet
	LikesCount int
	IsMine     int
	IsLiked    int
}

func (r *Repository) publishedComets(userID uint) *gorm.DB {
	likes := r.db.Model(&models.CometLike{}).Select("COUNT(*)").Where("comet_likes.comet_id = comets.id")
	liked := r.db.Model(&models.CometLike{}).Select("COUNT(*)").Where("comet_likes.comet_id = comets.id AND comet_likes.user_id = ?", userID)
	return r.db.Model(&models.Comet{}).
		Select("comets.*, (?) AS likes_count, (comets.creator_id = ?)::int AS is_mine, (?) AS is_liked", likes, userID, liked).
		Where("comets.comet_status = ?", models.CometStatusPublished)
}

func (r *Repository) PublishedComets(userID uint, periodFrom, periodTo *float64) ([]CometCard, error) {
	query := r.publishedComets(userID)
	if periodFrom != nil {
		query = query.Where("comets.orbital_period >= ?", *periodFrom)
	}
	if periodTo != nil {
		query = query.Where("comets.orbital_period <= ?", *periodTo)
	}
	var cards []CometCard
	err := query.Order("comets.id").Find(&cards).Error
	return cards, err
}

func (r *Repository) FirstPublishedComet(userID uint) (CometCard, error) {
	var card CometCard
	err := r.publishedComets(userID).Order("comets.id").Limit(1).Take(&card).Error
	return card, notFound(err)
}

func (r *Repository) PublishedCometByID(userID, id uint) (CometCard, error) {
	var card CometCard
	err := r.publishedComets(userID).Where("comets.id = ?", id).Limit(1).Take(&card).Error
	return card, notFound(err)
}

func (r *Repository) NextPublishedComet(userID, id uint) (CometCard, error) {
	var card CometCard
	err := r.publishedComets(userID).Where("comets.id > ?", id).Order("comets.id").Limit(1).Take(&card).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return r.FirstPublishedComet(userID)
	}
	return card, err
}

func (r *Repository) DraftComet(creatorID uint) (models.Comet, error) {
	var comet models.Comet
	err := r.db.Where("creator_id = ? AND comet_status = ?", creatorID, models.CometStatusDraft).Take(&comet).Error
	return comet, notFound(err)
}

func (r *Repository) CreateDraftComet(creatorID uint, cometName string) (models.Comet, error) {
	comet := models.Comet{
		CometName:   cometName,
		CometStatus: models.CometStatusDraft,
		CreatedAt:   time.Now(),
		CreatorID:   creatorID,
	}
	err := r.db.Create(&comet).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return comet, ErrDraftExists
	}
	return comet, err
}

func (r *Repository) PublishDraftComet(creatorID uint, description string, period, eccentricity float64) (models.Comet, error) {
	comet, err := r.DraftComet(creatorID)
	if err != nil {
		return comet, err
	}
	res := r.db.Model(&comet).Where("comet_status = ?", models.CometStatusDraft).Updates(map[string]any{
		"comet_description":  description,
		"orbital_period":     period,
		"orbit_eccentricity": eccentricity,
		"comet_status":       models.CometStatusPublished,
		"formed_at":          time.Now(),
	})
	if res.Error != nil {
		return comet, res.Error
	}
	if res.RowsAffected == 0 {
		return comet, ErrNotFound
	}
	return comet, nil
}

func (r *Repository) DeleteComet(ctx context.Context, id uint) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	res, err := sqlDB.ExecContext(ctx,
		"UPDATE comets SET comet_status = $1 WHERE id = $2 AND comet_status = $3",
		models.CometStatusDeleted, id, models.CometStatusPublished)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
