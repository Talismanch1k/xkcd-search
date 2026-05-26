package rest

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"yadro.com/course/api/internal/httputil"
)

type Authenticator interface {
	Login(name, password string) (string, error)
}

type credentials struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func NewLoginHandler(auth Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cr credentials
		if err := json.NewDecoder(r.Body).Decode(&cr); err != nil {
			httputil.WriteJSONError(w, "bad request", http.StatusBadRequest)
			return
		}

		token, err := auth.Login(cr.Name, cr.Password)
		if err != nil {
			httputil.WriteJSONError(w, "unauthorized", http.StatusUnauthorized)
			slog.Warn("login failed", "name", cr.Name, "remote", r.RemoteAddr)
			return
		}

		httputil.WriteJSONOk(w, map[string]string{
			"token": token,
		})
	}
}
