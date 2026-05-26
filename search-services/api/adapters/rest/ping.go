package rest

import (
	"context"
	"net/http"
	"sync"
	"time"

	"yadro.com/course/api/core"
	"yadro.com/course/api/internal/httputil"
)

type pingResponse struct {
	Replies map[string]string `json:"replies"`
}

func NewPingHandler(pingers map[string]core.Pinger, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := pingResponse{Replies: make(map[string]string, len(pingers))}

		var wg sync.WaitGroup
		var mu sync.Mutex // for resp.Replies

		for n, srv := range pingers {
			// concurrect pinging
			wg.Go(func() {
				ctx, cancel := context.WithTimeout(r.Context(), timeout)
				defer cancel()

				err := srv.Ping(ctx)

				mu.Lock()
				defer mu.Unlock()

				if err != nil {
					resp.Replies[n] = "unavailable"
				} else {
					resp.Replies[n] = "ok"
				}
			})
		}

		wg.Wait()

		httputil.WriteJSONOk(w, resp)
	}
}
