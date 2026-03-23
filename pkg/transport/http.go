package transport

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// HTTPTransport implements Transport using direct HTTP requests.
type HTTPTransport struct {
	client  *http.Client
	baseURL string
	token   string
}

// NewHTTPTransport creates an HTTPTransport from cfg.
func NewHTTPTransport(cfg *Config) (*HTTPTransport, error) {
	timeout := cfg.RequestTimeout
	if timeout == 0 {
		timeout = DefaultConfig().RequestTimeout
	}
	return &HTTPTransport{
		client:  &http.Client{Timeout: timeout},
		baseURL: cfg.APIURL,
		token:   cfg.APIToken,
	}, nil
}

// Execute performs an HTTP request and returns the response.
func (t *HTTPTransport) Execute(ctx context.Context, req *Request) (*Response, error) {
	// Build URL.
	u, err := url.Parse(t.baseURL + req.Path)
	if err != nil {
		return &Response{Error: err.Error()}, err
	}
	if len(req.Params) > 0 {
		q := u.Query()
		for k, v := range req.Params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	// Build request.
	var bodyReader io.Reader
	if len(req.Body) > 0 {
		bodyReader = bytes.NewReader(req.Body)
	}
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, u.String(), bodyReader)
	if err != nil {
		return &Response{Error: err.Error()}, err
	}
	if t.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+t.token)
	}
	if len(req.Body) > 0 {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Execute.
	httpResp, err := t.client.Do(httpReq)
	if err != nil {
		return &Response{Error: err.Error()}, fmt.Errorf("http request: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return &Response{StatusCode: httpResp.StatusCode, Error: err.Error()}, err
	}
	return &Response{StatusCode: httpResp.StatusCode, Body: body}, nil
}

// Close is a no-op for HTTPTransport.
func (t *HTTPTransport) Close() error { return nil }
