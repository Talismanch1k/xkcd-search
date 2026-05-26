package middleware

import (
	"net/http"
	"strings"

	"yadro.com/course/api/internal/httputil"
)

type TokenVerifier interface {
	Verify(token string) error
}

func Auth(next http.HandlerFunc, verifier TokenVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")

		token, ok := strings.CutPrefix(auth, "Token ")
		if !ok {
			httputil.WriteJSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if err := verifier.Verify(token); err != nil {
			httputil.WriteJSONError(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next(w, r)
	}
}
