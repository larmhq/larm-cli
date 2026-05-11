package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/larmhq/larm-go/client"
)

const (
	ExitCodeUserError    = 1
	ExitCodeAPIError     = 2
	ExitCodeNetworkError = 3
)

// CLIError wraps a Larm SDK API error with CLI-specific exit code and
// human-readable suggestion. The embedded *client.APIError contributes
// StatusCode, Type, and Message; the CLI adds ExitCode and Suggestion.
type CLIError struct {
	*client.APIError
	ExitCode   int    `json:"-"`
	Suggestion string `json:"suggestion,omitempty"`
}

func (e *CLIError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("%s\n%s", e.APIError.Error(), e.Suggestion)
	}
	return e.APIError.Error()
}

// ParseAPIError parses an API error response body and wraps it with CLI
// concerns (exit code, suggestion).
func ParseAPIError(statusCode int, body []byte) *CLIError {
	return &CLIError{
		APIError:   client.ParseAPIError(statusCode, body),
		ExitCode:   ExitCodeAPIError,
		Suggestion: suggestionForStatus(statusCode),
	}
}

// NewUserError creates an error for bad user input (exit code 1).
func NewUserError(message string) *CLIError {
	return &CLIError{
		APIError: &client.APIError{Type: "user_error", Message: message},
		ExitCode: ExitCodeUserError,
	}
}

// HandleResponse checks the HTTP status code and returns an error for non-2xx responses.
func HandleResponse(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	return ParseAPIError(statusCode, body)
}

// PrintError outputs an error to the writer. In JSON mode, outputs structured JSON.
func PrintError(w io.Writer, err error, jsonMode bool) {
	var cliErr *CLIError
	if errors.As(err, &cliErr) && jsonMode {
		_ = json.NewEncoder(w).Encode(cliErr)
		return
	}
	fmt.Fprintln(w, err)
}

// ExitCodeFor returns the appropriate exit code for an error.
func ExitCodeFor(err error) int {
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr.ExitCode
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
