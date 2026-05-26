package rest

import (
	"html/template"
	"log/slog"
	"net/http"

	"yadro.com/course/frontend/core"
	"yadro.com/course/frontend/internal/session"
)

func NewAdminHandler(tmpl *template.Template, store *session.Storage, admin core.AdminClient) http.HandlerFunc {
	type pageData struct {
		LoggedIn bool
		Error    string
		Stats    core.Stats
		Status   string
	}
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := tokenFromRequest(r, store)
		if !ok {
			if err := tmpl.ExecuteTemplate(w, "admin.html", pageData{Error: r.URL.Query().Get("error")}); err != nil {
				slog.Error("render admin login", "err", err)
			}
			return
		}
		stats, err := admin.Stats(r.Context(), token)
		if err != nil {
			slog.Error("get stats", "err", err)
		}

		status, err := admin.Status(r.Context(), token)
		if err != nil {
			slog.Error("get status", "err", err)
		}

		if err := tmpl.ExecuteTemplate(w, "admin.html", pageData{
			LoggedIn: true,
			Stats:    stats,
			Status:   status,
		}); err != nil {
			slog.Error("render admin", "err", err)
		}
	}
}

func NewAdminLoginHandler(store *session.Storage, admin core.AdminClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/admin?error=bad+request", http.StatusSeeOther)
			return
		}

		token, err := admin.Login(r.Context(), r.FormValue("username"), r.FormValue("password"))
		if err != nil {
			slog.Warn("admin login failed", "remote", r.RemoteAddr)
			http.Redirect(w, r, "/admin?error=invalid+credentials", http.StatusSeeOther)
			return
		}

		id, err := store.Create(token)
		if err != nil {
			slog.Error("create session", "err", err)
			http.Redirect(w, r, "/admin?error=internal+error", http.StatusSeeOther)
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    id,
			Path:     "/admin",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func NewAdminLogoutHandler(store *session.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookieName); err == nil {
			store.Delete(c.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:   sessionCookieName,
			Value:  "",
			Path:   "/admin",
			MaxAge: -1,
		})
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func NewAdminUpdateHandler(store *session.Storage, admin core.AdminClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := tokenFromRequest(r, store)
		if err := admin.Update(r.Context(), token); err != nil {
			slog.Error("admin update", "err", err)
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}

func NewAdminDropHandler(store *session.Storage, admin core.AdminClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, _ := tokenFromRequest(r, store)
		if err := admin.Drop(r.Context(), token); err != nil {
			slog.Error("admin drop", "err", err)
		}
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	}
}
