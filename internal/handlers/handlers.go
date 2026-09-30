package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/frolovme74/Comet-Calc-Backend/internal/repository"
)

const CurrentUserID uint = 1

const (
	orbitalPeriodMin = 0
	orbitalPeriodMax = 140
)

type CometHandler struct {
	tmpl *template.Template
	repo *repository.Repository
}

func NewCometHandler(tmpl *template.Template, repo *repository.Repository) *CometHandler {
	return &CometHandler{tmpl: tmpl, repo: repo}
}

func (h *CometHandler) render(w http.ResponseWriter, status int, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Println("template:", err)
	}
}

func (h *CometHandler) serverError(w http.ResponseWriter, err error) {
	log.Println("ошибка:", err)
	http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
}

func parseNumber(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
	return v, err == nil
}

func (h *CometHandler) CometFeed(w http.ResponseWriter, r *http.Request) {
	idStr := strings.Trim(r.PathValue("id"), "/")

	var (
		card repository.CometCard
		err  error
	)
	if idStr == "" {
		card, err = h.repo.FirstPublishedComet()
	} else {
		id, convErr := strconv.ParseUint(idStr, 10, 63)
		if convErr != nil {
			h.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("next") == "true" {
			card, err = h.repo.NextPublishedComet(uint(id))
		} else {
			card, err = h.repo.PublishedCometByID(uint(id))
		}
	}
	if errors.Is(err, repository.ErrNotFound) {
		h.NotFound(w, r)
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}

	h.render(w, http.StatusOK, "comet_feed.html", map[string]any{
		"Tab":   "feed",
		"Comet": card,
	})
}

type draftForm struct {
	CometName         string
	CometDescription  string
	OrbitalPeriod     string
	OrbitEccentricity string
	Errors            []string
}

func (h *CometHandler) renderDraft(w http.ResponseWriter, status int, form draftForm) {
	data := map[string]any{"Tab": "add", "Form": form}
	comet, err := h.repo.DraftComet(CurrentUserID)
	switch {
	case err == nil:
		data["Comet"] = comet
	case !errors.Is(err, repository.ErrNotFound):
		h.serverError(w, err)
		return
	}
	h.render(w, status, "comet_draft.html", data)
}

func (h *CometHandler) CometDraft(w http.ResponseWriter, r *http.Request) {
	h.renderDraft(w, http.StatusOK, draftForm{})
}

func (h *CometHandler) CreateCometDraft(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("comet_name"))
	if !utf8.ValidString(name) || name == "" || utf8.RuneCountInString(name) > 100 {
		if !utf8.ValidString(name) {
			name = ""
		}
		h.renderDraft(w, http.StatusUnprocessableEntity, draftForm{
			CometName: name,
			Errors:    []string{"Укажите название кометы (до 100 символов)"},
		})
		return
	}

	if _, err := h.repo.DraftComet(CurrentUserID); err == nil {
		http.Redirect(w, r, "/comets/draft", http.StatusSeeOther)
		return
	}
	if _, err := h.repo.CreateDraftComet(CurrentUserID, name); err != nil && !errors.Is(err, repository.ErrDraftExists) {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/comets/draft", http.StatusSeeOther)
}

func (h *CometHandler) PublishCometDraft(w http.ResponseWriter, r *http.Request) {
	form := draftForm{
		CometDescription:  strings.TrimSpace(r.FormValue("comet_description")),
		OrbitalPeriod:     strings.TrimSpace(r.FormValue("orbital_period")),
		OrbitEccentricity: strings.TrimSpace(r.FormValue("orbit_eccentricity")),
	}
	period, okPeriod := parseNumber(form.OrbitalPeriod)
	eccentricity, okEccentricity := parseNumber(form.OrbitEccentricity)
	period = math.Round(period*100) / 100
	eccentricity = math.Round(eccentricity*1000) / 1000

	if !utf8.ValidString(form.CometDescription) {
		form.CometDescription = ""
		form.Errors = append(form.Errors, "Описание должно быть в кодировке UTF-8")
	} else if form.CometDescription == "" || utf8.RuneCountInString(form.CometDescription) > 500 {
		form.Errors = append(form.Errors, "Заполните краткое описание (до 500 символов)")
	}
	if !okPeriod || period <= 0 || period > 999999.99 {
		form.Errors = append(form.Errors, "Период обращения — положительное число лет, не больше 999 999,99")
	}
	if !okEccentricity || eccentricity < 0 || eccentricity >= 1 {
		form.Errors = append(form.Errors, "Эксцентриситет — число от 0 до 0,999")
	}
	if len(form.Errors) > 0 {
		h.renderDraft(w, http.StatusUnprocessableEntity, form)
		return
	}

	comet, err := h.repo.PublishDraftComet(CurrentUserID, form.CometDescription, period, eccentricity)
	if errors.Is(err, repository.ErrNotFound) {
		http.Redirect(w, r, "/comets/draft", http.StatusSeeOther)
		return
	}
	if err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, fmt.Sprintf("/comets/feed/%d", comet.ID), http.StatusSeeOther)
}

func (h *CometHandler) CometList(w http.ResponseWriter, r *http.Request) {
	from, hasFrom := parseNumber(r.URL.Query().Get("min_orbital_period"))
	to, hasTo := parseNumber(r.URL.Query().Get("max_orbital_period"))
	hasFrom = hasFrom && from > orbitalPeriodMin
	hasTo = hasTo && to < orbitalPeriodMax
	if hasFrom && hasTo && from > to {
		from, to = to, from
	}

	var periodFrom, periodTo *float64
	if hasFrom {
		periodFrom = &from
	}
	if hasTo {
		periodTo = &to
	}
	cards, err := h.repo.PublishedComets(periodFrom, periodTo)
	if err != nil {
		h.serverError(w, err)
		return
	}

	sliderFrom, sliderTo := float64(orbitalPeriodMin), float64(orbitalPeriodMax)
	if hasFrom {
		sliderFrom = from
	}
	if hasTo {
		sliderTo = to
	}

	h.render(w, http.StatusOK, "comet_list.html", map[string]any{
		"Tab":           "tiles",
		"Comets":        cards,
		"PeriodFrom":    from,
		"PeriodTo":      to,
		"HasPeriodFrom": hasFrom,
		"HasPeriodTo":   hasTo,
		"PeriodMin":     orbitalPeriodMin,
		"PeriodMax":     orbitalPeriodMax,
		"SliderFrom":    sliderFrom,
		"SliderTo":      sliderTo,
	})
}

func (h *CometHandler) DeleteComet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(r.PathValue("id"), 10, 63)
	if err != nil {
		h.NotFound(w, r)
		return
	}
	err = h.repo.DeleteComet(r.Context(), uint(id))
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		h.serverError(w, err)
		return
	}
	back := url.Values{}
	for _, key := range []string{"min_orbital_period", "max_orbital_period"} {
		if v, ok := parseNumber(r.FormValue(key)); ok {
			back.Set(key, strconv.FormatFloat(v, 'f', -1, 64))
		}
	}
	target := "/comets"
	if len(back) > 0 {
		target += "?" + back.Encode()
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func (h *CometHandler) NotFound(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusNotFound, "error.html", map[string]any{
		"Tab":   "",
		"Title": "Страница не найдена",
		"Text":  "Такой кометы нет или она удалена.",
	})
}

type russianErrors struct {
	http.ResponseWriter
	h        *CometHandler
	replaced bool
}

func (rw *russianErrors) WriteHeader(code int) {
	plain := strings.HasPrefix(rw.Header().Get("Content-Type"), "text/plain")
	if plain && (code == http.StatusNotFound || code == http.StatusMethodNotAllowed) {
		rw.replaced = true
		rw.Header().Del("Content-Length")
		rw.Header().Del("X-Content-Type-Options")
		title, text := "Страница не найдена", "Такой страницы нет."
		if code == http.StatusMethodNotAllowed {
			title, text = "Метод не поддерживается", "Этот адрес не принимает такой запрос."
		}
		rw.h.render(rw.ResponseWriter, code, "error.html", map[string]any{"Tab": "", "Title": title, "Text": text})
		return
	}
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *russianErrors) Write(b []byte) (int, error) {
	if rw.replaced {
		return len(b), nil
	}
	return rw.ResponseWriter.Write(b)
}

func (h *CometHandler) RussianErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(&russianErrors{ResponseWriter: w, h: h}, r)
	})
}
