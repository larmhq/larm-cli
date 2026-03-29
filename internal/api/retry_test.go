package api

import (
	"bytes"
	"io"
	"net/http"
	"testing"
)

type mockTransport struct {
	responses []*http.Response
	calls     int
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := m.calls
	if idx >= len(m.responses) {
		idx = len(m.responses) - 1
	}
	m.calls++

	resp := m.responses[idx]
	return resp, nil
}

func makeResp(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader([]byte("{}"))),
	}
}

func TestNoRetryOn200(t *testing.T) {
	mock := &mockTransport{responses: []*http.Response{makeResp(200)}}
	rt := &RetryTransport{Base: mock}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if mock.calls != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls)
	}
}

func TestNoRetryOn4xx(t *testing.T) {
	mock := &mockTransport{responses: []*http.Response{makeResp(404)}}
	rt := &RetryTransport{Base: mock}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
	if mock.calls != 1 {
		t.Errorf("expected 1 call, got %d", mock.calls)
	}
}

func TestRetryOn5xx(t *testing.T) {
	mock := &mockTransport{
		responses: []*http.Response{makeResp(500), makeResp(500), makeResp(200)},
	}
	rt := &RetryTransport{Base: mock}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retries, got %d", resp.StatusCode)
	}
	if mock.calls != 3 {
		t.Errorf("expected 3 calls, got %d", mock.calls)
	}
}

func TestRetryOn429(t *testing.T) {
	resp429 := makeResp(429)
	resp429.Header.Set("Retry-After", "0")

	mock := &mockTransport{
		responses: []*http.Response{resp429, makeResp(200)},
	}
	rt := &RetryTransport{Base: mock}

	req, _ := http.NewRequest("GET", "http://example.com", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retry, got %d", resp.StatusCode)
	}
	if mock.calls != 2 {
		t.Errorf("expected 2 calls, got %d", mock.calls)
	}
}

func TestRequestBodyPreservedOnRetry(t *testing.T) {
	var bodies []string
	mock := &mockTransport{
		responses: []*http.Response{makeResp(500), makeResp(200)},
	}

	original := mock.RoundTrip
	_ = original // suppress unused

	// Wrap to capture body
	captureTransport := &bodyCapture{
		base:   mock,
		bodies: &bodies,
	}

	rt := &RetryTransport{Base: captureTransport}

	body := `{"name":"test"}`
	req, _ := http.NewRequest("POST", "http://example.com", bytes.NewReader([]byte(body)))
	_, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}

	if len(bodies) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(bodies))
	}
	if bodies[0] != body {
		t.Errorf("first attempt body: %q, want %q", bodies[0], body)
	}
	if bodies[1] != body {
		t.Errorf("second attempt body: %q, want %q", bodies[1], body)
	}
}

type bodyCapture struct {
	base   http.RoundTripper
	bodies *[]string
}

func (b *bodyCapture) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		data, _ := io.ReadAll(req.Body)
		*b.bodies = append(*b.bodies, string(data))
		req.Body = io.NopCloser(bytes.NewReader(data))
	}
	return b.base.RoundTrip(req)
}
