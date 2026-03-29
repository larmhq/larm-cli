package config

import "testing"

func TestIsValidKey(t *testing.T) {
	valid := []string{"api_key", "api_url", "access_token", "refresh_token", "token_expires_at", "organization_name"}
	for _, k := range valid {
		if !IsValidKey(k) {
			t.Errorf("expected %q to be valid", k)
		}
	}

	invalid := []string{"garbage", "password", "secret", ""}
	for _, k := range invalid {
		if IsValidKey(k) {
			t.Errorf("expected %q to be invalid", k)
		}
	}
}
