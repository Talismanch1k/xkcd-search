package middleware

import (
	"log/slog"
	"net/http"

	"yadro.com/course/api/internal/httputil"
)

func Concurrency(next http.HandlerFunc, limit int) http.HandlerFunc {
	sem := make(chan struct{}, limit)

	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case sem <- struct{}{}:
			defer func() { <-sem }()
			next(w, r)

		default:
			httputil.WriteJSONError(w, "service unavailable", http.StatusServiceUnavailable)
			slog.Warn("concurrency limit reached", "path", r.URL.Path, "limit", cap(sem))
		}
	}
}
