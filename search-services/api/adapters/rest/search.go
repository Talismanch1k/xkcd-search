package rest

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"yadro.com/course/api/core"
	"yadro.com/course/api/internal/httputil"
)

const defaultLimit int32 = 10

type comicResponse struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type comicInfoResponse struct {
	ID         int      `json:"id"`
	URL        string   `json:"url"`
	Title      []string `json:"title"`
	Alt        []string `json:"alt"`
	Transcript []string `json:"transcript"`
}

type searchResponse struct {
	Comics []comicResponse `json:"comics"`
	Total  int             `json:"total"`
}

func NewSearchHandler(searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("phrase")
		if query == "" {
			httputil.WriteJSONError(w, "phrase is required", http.StatusBadRequest)
			return
		}

		limit := defaultLimit
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {

			parsedLimit, err := strconv.ParseInt(limitStr, 10, 32)
			if err != nil {
				httputil.WriteJSONError(w, "failed convert limit to integer", http.StatusBadRequest)
				return
			}
			if parsedLimit < 1 {
				httputil.WriteJSONError(w, "limit must be positive", http.StatusBadRequest)
				return
			}

			limit = int32(parsedLimit)
		}

		result, err := searcher.Search(r.Context(), query, limit)
		if err != nil {
			slog.Error("search failed", "error", err)
			httputil.WriteJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		respComics := make([]comicResponse, 0, len(result.Comics))
		for _, c := range result.Comics {
			respComics = append(respComics, comicResponse{ID: c.ID, URL: c.URL})
		}

		httputil.WriteJSONOk(w, searchResponse{
			Comics: respComics,
			Total:  result.Total,
		})
	}
}

func NewComicHandler(searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.PathValue("id"))
		if err != nil {
			httputil.WriteJSONError(w, "invalid comic id", http.StatusBadRequest)
			return
		}

		info, err := searcher.GetComic(r.Context(), id)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				httputil.WriteJSONError(w, "comic not found", http.StatusNotFound)
				return
			}
			slog.Error("get comic", "id", id, "err", err)
			httputil.WriteJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		httputil.WriteJSONOk(w, comicInfoResponse{
			ID:         info.ID,
			URL:        info.URL,
			Title:      info.Title,
			Alt:        info.Alt,
			Transcript: info.Transcript,
		})
	}
}

func NewISearchHandler(searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("phrase")
		if query == "" {
			httputil.WriteJSONError(w, "phrase is required", http.StatusBadRequest)
			return
		}

		limit := defaultLimit
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {

			parsedLimit, err := strconv.ParseInt(limitStr, 10, 32)
			if err != nil {
				httputil.WriteJSONError(w, "failed convert limit to integer", http.StatusBadRequest)
				return
			}
			if parsedLimit < 1 {
				httputil.WriteJSONError(w, "limit must be positive", http.StatusBadRequest)
				return
			}

			limit = int32(parsedLimit)
		}

		result, err := searcher.ISearch(r.Context(), query, limit)
		if err != nil {
			slog.Error("search failed", "error", err)
			httputil.WriteJSONError(w, "internal server error", http.StatusInternalServerError)
			return
		}

		respComics := make([]comicResponse, 0, len(result.Comics))
		for _, c := range result.Comics {
			respComics = append(respComics, comicResponse{ID: c.ID, URL: c.URL})
		}

		httputil.WriteJSONOk(w, searchResponse{
			Comics: respComics,
			Total:  result.Total,
		})
	}
}
