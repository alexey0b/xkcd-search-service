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
	maxSearchLimit = 10000
	searchEndpoint = "/api/isearch"

	statusEndpoint = "/api/db/status"
	statsEndpoint  = "/api/db/stats"

	updateEndpoint = "/api/db/update"
	dropEndpoint   = "/api/db"
)

// Client is an HTTP client for the API service.
type Client struct {
	log     *slog.Logger
	client  http.Client
	address string
}

// NewClient creates a new API service HTTP client with specified timeout.
func NewClient(address string, timeout time.Duration, log *slog.Logger) *Client {
	return &Client{
		client:  http.Client{Timeout: timeout},
		log:     log,
		address: address,
	}
}

// Search performs a index search query and returns matching comics.
// Uses indexed search with normalized and stemmed words.
func (c *Client) Search(ctx context.Context, phrase string) (core.SearchResult, error) {
	u, err := url.JoinPath(c.address, searchEndpoint)
	if err != nil {
		return core.SearchResult{}, fmt.Errorf("cannot join url path: %w", err)
	}

	parsedURL, err := url.Parse(u)
	if err != nil {
		return core.SearchResult{}, fmt.Errorf("cannot parse url: %w", err)
	}

	// build query parameters
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

// GetUpdateStats returns current database statistics.
// Includes total/unique words count and fetched/total comics count.
func (c *Client) GetUpdateStats(ctx context.Context) (core.UpdateStats, error) {
	var reply core.UpdateStats
	if err := c.doGetEndpoint(ctx, statsEndpoint, &reply); err != nil {
		return core.UpdateStats{}, fmt.Errorf("failed to get update stats: %w", err)
	}
	return reply, nil
}

// GetUpdateStatus returns the current status of the database update process.
func (c *Client) GetUpdateStatus(ctx context.Context) (core.UpdateStatus, error) {
	var reply struct {
		Status core.UpdateStatus `json:"status"`
	}
	if err := c.doGetEndpoint(ctx, statusEndpoint, &reply); err != nil {
		return "", fmt.Errorf("failed to get update status: %w", err)
	}
	return reply.Status, nil
}

// doGetEndpoint performs GET request to the specified endpoint.
func (c *Client) doGetEndpoint(ctx context.Context, endpoint string, result any) error {
	fullURL, err := url.JoinPath(c.address, endpoint)
	if err != nil {
		return fmt.Errorf("cannot join url path: %w", err)
	}
	return c.doGet(ctx, fullURL, result)
}

// doGet performs GET request and decodes JSON response.
func (c *Client) doGet(ctx context.Context, fullURL string, result any) error {
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

// Update triggers asynchronous database update to fetch new comics from XKCD API.
func (c *Client) Update(ctx context.Context) error {
	return c.doMutateEndpoint(ctx, http.MethodPost, updateEndpoint)
}

// Drop removes all comics and indexed words from the database.
func (c *Client) Drop(ctx context.Context) error {
	return c.doMutateEndpoint(ctx, http.MethodDelete, dropEndpoint)
}

// doMutateEndpoint performs mutating request (POST/DELETE) to the specified endpoint.
func (c *Client) doMutateEndpoint(ctx context.Context, method, endpoint string) error {
	fullURL, err := url.JoinPath(c.address, endpoint)
	if err != nil {
		return fmt.Errorf("cannot join url path: %w", err)
	}
	return c.doMutate(ctx, method, fullURL)
}

// doMutate performs mutating request with JWT token from context.
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

// closeBody closes response body and logs errors.
func (c *Client) closeBody(body io.Closer) {
	if err := body.Close(); err != nil {
		c.log.Warn("failed to close response body", "error", err)
	}
}
