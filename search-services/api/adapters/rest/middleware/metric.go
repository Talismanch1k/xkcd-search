package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/VictoriaMetrics/metrics"
)

type responseWriter struct {
	http.ResponseWriter
	code int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.code = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func WithMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, code: http.StatusOK}

		next.ServeHTTP(rw, r)

		name := fmt.Sprintf(
			`http_request_duration_seconds{url="%s", method="%s", status="%s"}`,
			r.URL.Path,
			r.Method,
			strconv.Itoa(rw.code),
		)

		metrics.GetOrCreateHistogram(name).UpdateDuration(start)
	})
}
