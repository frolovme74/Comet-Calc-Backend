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

// GetComets godoc
// @Summary      Список комет
// @Description  Опубликованные кометы с фильтром по периоду обращения. Доступно без авторизации. is_mine = 1, если создатель кометы — текущий пользователь.
// @Tags         Кометы
// @Produce      json
// @Param        min_orbital_period  query     number  false  "Период обращения от, лет"
// @Param        max_orbital_period  query     number  false  "Период обращения до, лет"
// @Success      200                 {array}   CometListResponse
// @Failure      400                 {object}  MessageResponse  "Период не число"
// @Router       /comets [get]
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

	cards, err := a.repo.PublishedComets(userOf(r).ID, from, to)
	if err != nil {
		a.serverError(w, err)
		return
	}
	resp := make([]CometListResponse, 0, len(cards))
	for _, c := range cards {
		resp = append(resp, a.cometListResponse(c))
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetCometFeed godoc
// @Summary      Лента
// @Description  Одна опубликованная комета: без id — первая, с id — по id, с next=true — следующая после id. Доступно без авторизации. is_liked = 1, если текущий пользователь лайкнул комету.
// @Tags         Кометы
// @Produce      json
// @Param        id    path      int   false  "id кометы; пусто — первая комета ленты"  allowEmptyValue(true)
// @Param        next  query     bool  false  "true — следующая комета после id"
// @Success      200   {object}  CometFeedResponse
// @Failure      404   {object}  MessageResponse  "Комета не найдена"
// @Router       /comets/feed/{id} [get]
func (a *API) GetCometFeed(w http.ResponseWriter, r *http.Request) {
	userID := userOf(r).ID
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
	writeJSON(w, http.StatusOK, a.cometFeedResponse(card))
}

// GetCometDraft godoc
// @Summary      Черновик
// @Description  Черновик текущего пользователя, id не указывается.
// @Tags         Кометы
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  CometResponse
// @Failure      401  {object}  MessageResponse  "Требуется авторизация"
// @Failure      404  {object}  MessageResponse  "Черновика нет"
// @Router       /comets/draft [get]
func (a *API) GetCometDraft(w http.ResponseWriter, r *http.Request) {
	comet, err := a.repo.DraftComet(userOf(r).ID)
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

// CreateComet godoc
// @Summary      Добавление кометы
// @Description  Создаёт черновик, создатель — текущий пользователь из токена. Файлы загружаются в Minio, в БД сохраняются их имена.
// @Tags         Кометы
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        comet_name   formData  string  true  "Название кометы"
// @Param        comet_photo  formData  file    true  "Изображение: jpeg, png, webp, gif до 10 МБ"
// @Param        comet_video  formData  file    true  "Видео: mp4, webm до 50 МБ"
// @Success      201          {object}  CometResponse
// @Failure      400          {object}  MessageResponse  "Неверные данные"
// @Failure      401          {object}  MessageResponse  "Требуется авторизация"
// @Failure      409          {object}  MessageResponse  "Черновик уже есть"
// @Router       /comets [post]
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

	userID := userOf(r).ID
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

// PublishComet godoc
// @Summary      Публикация кометы
// @Description  Переводит свой черновик в статус «опубликован», заполняет описание и поля по теме.
// @Tags         Кометы
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id     path      int                  true  "id кометы"
// @Param        comet  body      PublishCometRequest  true  "Описание, период обращения и эксцентриситет"
// @Success      200    {object}  CometResponse
// @Failure      400    {object}  MessageResponse  "Неверные данные"
// @Failure      401    {object}  MessageResponse  "Требуется авторизация"
// @Failure      403    {object}  MessageResponse  "Комета другого пользователя"
// @Failure      404    {object}  MessageResponse  "Комета не найдена"
// @Failure      409    {object}  MessageResponse  "Комета не в статусе черновика"
// @Router       /comets/{id}/publish [put]
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

	comet, err := a.repo.PublishComet(id, userOf(r).ID, description, period, eccentricity)
	if err != nil {
		a.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, a.cometResponse(comet))
}

// DeleteComet godoc
// @Summary      Удаление кометы
// @Description  Логическое удаление: статус меняется на «удалён». Только своя комета.
// @Tags         Кометы
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "id кометы"
// @Success      200  {object}  DeleteResponse
// @Failure      401  {object}  MessageResponse  "Требуется авторизация"
// @Failure      403  {object}  MessageResponse  "Комета другого пользователя"
// @Failure      404  {object}  MessageResponse  "Комета не найдена"
// @Router       /comets/{id} [delete]
func (a *API) DeleteComet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		fail(w, http.StatusBadRequest, "id должен быть положительным числом")
		return
	}
	if err := a.repo.SoftDeleteComet(id, userOf(r).ID); err != nil {
		a.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, DeleteResponse{ID: id, CometStatus: "deleted"})
}

// LikeComet godoc
// @Summary      Лайк
// @Description  Лайк от текущего пользователя: 1 — поставить, 0 — отменить.
// @Tags         Кометы
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      int          true  "id кометы"
// @Param        like  body      LikeRequest  true  "0 или 1"
// @Success      200   {object}  LikeResponse
// @Failure      400   {object}  MessageResponse  "like не 0 и не 1"
// @Failure      401   {object}  MessageResponse  "Требуется авторизация"
// @Failure      404   {object}  MessageResponse  "Комета не найдена"
// @Router       /comets/{id}/like [post]
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
	count, err := a.repo.SetCometLike(userOf(r).ID, id, *req.Like == 1)
	if err != nil {
		a.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, LikeResponse{CometID: id, Like: *req.Like, LikesCount: count})
}
