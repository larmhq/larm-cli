package output

import (
	"testing"
)

func TestUnwrapData(t *testing.T) {
	body := `{"data":[{"id":"123","name":"Test"}]}`
	data, err := UnwrapData([]byte(body))
	if err != nil {
		t.Fatal(err)
	}

	if string(data) != `[{"id":"123","name":"Test"}]` {
		t.Errorf("unexpected data: %s", string(data))
	}
}

func TestUnwrapDataMissing(t *testing.T) {
	body := `{"error":"not_found"}`
	_, err := UnwrapData([]byte(body))
	if err == nil {
		t.Fatal("expected error for missing data key")
	}
}

func TestUnwrapDataInvalid(t *testing.T) {
	_, err := UnwrapData([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
