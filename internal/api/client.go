package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a lightweight HTTP client for raw API calls and identity checks.
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new raw API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:    baseURL,
		APIKey:     apiKey,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// IdentityResponse is the response from GET /api/v1/me.
type IdentityResponse struct {
	Data struct {
		Organization struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"organization"`
	} `json:"data"`
}

// GetIdentity calls GET /api/v1/me to verify the API key.
func (c *Client) GetIdentity(ctx context.Context) (*IdentityResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/api/v1/me", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var identity IdentityResponse
	if err := json.Unmarshal(body, &identity); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return &identity, nil
}
