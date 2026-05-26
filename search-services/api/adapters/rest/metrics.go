package rest

import (
	"net/http"

	"github.com/VictoriaMetrics/metrics"
)

func NewMetricsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, true)
	}
}
