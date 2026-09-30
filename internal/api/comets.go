package api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/frolovme74/Comet-Calc-Backend/internal/repository"
	"github.com/frolovme74/Comet-Calc-Backend/internal/session"
	"github.com/frolovme74/Comet-Calc-Backend/internal/storage"
)

const (
	maxPhotoSize = 10 << 20
	maxVideoSize = 50 << 20
)

var (
	photoTypes = map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif"}
	videoTypes = map[string]string{"video/mp4": ".mp4", "video/webm": ".webm"}
)

func (a *API) GetComets(w http.ResponseWriter, r *http.Request) {
	var from, to *float64
	if v, ok := parseNumber(r.URL.Query().Get("min_orbital_period")); ok {
		from = &v
	} else if r.URL.Query().Get("min_orbital_period") != "" {
		fail(w, http.StatusBadRequest, "min_orbital_period должен быть числом")
		return
	}
	if v, ok := parseNumber(r.URL.Query().Get("max_orbital_period")); ok {
		to = &v
	} else if r.URL.Query().Get("max_orbital_period") != "" {
		fail(w, http.StatusBadRequest, "max_orbital_period должен быть числом")
		return
	}
	if from != nil && to != nil && *from > *to {
		from, to = to, from
	}

	cards, err := a.repo.PublishedComets(session.CurrentUser().ID, from, to)
	if err != nil {
		a.serverError(w, err)
		return
	}
	resp := make([]CometResponse, 0, len(cards))
	for _, c := range cards {
		resp = append(resp, a.cardResponse(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) GetCometFeed(w http.ResponseWriter, r *http.Request) {
	userID := session.CurrentUser().ID
	idStr := strings.Trim(r.PathValue("id"), "/")

	var (
		card repository.CometCard
		err  error
	)
	if idStr == "" {
		card, err = a.repo.FirstPublishedComet(userID)
	} else {
		id, convErr := strconv.ParseUint(idStr, 10, 63)
		if convErr != nil {
			fail(w, http.StatusBadRequest, "id должен быть положительным числом")
			return
		}
		if r.URL.Query().Get("next") == "true" {
			card, err = a.repo.NextPublishedComet(userID, uint(id))
		} else {
			card, err = a.repo.PublishedCometByID(userID, uint(id))
		}
	}
	if err != nil {
		a.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a.cardResponse(card))
}

func (a *API) GetCometDraft(w http.ResponseWriter, r *http.Request) {
	comet, err := a.repo.DraftComet(session.CurrentUser().ID)
	if errors.Is(err, repository.ErrNotFound) {
		fail(w, http.StatusNotFound, "У пользователя нет черновика")
		return
	}
	if err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a.cometResponse(comet))
}

type uploadedFile struct {
	file        multipart.File
	size        int64
	contentType string
	extension   string
}

func readUpload(r *http.Request, field string, allowed map[string]string, maxSize int64, what string) (*uploadedFile, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		return nil, fmt.Errorf("приложите %s в поле %s", what, field)
	}
	if header.Size > maxSize {
		file.Close()
		return nil, fmt.Errorf("%s больше %d МБ", what, maxSize>>20)
	}
	head := make([]byte, 512)
	n, _ := file.Read(head)
	contentType := http.DetectContentType(head[:n])
	extension, ok := allowed[contentType]
	if _, err := file.Seek(0, 0); err != nil || !ok {
		file.Close()
		return nil, fmt.Errorf("%s: неподдерживаемый формат %s", what, contentType)
	}
	return &uploadedFile{file: file, size: header.Size, contentType: contentType, extension: extension}, nil
}

func (a *API) CreateComet(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxPhotoSize+maxVideoSize+(1<<20))
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		fail(w, http.StatusBadRequest, "Ожидается multipart/form-data с полями comet_name, comet_photo, comet_video")
		return
	}
	defer r.MultipartForm.RemoveAll()

	for key := range r.MultipartForm.Value {
		if key == "comet_photo" || key == "comet_video" {
			fail(w, http.StatusBadRequest, fmt.Sprintf("Поле %s должно быть файлом", key))
			return
		}
		if key != "comet_name" {
			fail(w, http.StatusBadRequest, fmt.Sprintf("Поле %s нельзя передавать при создании", key))
			return
		}
	}
	name := strings.TrimSpace(r.FormValue("comet_name"))
	if !utf8.ValidString(name) || name == "" || utf8.RuneCountInString(name) > 100 {
		fail(w, http.StatusBadRequest, "comet_name: название обязательно, до 100 символов")
		return
	}
	photo, err := readUpload(r, "comet_photo", photoTypes, maxPhotoSize, "фото")
	if err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	defer photo.file.Close()
	video, err := readUpload(r, "comet_video", videoTypes, maxVideoSize, "видео")
	if err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	defer video.file.Close()

	userID := session.CurrentUser().ID
	if _, err := a.repo.DraftComet(userID); err == nil {
		fail(w, http.StatusConflict, "У пользователя уже есть черновик")
		return
	}

	ctx := r.Context()
	var uploaded []string
	comet, err := a.repo.CreateDraftCometWithMedia(userID, name, func(cometID uint) (string, string, error) {
		photoName := storage.ObjectName(cometID, "photo", photo.extension)
		if err := a.storage.Put(ctx, photoName, photo.file, photo.size, photo.contentType); err != nil {
			return "", "", fmt.Errorf("загрузка фото в Minio: %w", err)
		}
		uploaded = append(uploaded, photoName)
		videoName := storage.ObjectName(cometID, "video", video.extension)
		if err := a.storage.Put(ctx, videoName, video.file, video.size, video.contentType); err != nil {
			return "", "", fmt.Errorf("загрузка видео в Minio: %w", err)
		}
		uploaded = append(uploaded, videoName)
		return photoName, videoName, nil
	})
	if err != nil {
		for _, name := range uploaded {
			_ = a.storage.Remove(context.Background(), name)
		}
		a.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, a.cometResponse(comet))
}

func (a *API) PublishComet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		fail(w, http.StatusBadRequest, "id должен быть положительным числом")
		return
	}
	var req PublishCometRequest
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	description := strings.TrimSpace(req.CometDescription)
	var problems []string
	if description == "" || utf8.RuneCountInString(description) > 500 {
		problems = append(problems, "comet_description: обязательно, до 500 символов")
	}
	var period, eccentricity float64
	if req.OrbitalPeriod == nil {
		problems = append(problems, "orbital_period: обязательно")
	} else if period = math.Round(*req.OrbitalPeriod*100) / 100; period <= 0 || period > 999999.99 {
		problems = append(problems, "orbital_period: положительное число лет, не больше 999999.99")
	}
	if req.OrbitEccentricity == nil {
		problems = append(problems, "orbit_eccentricity: обязательно")
	} else if eccentricity = math.Round(*req.OrbitEccentricity*1000) / 1000; eccentricity < 0 || eccentricity >= 1 {
		problems = append(problems, "orbit_eccentricity: число от 0 до 0.999")
	}
	if len(problems) > 0 {
		fail(w, http.StatusBadRequest, strings.Join(problems, "; "))
		return
	}

	comet, err := a.repo.PublishComet(id, session.CurrentUser().ID, description, period, eccentricity)
	if err != nil {
		a.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a.cometResponse(comet))
}

func (a *API) DeleteComet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		fail(w, http.StatusBadRequest, "id должен быть положительным числом")
		return
	}
	if err := a.repo.SoftDeleteComet(id, session.CurrentUser().ID); err != nil {
		a.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DeleteResponse{ID: id, CometStatus: "deleted"})
}

func (a *API) LikeComet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		fail(w, http.StatusBadRequest, "id должен быть положительным числом")
		return
	}
	var req LikeRequest
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Like == nil || (*req.Like != 0 && *req.Like != 1) {
		fail(w, http.StatusBadRequest, "like: 1 — поставить лайк, 0 — отменить")
		return
	}
	count, err := a.repo.SetCometLike(session.CurrentUser().ID, id, *req.Like == 1)
	if err != nil {
		a.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, LikeResponse{CometID: id, Like: *req.Like, LikesCount: count})
}
