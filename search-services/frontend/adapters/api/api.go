package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"search-service/frontend/core"
	"time"
)

const (
	pingEndpoint = "/api/ping"

	searchEndpoint = "/api/search"
	maxSearchLimit = 10000

	statusEndpoint = "/api/db/status"
	statsEndpoint  = "/api/db/stats"

	updateEndpoint = "/api/db/update"
	dropEndpoint   = "/api/db"
)

type Client struct {
	log     *slog.Logger
	client  http.Client
	address string
}

func NewClient(address string, timeout time.Duration, log *slog.Logger) *Client {
	return &Client{
		client:  http.Client{Timeout: timeout},
		log:     log,
		address: address,
	}
}

func (c *Client) Ping(ctx context.Context) (core.PingResponse, error) {
	var reply core.PingResponse
	if err := c.doGetEndpoint(ctx, pingEndpoint, &reply); err != nil {
		return core.PingResponse{}, fmt.Errorf("failed to get ping result: %w", err)
	}
	return reply, nil
}

func (c *Client) Search(ctx context.Context, phrase string) (core.SearchResult, error) {
	u, err := url.JoinPath(c.address, searchEndpoint)
	if err != nil {
		return core.SearchResult{}, fmt.Errorf("cannot join url path: %w", err)
	}

	parsedURL, err := url.Parse(u)
	if err != nil {
		return core.SearchResult{}, fmt.Errorf("cannot parse url: %w", err)
	}

	q := parsedURL.Query()
	q.Set("phrase", phrase)
	q.Set("limit", fmt.Sprintf("%d", maxSearchLimit))
	parsedURL.RawQuery = q.Encode()

	var reply core.SearchResult
	if err := c.doGet(ctx, parsedURL.String(), &reply); err != nil {
		return core.SearchResult{}, fmt.Errorf("failed to get search result: %w", err)
	}
	return reply, nil
}

func (c *Client) GetUpdateStats(ctx context.Context) (core.UpdateStats, error) {
	var reply core.UpdateStats
	if err := c.doGetEndpoint(ctx, statsEndpoint, &reply); err != nil {
		return core.UpdateStats{}, fmt.Errorf("failed to get update stats: %w", err)
	}
	return reply, nil
}

func (c *Client) GetUpdateStatus(ctx context.Context) (core.UpdateStatus, error) {
	var reply struct {
		Status core.UpdateStatus `json:"status"`
	}
	if err := c.doGetEndpoint(ctx, statusEndpoint, &reply); err != nil {
		return "", fmt.Errorf("failed to get update status: %w", err)
	}
	return reply.Status, nil
}

func (c *Client) doGetEndpoint(ctx context.Context, endpoint string, result interface{}) error {
	fullURL, err := url.JoinPath(c.address, endpoint)
	if err != nil {
		return fmt.Errorf("cannot join url path: %w", err)
	}
	return c.doGet(ctx, fullURL, result)
}

func (c *Client) doGet(ctx context.Context, fullURL string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot get response: %w", err)
	}
	defer c.closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return core.ErrBadArguments
		case http.StatusServiceUnavailable:
			return core.ErrServiceUnavailable
		default:
			return fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("cannot decode reply: %w", err)
	}
	return nil
}

func (c *Client) Update(ctx context.Context) error {
	return c.doMutateEndpoint(ctx, http.MethodPost, updateEndpoint)
}

func (c *Client) Drop(ctx context.Context) error {
	return c.doMutateEndpoint(ctx, http.MethodDelete, dropEndpoint)
}

func (c *Client) doMutateEndpoint(ctx context.Context, method, endpoint string) error {
	fullURL, err := url.JoinPath(c.address, endpoint)
	if err != nil {
		return fmt.Errorf("cannot join url path: %w", err)
	}
	return c.doMutate(ctx, method, fullURL)
}

func (c *Client) doMutate(ctx context.Context, method, fullURL string) error {
	req, err := http.NewRequestWithContext(ctx, method, fullURL, nil)
	if err != nil {
		return fmt.Errorf("cannot create request: %w", err)
	}

	if tokenValue := ctx.Value(core.JwtTokenContextKey); tokenValue != nil {
		if token, ok := tokenValue.(string); ok {
			req.Header.Set("Authorization", "Token "+token)
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot get response: %w", err)
	}
	defer c.closeBody(resp.Body)

	if resp.StatusCode != http.StatusOK {
		switch resp.StatusCode {
		case http.StatusAccepted:
			return core.ErrAlreadyExists
		case http.StatusServiceUnavailable:
			return core.ErrServiceUnavailable
		default:
			return fmt.Errorf("unexpected status code %d", resp.StatusCode)
		}
	}
	return nil
}

func (c *Client) closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		c.log.Warn("failed to close response body", "error", err)
	}
}
