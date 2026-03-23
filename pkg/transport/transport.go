package transport

import (
	"context"
	"fmt"

	"github.com/tvmaly/clikit/pkg/schema"
)

// Transport defines how tool commands communicate with backend APIs.
// Implementations must be safe for concurrent use.
type Transport interface {
	// Execute sends a request and returns the raw response body.
	Execute(ctx context.Context, req *Request) (*Response, error)
	// Close releases any resources held by the transport.
	Close() error
}

// Request represents a tool operation request, transport-agnostic.
type Request struct {
	Tool      string            // logical tool name
	Operation string            // logical operation
	Method    string            // HTTP method: GET, POST, PUT, DELETE
	Path      string            // API path
	Params    map[string]string // query parameters
	Body      []byte            // request body (POST/PUT), may be nil
	Schema    *schema.Schema    // optional schema for response transformation
}

// Response represents the result from a transport execution.
type Response struct {
	StatusCode int    // HTTP-equivalent status code
	Body       []byte // raw response body
	Error      string // transport-level error message
}

// NewTransport is a factory that returns the appropriate Transport based on cfg.Mode.
func NewTransport(cfg *Config) (Transport, error) {
	switch cfg.Mode {
	case "http":
		return NewHTTPTransport(cfg)
	case "filequeue":
		return NewFileQueueTransport(cfg)
	default:
		return nil, fmt.Errorf("unknown transport mode %q (want \"http\" or \"filequeue\")", cfg.Mode)
	}
}
