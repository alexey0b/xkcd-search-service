package xkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"search-service/update/core"
	"time"
)

const xkcdInfoEndpoint = "info.0.json"

type Client struct {
	log    *slog.Logger
	client http.Client
	url    string
}

func NewClient(url string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	return &Client{
		client: http.Client{Timeout: timeout},
		log:    log,
		url:    url,
	}, nil
}

func (c *Client) Get(ctx context.Context, id int64) (core.XKCDInfo, error) {
	url, err := url.JoinPath(c.url, fmt.Sprint(id), xkcdInfoEndpoint)
	if err != nil {
		return core.XKCDInfo{}, fmt.Errorf("cannot join url path: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return core.XKCDInfo{}, fmt.Errorf("cannot create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return core.XKCDInfo{}, fmt.Errorf("cannot get response for comic %d: %w", id, err)
	}
	defer c.closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return core.XKCDInfo{}, core.ErrNotFound
		} else {
			return core.XKCDInfo{}, fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}
	}

	var info core.XKCDInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return core.XKCDInfo{}, fmt.Errorf("cannot decode reply: %w", err)
	}
	return info, nil
}

func (c *Client) LastID(ctx context.Context) (int64, error) {
	url, err := url.JoinPath(c.url, xkcdInfoEndpoint)
	if err != nil {
		return 0, fmt.Errorf("cannot join url path: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, fmt.Errorf("cannot create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("cannot get response: %w", err)
	}
	defer c.closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return 0, core.ErrNotFound
		} else {
			return 0, fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}
	}

	var info core.XKCDInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return 0, fmt.Errorf("cannot decode reply: %w", err)
	}
	return info.ID, nil
}

func (c *Client) closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		c.log.Warn("failed to close response body", "error", err)
	}
}
