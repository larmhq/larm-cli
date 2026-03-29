package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetIdentitySuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Bearer test-key, got %s", r.Header.Get("Authorization"))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"data":{"organization":{"id":1,"name":"Acme"}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	identity, err := client.GetIdentity(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if identity.Data.Organization.Name != "Acme" {
		t.Errorf("expected Acme, got %s", identity.Data.Organization.Name)
	}
}

func TestGetIdentity401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"error":{"type":"unauthorized"}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "bad-key")
	_, err := client.GetIdentity(context.Background())
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if err.Error() != "invalid API key" {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestGetIdentityNon200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "key")
	_, err := client.GetIdentity(context.Background())
	if err == nil {
		t.Fatal("expected error for 500")
	}
}
