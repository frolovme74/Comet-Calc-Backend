package api

import (
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"github.com/frolovme74/Comet-Calc-Backend/internal/repository"
)

var loginPattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,50}$`)

// RegisterUser godoc
// @Summary      Регистрация
// @Description  Создаёт пользователя. Пароль хранится в виде bcrypt-хеша.
// @Tags         Пользователи
// @Accept       json
// @Produce      json
// @Param        user  body      RegisterRequest  true  "Данные пользователя"
// @Success      201   {object}  UserResponse
// @Failure      400   {object}  MessageResponse  "Неверные данные"
// @Failure      409   {object}  MessageResponse  "Логин уже занят"
// @Router       /users/register [post]
func (a *API) RegisterUser(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	req.Login = strings.TrimSpace(req.Login)
	req.FullName = strings.TrimSpace(req.FullName)

	var problems []string
	if !loginPattern.MatchString(req.Login) {
		problems = append(problems, "login: 3–50 символов, латиница, цифры и _")
	}
	if req.FullName == "" || utf8.RuneCountInString(req.FullName) > 100 {
		problems = append(problems, "full_name: обязательно, до 100 символов")
	}
	if utf8.RuneCountInString(req.Password) < 8 || len(req.Password) > 72 {
		problems = append(problems, "password: от 8 символов (не больше 72 байт)")
	}
	if len(problems) > 0 {
		fail(w, http.StatusBadRequest, strings.Join(problems, "; "))
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		a.serverError(w, err)
		return
	}
	user, err := a.repo.CreateUser(req.Login, req.FullName, string(hash))
	if err != nil {
		a.repoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, userResponse(user))
}

// LoginUser godoc
// @Summary      Аутентификация
// @Description  Проверяет логин и пароль и выдаёт JWT. Токен передаётся в заголовке Authorization: Bearer <токен>.
// @Tags         Пользователи
// @Accept       json
// @Produce      json
// @Param        credentials  body      LoginRequest  true  "Логин и пароль"
// @Success      200          {object}  LoginResponse
// @Failure      400          {object}  MessageResponse  "Некорректный JSON"
// @Failure      401          {object}  MessageResponse  "Неверный логин или пароль"
// @Router       /users/login [post]
func (a *API) LoginUser(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		fail(w, http.StatusBadRequest, err.Error())
		return
	}
	user, err := a.repo.UserByLogin(strings.TrimSpace(req.Login))
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		a.serverError(w, err)
		return
	}
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		fail(w, http.StatusUnauthorized, "Неверный логин или пароль")
		return
	}
	token, err := a.auth.Issue(user.ID, user.Login)
	if err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(a.auth.ExpiresIn().Seconds()),
	})
}

// LogoutUser godoc
// @Summary      Деавторизация
// @Description  Заносит текущий JWT в чёрный список Redis до истечения срока его действия.
// @Tags         Пользователи
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  MessageResponse
// @Failure      401  {object}  MessageResponse  "Нет токена или токен недействителен"
// @Router       /users/logout [post]
func (a *API) LogoutUser(w http.ResponseWriter, r *http.Request) {
	user := userOf(r)
	if err := a.blacklist.Add(r.Context(), user.Token, user.Login, time.Until(user.ExpiresAt)); err != nil {
		a.serverError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, MessageResponse{Status: "ok", Message: "Токен отозван"})
}
