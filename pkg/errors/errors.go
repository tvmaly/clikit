package errors

import "encoding/json"

// CLIError is the structured error type used throughout clikit.
// It serializes to JSON with the agent-consumable error format.
type CLIError struct {
	Message    string   `json:"error"`
	Suggestion string   `json:"suggestion,omitempty"`
	Available  []string `json:"available,omitempty"`
	Retry      bool     `json:"retry"`
	ExitCode   int      `json:"exit_code"`
}

// Error implements the error interface.
func (e *CLIError) Error() string {
	return e.Message
}

// MarshalJSON serializes CLIError to JSON.
func (e *CLIError) MarshalJSON() ([]byte, error) {
	type alias CLIError
	return json.Marshal((*alias)(e))
}

// New creates a CLIError with a message and exit code.
func New(message string, exitCode int) *CLIError {
	return &CLIError{Message: message, ExitCode: exitCode}
}

// NewWithSuggestion creates a CLIError with a message, suggestion, and exit code.
func NewWithSuggestion(message, suggestion string, exitCode int) *CLIError {
	return &CLIError{Message: message, Suggestion: suggestion, ExitCode: exitCode}
}

// NewRetryable creates a CLIError that the agent should retry.
func NewRetryable(message string) *CLIError {
	return &CLIError{Message: message, Retry: true, ExitCode: 1}
}
