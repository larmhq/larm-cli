package output

import (
	"encoding/json"
	"fmt"
)

// UnwrapData extracts the "data" key from an API response envelope.
func UnwrapData(body []byte) (json.RawMessage, error) {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if envelope.Data == nil {
		return nil, fmt.Errorf("response missing 'data' key")
	}
	return envelope.Data, nil
}
