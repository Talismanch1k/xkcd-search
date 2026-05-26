package rest

import (
	"net/http"

	"yadro.com/course/frontend/internal/session"
)

const sessionCookieName = "session_id"

func RequireAuth(store *session.Storage) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := tokenFromRequest(r, store); !ok {
				http.Redirect(w, r, "/admin?error=unauthorized", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func tokenFromRequest(r *http.Request, store *session.Storage) (string, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}
	return store.Get(c.Value)
}
