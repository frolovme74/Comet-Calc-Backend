package auth

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const issuer = "comet-calc"

var ErrInvalidToken = errors.New("недействительный токен")

type Claims struct {
	jwt.RegisteredClaims
	UserID uint   `json:"user_id"`
	Login  string `json:"login"`
}

type Manager struct {
	secret    []byte
	expiresIn time.Duration
}

func FromEnv() (*Manager, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("не задан JWT_SECRET")
	}
	expiresIn := time.Hour
	if v := os.Getenv("JWT_EXPIRES_IN"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return nil, errors.New("JWT_EXPIRES_IN должен быть длительностью, например 1h")
		}
		expiresIn = d
	}
	return &Manager{secret: []byte(secret), expiresIn: expiresIn}, nil
}

func (m *Manager) ExpiresIn() time.Duration {
	return m.expiresIn
}

func (m *Manager) Issue(userID uint, login string) (string, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expiresIn)),
		},
		UserID: userID,
		Login:  login,
	})
	return token.SignedString(m.secret)
}

func (m *Manager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid || claims.UserID == 0 {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
