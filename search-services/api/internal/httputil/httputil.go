package httputil

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	js, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to encode json response", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		if _, err := w.Write([]byte(`{"error":"failed to encode json response"}`)); err != nil {
			slog.Debug("failed to write error response", "err", err)
		}
		return
	}

	w.WriteHeader(code)
	if _, err := w.Write(js); err != nil {
		slog.Debug("failed to write response", "err", err)
	}
}

func WriteJSONOk(w http.ResponseWriter, data any) {
	WriteJSON(w, http.StatusOK, data)
}

// Standardizing the style of server responses (errors)
func WriteJSONError(w http.ResponseWriter, msg string, code int) {
	WriteJSON(w, code, map[string]string{
		"error": msg,
	})
}
