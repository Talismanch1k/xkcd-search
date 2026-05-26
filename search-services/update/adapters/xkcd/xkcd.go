package xkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"yadro.com/course/pkg/closer"
	"yadro.com/course/update/core"
)

const infoSuffix = "/info.0.json"

type Client struct {
	client http.Client
	url    string
}

type xkcdResponse struct {
	Num        int    `json:"num"`
	Title      string `json:"safe_title"`
	Transcript string `json:"transcript"`
	Alt        string `json:"alt"`
	Img        string `json:"img"`
}

func NewClient(url string, timeout time.Duration) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	return &Client{
		client: http.Client{Timeout: timeout},
		url:    url,
	}, nil
}

func (c Client) fetch(ctx context.Context, url string) (xkcdResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return xkcdResponse{}, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return xkcdResponse{}, fmt.Errorf("execute request: %w", err)
	}
	defer closer.CloseOrIgnore(resp.Body) // err not important

	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusNotFound:
		return xkcdResponse{}, core.ErrNotFound

	case http.StatusTooManyRequests:
		return xkcdResponse{}, core.ErrRateLimit

	default:
		return xkcdResponse{}, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var result xkcdResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return xkcdResponse{}, fmt.Errorf("decode response body: %w", err)
	}

	return result, nil
}

func (c Client) Get(ctx context.Context, id int) (core.XKCDInfo, error) {
	resp, err := c.fetch(ctx,
		fmt.Sprintf("%s/%d%s", c.url, id, infoSuffix),
	)
	if err != nil {
		return core.XKCDInfo{}, fmt.Errorf("get comic %d: %w", id, err)
	}

	return core.XKCDInfo{
		ID:         resp.Num,
		URL:        resp.Img,
		Title:      resp.Title,
		Alt:        resp.Alt,
		Transcript: resp.Transcript,
	}, nil
}

func (c Client) LastID(ctx context.Context) (int, error) {
	resp, err := c.fetch(ctx, c.url+infoSuffix)
	if err != nil {
		return 0, fmt.Errorf("fetch last comic id: %w", err)
	}

	return resp.Num, nil
}
