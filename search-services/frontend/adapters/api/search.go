package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"yadro.com/course/frontend/core"
	"yadro.com/course/pkg/closer"
)

type comicDTO struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type searchDTO struct {
	Comics []comicDTO `json:"comics"`
}

func (c *Client) Search(ctx context.Context, phrase string, limit int) ([]core.Comic, error) {
	u := c.base + "/api/search?phrase=" + url.QueryEscape(phrase) + "&limit=" + strconv.Itoa(limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer closer.CloseOrIgnore(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search returned %d", resp.StatusCode)
	}

	var dto searchDTO
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}

	comics := make([]core.Comic, 0, len(dto.Comics))
	for _, c := range dto.Comics {
		comics = append(comics, core.Comic{ID: c.ID, URL: c.URL})
	}
	return comics, nil
}

func (c *Client) ComicsTotal(ctx context.Context) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/db/stats", nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return 0, fmt.Errorf("stats: %w", err)
	}
	defer closer.CloseOrIgnore(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("stats returned %d", resp.StatusCode)
	}

	var dto struct {
		ComicsFetched int `json:"comics_fetched"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return 0, fmt.Errorf("decode: %w", err)
	}

	return dto.ComicsFetched, nil
}

func (c *Client) FetchMeta(ctx context.Context, id int) (core.ComicMeta, error) {
	u := fmt.Sprintf("%s/api/comics/%d", c.base, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return core.ComicMeta{}, fmt.Errorf("build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return core.ComicMeta{}, fmt.Errorf("fetch meta: %w", err)
	}
	defer closer.CloseOrIgnore(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return core.ComicMeta{}, fmt.Errorf("api returned %d for comic %d", resp.StatusCode, id)
	}

	var dto struct {
		URL        string   `json:"url"`
		Title      []string `json:"title"`
		Alt        []string `json:"alt"`
		Transcript []string `json:"transcript"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return core.ComicMeta{}, fmt.Errorf("decode: %w", err)
	}

	return core.ComicMeta{
		URL:        dto.URL,
		Title:      strings.Join(dto.Title, " "),
		Alt:        strings.Join(dto.Alt, " "),
		Transcript: strings.Join(dto.Transcript, " "),
	}, nil
}
