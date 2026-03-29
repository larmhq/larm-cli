package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	data := json.RawMessage(`[{"name":"Test","id":"123"}]`)

	err := Print(&buf, "json", "", "", data)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), `"name"`) {
		t.Errorf("expected JSON output, got: %s", buf.String())
	}
}

func TestPrintJSONWithFields(t *testing.T) {
	var buf bytes.Buffer
	data := json.RawMessage(`[{"name":"Test","id":"123","extra":"drop"}]`)

	err := Print(&buf, "json", "", "name,id", data)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(buf.String(), "extra") {
		t.Errorf("field filtering did not drop 'extra': %s", buf.String())
	}
}

func TestPrintJSONWithJQ(t *testing.T) {
	var buf bytes.Buffer
	data := json.RawMessage(`[{"name":"A"},{"name":"B"}]`)

	err := Print(&buf, "json", ".[].name", "", data)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.TrimSpace(buf.String())
	if lines != "\"A\"\n\"B\"" {
		t.Errorf("unexpected JQ output: %q", lines)
	}
}

func TestPrintTable(t *testing.T) {
	var buf bytes.Buffer
	data := json.RawMessage(`[{"name":"Test","check_type":"http"}]`)

	err := Print(&buf, "table", "", "name,check_type", data)
	if err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "name") || !strings.Contains(output, "Test") {
		t.Errorf("expected table with name column, got: %s", output)
	}
}

func TestPrintTableEmpty(t *testing.T) {
	var buf bytes.Buffer
	data := json.RawMessage(`[]`)

	err := Print(&buf, "table", "", "", data)
	if err != nil {
		t.Fatal(err)
	}

	if buf.String() != "" {
		t.Errorf("expected empty output for empty array, got: %q", buf.String())
	}
}

func TestResolveFieldDotPath(t *testing.T) {
	obj := map[string]any{
		"config": map[string]any{
			"url": "https://example.com",
		},
	}

	result := ResolveField(obj, "config.url")
	if result != "https://example.com" {
		t.Errorf("expected https://example.com, got: %s", result)
	}
}

func TestResolveFieldNil(t *testing.T) {
	obj := map[string]any{"name": nil}

	result := ResolveField(obj, "name")
	if result != "" {
		t.Errorf("expected empty string for nil, got: %q", result)
	}
}

func TestResolveFieldNestedName(t *testing.T) {
	obj := map[string]any{
		"org": map[string]any{"name": "Acme"},
	}

	result := ResolveField(obj, "org")
	if result != "Acme" {
		t.Errorf("expected 'Acme' (extracted from .name), got: %q", result)
	}
}
