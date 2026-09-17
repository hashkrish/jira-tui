// Package jiraclient is a minimal, read-only client for the Jira Cloud
// REST API (v3) and the Jira Agile API. Only GET requests are implemented.
package jiraclient

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/hashkrish/jira-tui/internal/config"
)

// Client performs authenticated read-only requests against a Jira instance.
type Client struct {
	baseURL    string
	authHeader string
	httpClient *http.Client
}

// APIError represents a non-2xx response from the Jira API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("jira API error: status %d: %s", e.StatusCode, e.Body)
}

// New builds a Client from a resolved Config.
func New(cfg *config.Config) *Client {
	var authHeader string
	switch cfg.AuthMode {
	case "bearer":
		authHeader = "Bearer " + cfg.APIToken
	default:
		creds := base64.StdEncoding.EncodeToString([]byte(cfg.Email + ":" + cfg.APIToken))
		authHeader = "Basic " + creds
	}

	return &Client{
		baseURL:    cfg.BaseURL,
		authHeader: authHeader,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// getQuery performs a GET request against path with the given query
// parameters and decodes the JSON response body into out.
func (c *Client) getQuery(path string, params url.Values, out any) error {
	if len(params) > 0 {
		path = path + "?" + params.Encode()
	}
	return c.get(path, out)
}

// get performs a GET request against path (relative to baseURL) and decodes
// the JSON response body into out. It retries once on 429 honoring
// Retry-After.
func (c *Client) get(path string, out any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("performing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		wait := retryAfter(resp.Header.Get("Retry-After"))
		time.Sleep(wait)
		return c.get(path, out)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}
	return nil
}

func retryAfter(header string) time.Duration {
	if header == "" {
		return 2 * time.Second
	}
	if secs, err := time.ParseDuration(header + "s"); err == nil {
		return secs
	}
	return 2 * time.Second
}

// Myself is the response from GET /rest/api/3/myself, used to verify auth.
type Myself struct {
	AccountID    string `json:"accountId"`
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

// Myself checks connectivity and credentials by fetching the current user.
func (c *Client) Myself() (*Myself, error) {
	var me Myself
	if err := c.get("/rest/api/3/myself", &me); err != nil {
		return nil, err
	}
	return &me, nil
}
