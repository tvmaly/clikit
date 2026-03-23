package transport

import "time"

// Config holds all transport configuration.
// Resolution order: explicit flags > env vars > config file > defaults.
type Config struct {
	// Mode selects transport: "http" or "filequeue".
	Mode string

	// HTTP transport settings.
	APIURL   string // base URL for direct HTTP
	APIToken string // bearer token

	// File-queue transport settings.
	QueueRoot      string
	PollInterval   time.Duration
	RequestTimeout time.Duration
	WorkerID       string
	StaleClaim     time.Duration

	// Worker-side only.
	HandlerConfigPath string
	RetentionDone     time.Duration
	RetentionDead     time.Duration
	CleanupInterval   time.Duration
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Mode:           "http",
		PollInterval:   500 * time.Millisecond,
		RequestTimeout: 120 * time.Second,
		StaleClaim:     300 * time.Second,
		RetentionDone:  24 * time.Hour,
		RetentionDead:  72 * time.Hour,
		CleanupInterval: time.Hour,
	}
}
