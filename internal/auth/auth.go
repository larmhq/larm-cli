package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/larmhq/larm-cli/internal/config"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// ClientID is the well-known public OAuth client ID for the Larm CLI.
const ClientID = "larm-cli"

// TokenResponse is the response from the OAuth token endpoint.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// Resolve returns a bearer token from the first available source:
// 1. flagKey (from --api-key flag)
// 2. LARM_API_KEY env var
// 3. Config file api_key
// 4. Config file access_token (auto-refreshes if near expiry)
func Resolve(flagKey, apiURL string) (string, error) {
	if flagKey != "" {
		return flagKey, nil
	}

	if key := config.Get(config.KeyAPIKey); key != "" {
		return key, nil
	}

	if token := config.Get(config.KeyAccessToken); token != "" {
		if needsRefresh() {
			refreshed, err := refreshToken(context.Background(), apiURL)
			if err == nil {
				return refreshed, nil
			}
			_ = config.Set(config.KeyAccessToken, "")
			_ = config.Set(config.KeyRefreshToken, "")
			_ = config.Set(config.KeyExpiresAt, "")
			return "", fmt.Errorf("session expired (refresh failed: %v). Run 'larm auth login' to log in again", err)
		}
		return token, nil
	}

	return "", fmt.Errorf("no API key found. Run 'larm auth login' or set LARM_API_KEY")
}

func needsRefresh() bool {
	expiresStr := config.Get(config.KeyExpiresAt)
	if expiresStr == "" {
		return false
	}

	expiresAt, err := time.Parse(time.RFC3339, expiresStr)
	if err != nil {
		return true
	}

	return time.Until(expiresAt) < 5*time.Minute
}

type tokenRequest struct {
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ClientID     string `json:"client_id"`
	DeviceCode   string `json:"device_code,omitempty"`
}

// MarshalTokenRequest safely marshals a token request to JSON.
func MarshalTokenRequest(grantType, refreshToken, deviceCode string) ([]byte, error) {
	return json.Marshal(tokenRequest{
		GrantType:    grantType,
		RefreshToken: refreshToken,
		ClientID:     ClientID,
		DeviceCode:   deviceCode,
	})
}

func refreshToken(ctx context.Context, baseURL string) (string, error) {
	rt := config.Get(config.KeyRefreshToken)
	if rt == "" {
		return "", fmt.Errorf("no refresh token")
	}

	body, err := MarshalTokenRequest("refresh_token", rt, "")
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/oauth/device/token", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("refresh failed: %s", string(respBody))
	}

	var tokens TokenResponse
	if err := json.Unmarshal(respBody, &tokens); err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	_ = config.Set(config.KeyAccessToken, tokens.AccessToken)
	_ = config.Set(config.KeyRefreshToken, tokens.RefreshToken)
	_ = config.Set(config.KeyExpiresAt, expiresAt.Format(time.RFC3339))

	return tokens.AccessToken, nil
}
