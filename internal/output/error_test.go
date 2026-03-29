package output

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func TestParseAPIError(t *testing.T) {
	body := `{"error":{"type":"not_found","message":"Monitor not found"}}`
	err := ParseAPIError(404, []byte(body))

	if err.Type != "not_found" {
		t.Errorf("expected type not_found, got: %s", err.Type)
	}
	if err.StatusCode != 404 {
		t.Errorf("expected status 404, got: %d", err.StatusCode)
	}
	if err.Suggestion != "Check that the resource ID exists." {
		t.Errorf("expected suggestion, got: %q", err.Suggestion)
	}
}

func TestParseAPIErrorUnknownBody(t *testing.T) {
	err := ParseAPIError(500, []byte("Internal Server Error"))

	if err.Type != "http_500" {
		t.Errorf("expected http_500, got: %s", err.Type)
	}
}

func TestExitCodeFor(t *testing.T) {
	apiErr := &APIError{ExitCode: ExitCodeAPIError}
	if ExitCodeFor(apiErr) != ExitCodeAPIError {
		t.Errorf("expected exit code %d, got: %d", ExitCodeAPIError, ExitCodeFor(apiErr))
	}

	plainErr := fmt.Errorf("something")
	if ExitCodeFor(plainErr) != ExitCodeUserError {
		t.Errorf("expected exit code %d for plain error, got: %d", ExitCodeUserError, ExitCodeFor(plainErr))
	}
}

func TestPrintErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	err := &APIError{Type: "not_found", Message: "gone"}

	PrintError(&buf, err, true)

	if !strings.Contains(buf.String(), `"error":"not_found"`) {
		t.Errorf("expected JSON error output, got: %s", buf.String())
	}
}

func TestPrintErrorPlain(t *testing.T) {
	var buf bytes.Buffer
	err := &APIError{Type: "not_found", Message: "gone"}

	PrintError(&buf, err, false)

	if !strings.Contains(buf.String(), "not_found: gone") {
		t.Errorf("expected plain error, got: %s", buf.String())
	}
}
