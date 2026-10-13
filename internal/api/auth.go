package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/frolovme74/Comet-Calc-Backend/internal/auth"
)

const bearerPrefix = "Bearer "

type userKey struct{}

type currentUser struct {
	ID        uint
	Login     string
	Token     string
	ExpiresAt time.Time
}

func (a *API) userFromRequest(r *http.Request) (*currentUser, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, bearerPrefix) {
		return nil, auth.ErrInvalidToken
	}
	token := strings.TrimSpace(header[len(bearerPrefix):])
	claims, err := a.auth.Parse(token)
	if err != nil {
		return nil, err
	}
	revoked, err := a.blacklist.Contains(r.Context(), token)
	if err != nil {
		return nil, err
	}
	if revoked {
		return nil, auth.ErrInvalidToken
	}
	return &currentUser{ID: claims.UserID, Login: claims.Login, Token: token, ExpiresAt: claims.ExpiresAt.Time}, nil
}

func (a *API) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := a.userFromRequest(r)
		if err == auth.ErrInvalidToken {
			fail(w, http.StatusUnauthorized, "Требуется авторизация: заголовок Authorization: Bearer и токен из ответа /api/users/login")
			return
		}
		if err != nil {
			a.serverError(w, err)
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), userKey{}, user)))
	}
}

func (a *API) withOptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if user, err := a.userFromRequest(r); err == nil {
			r = r.WithContext(context.WithValue(r.Context(), userKey{}, user))
		}
		next(w, r)
	}
}

func userOf(r *http.Request) *currentUser {
	if user, ok := r.Context().Value(userKey{}).(*currentUser); ok {
		return user
	}
	return &currentUser{}
}
