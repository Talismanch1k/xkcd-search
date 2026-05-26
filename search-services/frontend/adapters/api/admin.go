package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"yadro.com/course/frontend/core"
	"yadro.com/course/pkg/closer"
)

func (c *Client) Login(ctx context.Context, user, pass string) (string, error) {
	body, err := json.Marshal(map[string]string{"name": user, "password": pass})
	if err != nil {
		return "", fmt.Errorf("marshal credentials: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/login", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}
	defer closer.CloseOrIgnore(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with %d", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	return result.Token, nil
}

func (c *Client) Stats(ctx context.Context, token string) (core.Stats, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/db/stats", nil)
	if err != nil {
		return core.Stats{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return core.Stats{}, fmt.Errorf("stats: %w", err)
	}
	defer closer.CloseOrIgnore(resp.Body)

	var dto struct {
		WordsTotal    int `json:"words_total"`
		WordsUnique   int `json:"words_unique"`
		ComicsFetched int `json:"comics_fetched"`
		ComicsTotal   int `json:"comics_total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dto); err != nil {
		return core.Stats{}, fmt.Errorf("decode: %w", err)
	}

	return core.Stats(dto), nil
}

func (c *Client) Status(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/api/db/status", nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("status: %w", err)
	}
	defer closer.CloseOrIgnore(resp.Body)

	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}

	return result.Status, nil
}

func (c *Client) Update(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/db/update", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	defer closer.CloseOrIgnore(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("update returned %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) Drop(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.base+"/api/db", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Token "+token)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("drop: %w", err)
	}
	defer closer.CloseOrIgnore(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("drop returned %d", resp.StatusCode)
	}

	return nil
}
