package api

import (
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

var loginPattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,50}$`)

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

func (a *API) LoginUser(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, MessageResponse{Status: "ok", Message: "Заглушка: аутентификация будет реализована в ЛР4"})
}

func (a *API) LogoutUser(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, MessageResponse{Status: "ok", Message: "Заглушка: деавторизация будет реализована в ЛР4"})
}
