package api

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"time"
)

const maxRetries = 3

// RetryTransport wraps an http.RoundTripper with retry logic for transient failures.
// Retries on 429 (rate limit) and 5xx errors with exponential backoff.
// Respects Retry-After header on 429 responses.
// Buffers the request body so it can be replayed on retries.
type RetryTransport struct {
	Base http.RoundTripper
}

func (t *RetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}

	// Buffer the request body so we can replay it on retries
	var bodyBytes []byte
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			return nil, err
		}
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			backoff := retryDelay(attempt, resp)
			select {
			case <-time.After(backoff):
			case <-req.Context().Done():
				return nil, req.Context().Err()
			}
		}

		// Drain and close previous response body to allow connection reuse
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}

		// Reset the request body for each attempt
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			req.ContentLength = int64(len(bodyBytes))
		}

		resp, err = base.RoundTrip(req)
		if err != nil {
			continue
		}

		if !shouldRetry(resp.StatusCode) {
			return resp, nil
		}
	}

	return resp, err
}

func shouldRetry(statusCode int) bool {
	return statusCode == 429 || statusCode >= 500
}

func retryDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil && resp.StatusCode == 429 {
		if after := resp.Header.Get("Retry-After"); after != "" {
			if seconds, err := strconv.Atoi(after); err == nil {
				return time.Duration(seconds) * time.Second
			}
		}
	}

	return time.Duration(1<<(attempt-1)) * time.Second
}
