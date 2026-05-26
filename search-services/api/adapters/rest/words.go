package rest

import (
	"errors"
	"log/slog"
	"net/http"

	"yadro.com/course/api/core"
	"yadro.com/course/api/internal/httputil"
)

type wordsResponse struct {
	Words []string `json:"words"`
	Total int      `json:"total"`
}

func NewNormHandler(normalizer core.Normalizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			httputil.WriteJSONError(w, "phrase is required", http.StatusBadRequest)
			return
		}

		norm, err := normalizer.Norm(r.Context(), phrase)
		if err != nil {
			if errors.Is(err, core.ErrBadArguments) {
				httputil.WriteJSONError(w, err.Error(), http.StatusBadRequest)
				return
			}
			slog.Error("normalize", "err", err)
			httputil.WriteJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		httputil.WriteJSONOk(w, wordsResponse{
			Words: norm,
			Total: len(norm),
		})
	}
}
