package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/frolovme74/Comet-Calc-Backend/internal/repository"
	"github.com/frolovme74/Comet-Calc-Backend/internal/storage"
)

type API struct {
	repo    *repository.Repository
	storage *storage.Storage
}

func New(repo *repository.Repository, storage *storage.Storage) *API {
	return &API{repo: repo, storage: storage}
}

func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/comets", a.GetComets)
	mux.HandleFunc("GET /api/comets/feed", a.GetCometFeed)
	mux.HandleFunc("GET /api/comets/feed/{id...}", a.GetCometFeed)
	mux.HandleFunc("GET /api/comets/draft", a.GetCometDraft)
	mux.HandleFunc("POST /api/comets", a.CreateComet)
	mux.HandleFunc("PUT /api/comets/{id}/publish", a.PublishComet)
	mux.HandleFunc("DELETE /api/comets/{id}", a.DeleteComet)
	mux.HandleFunc("POST /api/comets/{id}/like", a.LikeComet)

	mux.HandleFunc("POST /api/users/register", a.RegisterUser)
	mux.HandleFunc("POST /api/users/login", a.LoginUser)
	mux.HandleFunc("POST /api/users/logout", a.LogoutUser)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Println("json:", err)
	}
}

func fail(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, MessageResponse{Status: "fail", Message: message})
}

func (a *API) serverError(w http.ResponseWriter, err error) {
	log.Println("ошибка API:", err)
	fail(w, http.StatusInternalServerError, "Внутренняя ошибка сервера")
}

func (a *API) repoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		fail(w, http.StatusNotFound, "Комета не найдена")
	case errors.Is(err, repository.ErrForbidden):
		fail(w, http.StatusForbidden, "Действие доступно только создателю кометы")
	case errors.Is(err, repository.ErrWrongStatus):
		fail(w, http.StatusConflict, "Недопустимая смена статуса: опубликовать можно только черновик")
	case errors.Is(err, repository.ErrDraftExists):
		fail(w, http.StatusConflict, "У пользователя уже есть черновик")
	case errors.Is(err, repository.ErrLoginTaken):
		fail(w, http.StatusConflict, "Логин уже занят")
	default:
		a.serverError(w, err)
	}
}

func pathID(r *http.Request) (uint, bool) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 63)
	return uint(id), err == nil && id > 0
}

func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	return v, err == nil && !math.IsNaN(v) && !math.IsInf(v, 0)
}

var systemFields = map[string]bool{
	"id": true, "comet_status": true, "creator_id": true, "created_at": true, "formed_at": true,
	"comet_photo": true, "comet_video": true, "likes_count": true, "is_mine": true, "is_liked": true,
}

func decodeJSON(r *http.Request, dst any) error {
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return errors.New("не удалось прочитать тело запроса")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return errors.New("тело запроса должно быть JSON-объектом")
	}
	for key := range raw {
		if systemFields[key] {
			return fmt.Errorf("поле %s системное, его нельзя передавать с клиента", key)
		}
	}
	dec := json.NewDecoder(strings.NewReader(string(body)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("некорректный JSON: %v", strings.TrimPrefix(err.Error(), "json: "))
	}
	return nil
}
