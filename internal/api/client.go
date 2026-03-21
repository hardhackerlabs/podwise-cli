package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const defaultTimeout = 30 * time.Second

// Client is a thin HTTP client for the Podwise REST API.
// All requests are authenticated with a Bearer token derived from APIKey.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithTimeout overrides the default HTTP timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithHTTPClient replaces the underlying *http.Client (useful in tests).
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// New creates a Client pointed at baseURL and authenticated with apiKey.
func New(baseURL, apiKey string, opts ...Option) *Client {
	c := &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// APIError is returned when the server responds with a non-2xx status.
type APIError struct {
	StatusCode int
	ErrCode    string // parsed from the JSON "error" field
	Message    string // parsed from the JSON "message" field, or raw body as fallback
}

func (e *APIError) Error() string {
	label := statusLabel(e.StatusCode)
	if label != "" {
		return fmt.Sprintf("%s: %s", label, e.Message)
	}
	return fmt.Sprintf("api error %d: %s", e.StatusCode, e.Message)
}

// statusLabel returns a human-readable label for well-known API error codes.
func statusLabel(code int) string {
	switch code {
	case 400:
		return "bad request"
	case 401:
		return "authentication failed"
	case 402:
		return "paid plan required"
	case 404:
		return "not found"
	case 429:
		return "rate limit exceeded"
	default:
		return ""
	}
}

// parseErrorResponse tries to extract the "error" and "message" fields from a JSON error body.
// Falls back to the raw body string for the message when the body is not valid JSON.
func parseErrorResponse(body []byte) (string, string) {
	var payload struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && (payload.Message != "" || payload.Error != "") {
		return payload.Error, payload.Message
	}
	return "", string(body)
}

// Get performs a GET request to path (relative to baseURL) and decodes the
// JSON body into out. query params may be nil.
func (c *Client) Get(ctx context.Context, path string, query url.Values, out any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, out)
}

// Post performs a POST request with a JSON-encoded body and decodes the
// JSON response into out. body may be nil.
func (c *Client) Post(ctx context.Context, path string, body any, out any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, out)
}

// do is the single entry-point for all HTTP calls.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}
	u = u.JoinPath(path)
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http %s %s: %w", method, u.String(), err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errCode, errMsg := parseErrorResponse(respBody)
		return &APIError{
			StatusCode: resp.StatusCode,
			ErrCode:    errCode,
			Message:    errMsg,
		}
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}

	return nil
}
