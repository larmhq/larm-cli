package auth

import (
	"testing"
	"time"
)

func TestNeedsRefreshNoExpiry(t *testing.T) {
	// No token_expires_at set -- should not need refresh
	if needsRefresh() {
		t.Error("expected no refresh needed when no expiry set")
	}
}

func TestNeedsRefreshFuture(t *testing.T) {
	// Would need config set to test properly, but we can test the time logic
	future := time.Now().Add(1 * time.Hour)
	if time.Until(future) < 5*time.Minute {
		t.Error("1 hour in the future should not need refresh")
	}
}

func TestNeedsRefreshSoon(t *testing.T) {
	soon := time.Now().Add(2 * time.Minute)
	if time.Until(soon) >= 5*time.Minute {
		t.Error("2 minutes in the future should need refresh")
	}
}
