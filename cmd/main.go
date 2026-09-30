package main

import (
	"html/template"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/frolovme74/Comet-Calc-Backend/internal/handlers"
)

func main() {
	minioURL := strings.TrimRight(getenv("MINIO_URL", "http://localhost:9000/comets"), "/")

	funcs := template.FuncMap{
		"minio": func(key string) string { return minioURL + "/" + key },
		"num": func(v float64) string {
			return strings.ReplaceAll(strconv.FormatFloat(v, 'f', -1, 64), ".", ",")
		},
		"inputNum": func(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) },
		"years":    years,
	}
	tmpl := template.Must(template.New("").Funcs(funcs).ParseGlob("templates/*.html"))

	h := handlers.NewCometHandler(tmpl)
	mux := http.NewServeMux()

	mux.HandleFunc("GET /comets/feed/{id...}", h.CometFeed)
	mux.HandleFunc("GET /comets/draft", h.CometDraft)
	mux.HandleFunc("GET /comets", h.CometList)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	addr := getenv("ADDR", ":8080")
	log.Printf("Сервер: http://localhost%s/comets  (Minio: %s)", addr, minioURL)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
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
