package rest

import (
	"encoding/json"
	"html/template"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strconv"

	"yadro.com/course/frontend/core"
)

func NewIndexHandler(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "index.html", nil); err != nil {
			slog.Error("render index", "err", err)
		}
	}
}

func NewDiscoverHandler(tmpl *template.Template, sp core.StatsProvider, m core.MetaFetcher, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		total, err := sp.ComicsTotal(r.Context())
		if err != nil || total == 0 {
			slog.Error("get comics total", "err", err)
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		seen := make(map[int]bool)
		comics := make([]core.Comic, 0, limit)

		for attempts := 0; len(comics) < limit && attempts < limit*3; attempts++ {
			id := rand.IntN(total) + 1
			if seen[id] {
				continue
			}
			seen[id] = true

			meta, err := m.FetchMeta(r.Context(), id)
			if err != nil || meta.URL == "" {
				continue
			}

			comics = append(comics, core.Comic{ID: id, URL: meta.URL})
		}

		if len(comics) == 0 {
			slog.Error("discover: no comics found")
			http.Error(w, "could not find comics", http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "feed.html", struct {
			Phrase string
			Comics []core.Comic
		}{"Discover", comics}); err != nil {
			slog.Error("render discover", "err", err)
		}
	}
}

func NewFeedHandler(tmpl *template.Template, s core.Searcher, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		comics, err := s.Search(r.Context(), phrase, limit)
		if err != nil {
			slog.Error("feed search", "phrase", phrase, "err", err)
			http.Error(w, "search failed", http.StatusInternalServerError)
			return
		}

		data := struct {
			Phrase string
			Comics []core.Comic
		}{phrase, comics}

		if err := tmpl.ExecuteTemplate(w, "feed.html", data); err != nil {
			slog.Error("render feed", "err", err)
		}
	}
}

func NewFeedNextHandler(s core.Searcher, m core.MetaFetcher, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			http.Error(w, "invalid comic id", http.StatusBadRequest)
			return
		}

		meta, err := m.FetchMeta(r.Context(), id)
		if err != nil {
			slog.Error("fetch meta", "id", id, "err", err)
			http.Error(w, "failed to fetch comic meta", http.StatusInternalServerError)
			return
		}

		comics, err := s.Search(r.Context(), core.BuildPhrase(meta), limit)
		if err != nil {
			slog.Error("feed next search", "id", id, "err", err)
			http.Error(w, "search failed", http.StatusInternalServerError)
			return
		}

		// exclude current
		filtered := make([]core.Comic, 0, len(comics))
		for _, c := range comics {
			if c.ID != id {
				filtered = append(filtered, c)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(struct {
			Comics []core.Comic `json:"comics"`
		}{filtered}); err != nil {
			slog.Debug("encode feed next", "err", err)
		}
	}
}
