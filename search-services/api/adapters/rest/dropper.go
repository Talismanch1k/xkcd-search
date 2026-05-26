package rest

import (
	"context"
	"log/slog"
	"net/http"

	"yadro.com/course/api/internal/httputil"
)

type Dropper interface {
	Drop(context.Context) error
}

// NewDropHandler creates an HTTP handler that sequentially resets the state
// of all provided services (cascade drop) implementing the Dropper interface.
func NewDropHandler(droppers ...Dropper) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, d := range droppers {
			if err := d.Drop(r.Context()); err != nil {
				slog.Error("executing drop", "err", err)
				httputil.WriteJSONError(w, "internal error", http.StatusInternalServerError)
				return
			}
		}
	}
}
