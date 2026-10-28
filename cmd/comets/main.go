package main

import (
	"context"
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/frolovme74/Comet-Calc-Backend/docs"
	"github.com/frolovme74/Comet-Calc-Backend/internal/api"
	"github.com/frolovme74/Comet-Calc-Backend/internal/auth"
	"github.com/frolovme74/Comet-Calc-Backend/internal/blacklist"
	"github.com/frolovme74/Comet-Calc-Backend/internal/dsn"
	"github.com/frolovme74/Comet-Calc-Backend/internal/handlers"
	"github.com/frolovme74/Comet-Calc-Backend/internal/media"
	"github.com/frolovme74/Comet-Calc-Backend/internal/repository"
	"github.com/frolovme74/Comet-Calc-Backend/internal/storage"
)

// @title        Comet-Calc API
// @version      1.0
// @description  Веб-сервис «Расстояние кометы от Солнца»: кометы (услуги), лайки, пользователи. Чтение доступно без авторизации, методы создателя — с JWT.

// @schemes   http
// @BasePath  /api

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Введите: Bearer <access_token из ответа /users/login>

func main() {
	_ = godotenv.Load()

	repo, err := repository.New(dsn.FromEnv())
	if err != nil {
		log.Fatalf("подключение к БД: %v", err)
	}

	store, err := storage.FromEnv()
	if err != nil {
		log.Fatalf("подключение к Minio: %v", err)
	}

	jwtManager, err := auth.FromEnv()
	if err != nil {
		log.Fatalf("настройки JWT: %v", err)
	}

	tokenBlacklist, err := blacklist.FromEnv(context.Background())
	if err != nil {
		log.Fatalf("подключение к Redis: %v", err)
	}

	checker := media.NewChecker(store.URL)
	funcs := template.FuncMap{
		"photo": checker.Photo,
		"video": checker.Video,
		"num": func(v any) string {
			f, ok := number(v)
			if !ok {
				return ""
			}
			return strings.ReplaceAll(strconv.FormatFloat(f, 'f', -1, 64), ".", ",")
		},
		"inputNum": func(v any) string {
			f, ok := number(v)
			if !ok {
				return ""
			}
			return strconv.FormatFloat(f, 'f', -1, 64)
		},
		"years": func(v any) string {
			f, _ := number(v)
			return years(f)
		},
		"defaultPhoto": func() string { return media.DefaultCometPhoto },
		"defaultVideo": func() string { return media.DefaultCometVideo },
	}
	tmpl := template.Must(template.New("").Funcs(funcs).ParseGlob("templates/*.html"))

	h := handlers.NewCometHandler(tmpl, repo)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /comets/feed", h.CometFeed)
	mux.HandleFunc("GET /comets/feed/{id...}", h.CometFeed)
	mux.HandleFunc("GET /comets/draft", h.CometDraft)
	mux.HandleFunc("GET /comets", h.CometList)
	mux.HandleFunc("POST /comets/draft", h.CreateCometDraft)
	mux.HandleFunc("POST /comets/draft/publish", h.PublishCometDraft)
	mux.HandleFunc("POST /comets/{id}/delete", h.DeleteComet)

	api.New(repo, store, jwtManager, tokenBlacklist).Register(mux)
	mux.Handle("GET /swagger/", httpSwagger.Handler(httpSwagger.URL("/swagger/doc.json")))

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	addr := getenv("ADDR", ":8080")
	log.Printf("Сервер: http://localhost%s/comets, API: http://localhost%s/api/comets, Swagger: http://localhost%s/swagger/", addr, addr, addr)
	log.Fatal(http.ListenAndServe(addr, h.RussianErrors(mux)))
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case *float64:
		if n == nil {
			return 0, false
		}
		return *n, true
	case int:
		return float64(n), true
	}
	return 0, false
}

func years(v float64) string {
	if v != math.Trunc(v) {
		return "года"
	}
	n := int(v)
	switch {
	case n%100 >= 11 && n%100 <= 14:
		return "лет"
	case n%10 == 1:
		return "год"
	case n%10 >= 2 && n%10 <= 4:
		return "года"
	}
	return "лет"
}
