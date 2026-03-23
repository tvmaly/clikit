package worker

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// HandlerConfig is the top-level handler registry configuration.
type HandlerConfig struct {
	Handlers []HandlerEntry `json:"handlers"`
}

// HandlerEntry maps a (tool, operation) pair to a backend REST API endpoint.
type HandlerEntry struct {
	Tool      string `json:"tool"`
	Operation string `json:"operation"`
	Method    string `json:"method"`
	URLPath   string `json:"url_path"`
	BaseURL   string `json:"base_url"`
	Token     string `json:"token"`      // resolved at load time
	TokenEnv  string `json:"token_env"`  // env var name for token
	Timeout   int    `json:"timeout_seconds"`
}

// LoadHandlerConfig reads and parses a handler config JSON file.
// Tokens are resolved from environment variables on load.
func LoadHandlerConfig(path string) (*HandlerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading handler config %q: %w", path, err)
	}
	var cfg HandlerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing handler config %q: %w", path, err)
	}
	for i := range cfg.Handlers {
		h := &cfg.Handlers[i]
		if h.Tool == "" || h.Operation == "" {
			return nil, fmt.Errorf("handler at index %d missing tool or operation", i)
		}
		if h.TokenEnv != "" {
			h.Token = os.Getenv(h.TokenEnv)
		}
	}
	return &cfg, nil
}

// Lookup finds the handler for the given tool and operation.
func (hc *HandlerConfig) Lookup(tool, operation string) (*HandlerEntry, bool) {
	for i := range hc.Handlers {
		h := &hc.Handlers[i]
		if h.Tool == tool && h.Operation == operation {
			return h, true
		}
	}
	return nil, false
}

// BuildURL constructs the full URL for a handler entry, substituting {param}
// placeholders from params. It returns the final URL, the remaining params
// (those not consumed by path substitution), and any error.
func BuildURL(entry *HandlerEntry, params map[string]string) (string, map[string]string, error) {
	path := entry.URLPath
	remaining := make(map[string]string)

	// Substitute {param} placeholders.
	for k, v := range params {
		placeholder := "{" + k + "}"
		if strings.Contains(path, placeholder) {
			path = strings.ReplaceAll(path, placeholder, url.PathEscape(v))
		} else {
			remaining[k] = v
		}
	}

	// Check for unresolved placeholders.
	if strings.Contains(path, "{") {
		return "", nil, fmt.Errorf("unresolved path parameter in %q", path)
	}

	return entry.BaseURL + path, remaining, nil
}
