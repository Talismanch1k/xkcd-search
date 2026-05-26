package middleware

import (
	"net/http"

	"golang.org/x/time/rate"
	"yadro.com/course/api/internal/httputil"
)

func Rate(next http.HandlerFunc, rps int) http.HandlerFunc {
	burst := max(rps/10, 1)
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(w http.ResponseWriter, r *http.Request) {
		if err := limiter.Wait(r.Context()); err != nil {
			httputil.WriteJSONError(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
