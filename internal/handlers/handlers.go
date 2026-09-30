package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/frolovme74/Comet-Calc-Backend/internal/models"
)

type CometHandler struct {
	tmpl *template.Template
}

func NewCometHandler(tmpl *template.Template) *CometHandler {
	return &CometHandler{tmpl: tmpl}
}

type cometView struct {
	models.Comet
	LikesCount int
}

func view(c models.Comet) cometView {
	return cometView{Comet: c, LikesCount: len(c.Likes)}
}

func (h *CometHandler) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Println("template:", err)
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (h *CometHandler) CometFeed(w http.ResponseWriter, r *http.Request) {
	idStr := strings.Trim(r.PathValue("id"), "/")

	var (
		comet models.Comet
		ok    bool
	)
	if idStr == "" {
		list := models.PublishedComets()
		if len(list) > 0 {
			comet, ok = list[0], true
		}
	} else {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("next") == "true" {
			comet, ok = models.NextPublishedComet(id)
		} else {
			comet, ok = models.PublishedCometByID(id)
		}
	}
	if !ok {
		http.NotFound(w, r)
		return
	}

	h.render(w, "comet_feed.html", map[string]any{
		"Tab":   "feed",
		"Comet": view(comet),
	})
}

func (h *CometHandler) CometDraft(w http.ResponseWriter, r *http.Request) {
	comet, ok := models.DraftComet()
	if !ok {
		http.NotFound(w, r)
		return
	}
	h.render(w, "comet_draft.html", map[string]any{
		"Tab":   "add",
		"Comet": view(comet),
	})
}

const (
	orbitalPeriodMin = 0
	orbitalPeriodMax = 140
)

func parseOrbitalPeriod(r *http.Request, name string) (float64, bool) {
	query := strings.TrimSpace(r.URL.Query().Get(name))
	if query == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.ReplaceAll(query, ",", "."), 64)
	return v, err == nil
}

func (h *CometHandler) CometList(w http.ResponseWriter, r *http.Request) {
	from, hasFrom := parseOrbitalPeriod(r, "min_orbital_period")
	to, hasTo := parseOrbitalPeriod(r, "max_orbital_period")
	hasFrom = hasFrom && from > orbitalPeriodMin
	hasTo = hasTo && to < orbitalPeriodMax
	if hasFrom && hasTo && from > to {
		from, to = to, from
	}

	items := make([]cometView, 0)
	for _, c := range models.PublishedComets() {
		if hasFrom && c.OrbitalPeriod < from || hasTo && c.OrbitalPeriod > to {
			continue
		}
		items = append(items, view(c))
	}

	sliderFrom, sliderTo := float64(orbitalPeriodMin), float64(orbitalPeriodMax)
	if hasFrom {
		sliderFrom = from
	}
	if hasTo {
		sliderTo = to
	}

	h.render(w, "comet_list.html", map[string]any{
		"Tab":           "tiles",
		"Comets":        items,
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
