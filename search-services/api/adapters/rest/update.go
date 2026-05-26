package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"yadro.com/course/api/core"
	"yadro.com/course/api/internal/httputil"
)

type statsResponse struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

type statusResponse struct {
	Status string `json:"status"`
}

func NewStatsHandler(updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		stats, err := updater.Stats(r.Context())
		if err != nil {
			slog.Error("get stats", "err", err)
			httputil.WriteJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		httputil.WriteJSONOk(w, statsResponse{
			WordsTotal:    stats.WordsTotal,
			WordsUnique:   stats.WordsUnique,
			ComicsFetched: stats.ComicsFetched,
			ComicsTotal:   stats.ComicsTotal,
		})
	}
}

func NewUpdateHandler(updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := updater.Update(r.Context())
		if err != nil {
			if errors.Is(err, core.ErrAlreadyExists) {
				w.WriteHeader(http.StatusAccepted)
				return
			}

			slog.Error("update", "err", err)
			httputil.WriteJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
		
		w.WriteHeader(http.StatusAccepted)
	}
}

func NewStatusHandler(updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		st, err := updater.Status(r.Context())
		if err != nil {
			slog.Error("get status", "err", err)
			httputil.WriteJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}
		httputil.WriteJSONOk(w, statusResponse{Status: string(st)})
	}
}
