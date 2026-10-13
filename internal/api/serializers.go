package api

import (
	"time"

	"github.com/frolovme74/Comet-Calc-Backend/internal/models"
	"github.com/frolovme74/Comet-Calc-Backend/internal/repository"
)

type CometResponse struct {
	ID                uint       `json:"id"`
	CometName         string     `json:"comet_name"`
	CometDescription  string     `json:"comet_description"`
	CometStatus       string     `json:"comet_status"`
	CometPhoto        string     `json:"comet_photo"`
	CometPhotoURL     string     `json:"comet_photo_url"`
	CometVideo        string     `json:"comet_video"`
	CometVideoURL     string     `json:"comet_video_url"`
	OrbitalPeriod     *float64   `json:"orbital_period"`
	OrbitEccentricity *float64   `json:"orbit_eccentricity"`
	CreatedAt         time.Time  `json:"created_at"`
	FormedAt          *time.Time `json:"formed_at"`
}

type CometListResponse struct {
	CometResponse
	LikesCount int `json:"likes_count"`
	IsMine     int `json:"is_mine"`
}

type CometFeedResponse struct {
	CometResponse
	LikesCount int `json:"likes_count"`
	IsLiked    int `json:"is_liked"`
}

type PublishCometRequest struct {
	CometDescription  string   `json:"comet_description"`
	OrbitalPeriod     *float64 `json:"orbital_period"`
	OrbitEccentricity *float64 `json:"orbit_eccentricity"`
}

type LikeRequest struct {
	Like *int `json:"like"`
}

type LikeResponse struct {
	CometID    uint  `json:"comet_id"`
	Like       int   `json:"like"`
	LikesCount int64 `json:"likes_count"`
}

type DeleteResponse struct {
	ID          uint   `json:"id"`
	CometStatus string `json:"comet_status"`
}

type RegisterRequest struct {
	Login    string `json:"login"`
	FullName string `json:"full_name"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Login    string `json:"login" example:"frolov"`
	Password string `json:"password" example:"comets2026"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type" example:"Bearer"`
	ExpiresIn   int    `json:"expires_in" example:"3600"`
}

type UserResponse struct {
	ID       uint   `json:"id"`
	Login    string `json:"login"`
	FullName string `json:"full_name"`
}

type MessageResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (a *API) cometResponse(c models.Comet) CometResponse {
	return CometResponse{
		ID:                c.ID,
		CometName:         c.CometName,
		CometDescription:  c.CometDescription,
		CometStatus:       c.CometStatus,
		CometPhoto:        c.CometPhoto,
		CometPhotoURL:     a.storage.URL(c.CometPhoto),
		CometVideo:        c.CometVideo,
		CometVideoURL:     a.storage.URL(c.CometVideo),
		OrbitalPeriod:     c.OrbitalPeriod,
		OrbitEccentricity: c.OrbitEccentricity,
		CreatedAt:         c.CreatedAt,
		FormedAt:          c.FormedAt,
	}
}

func (a *API) cometListResponse(card repository.CometCard) CometListResponse {
	return CometListResponse{CometResponse: a.cometResponse(card.Comet), LikesCount: card.LikesCount, IsMine: card.IsMine}
}

func (a *API) cometFeedResponse(card repository.CometCard) CometFeedResponse {
	return CometFeedResponse{CometResponse: a.cometResponse(card.Comet), LikesCount: card.LikesCount, IsLiked: card.IsLiked}
}

func userResponse(u models.User) UserResponse {
	return UserResponse{ID: u.ID, Login: u.Login, FullName: u.FullName}
}
