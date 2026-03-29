package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	ExitCodeUserError    = 1
	ExitCodeAPIError     = 2
	ExitCodeNetworkError = 3
)

// APIError represents a structured error from the Larm API.
type APIError struct {
	StatusCode int    `json:"-"`
	ExitCode   int    `json:"-"`
	Type       string `json:"error"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

func (e *APIError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s: %s\n%s", e.Type, e.Message, e.Suggestion)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ParseAPIError attempts to parse an API error response body.
func ParseAPIError(statusCode int, body []byte) *APIError {
	var envelope struct {
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	}

	apiErr := &APIError{
		StatusCode: statusCode,
		ExitCode:   ExitCodeAPIError,
	}

	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error.Type != "" {
		apiErr.Type = envelope.Error.Type
		apiErr.Message = envelope.Error.Message
	} else {
		apiErr.Type = fmt.Sprintf("http_%d", statusCode)
		apiErr.Message = string(body)
	}

	apiErr.Suggestion = suggestionForStatus(statusCode)
	return apiErr
}

// NewUserError creates an error for bad user input (exit code 1).
func NewUserError(message string) *APIError {
	return &APIError{
		ExitCode: ExitCodeUserError,
		Type:     "user_error",
		Message:  message,
	}
}

// HandleResponse checks the HTTP status code and returns an error for non-2xx responses.
func HandleResponse(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	return ParseAPIError(statusCode, body)
}

// PrintError outputs an error to stderr. In JSON mode, outputs structured JSON.
func PrintError(w io.Writer, err error, jsonMode bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) && jsonMode {
		enc := json.NewEncoder(w)
		_ = enc.Encode(apiErr)
		return
	}
	fmt.Fprintln(w, err)
}

// ExitCodeFor returns the appropriate exit code for an error.
func ExitCodeFor(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.ExitCode
	}
	return ExitCodeUserError
}

func suggestionForStatus(statusCode int) string {
	switch statusCode {
	case 401:
		return "Run 'larm auth login' to authenticate."
	case 403:
		return "Your API key may not have permission for this action."
	case 404:
		return "Check that the resource ID exists."
	case 429:
		return "Rate limit exceeded. Wait a moment and try again."
	default:
		return ""
	}
}
