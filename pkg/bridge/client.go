package bridge

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is a simple HTTP client with auth and base URL.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
}

// NewClient creates a Client with the given base URL, bearer token, and timeout.
func NewClient(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: timeout},
		baseURL:    baseURL,
		token:      token,
	}
}

// Do performs an HTTP request. method is GET/POST/etc. path is appended to
// baseURL. body is the request body for POST/PUT (nil for GET). params are
// query parameters. Returns response body, HTTP status code, and error.
func (c *Client) Do(ctx context.Context, method, path string, body []byte, params map[string]string) ([]byte, int, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, 0, fmt.Errorf("parsing URL: %w", err)
	}
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	var bodyReader io.Reader
	if len(body) > 0 {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if len(body) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("reading response: %w", err)
	}
	return respBody, resp.StatusCode, nil
}
