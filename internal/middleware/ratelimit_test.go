package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`ok`))
	})
}

func TestRateLimiter_Pass(t *testing.T) {
	handler := NewRateLimiter(100, 200)(okHandler())

	for i := range 50 {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: got status %d, want %d", i, w.Code, http.StatusOK)
		}
		if w.Body.String() != "ok" {
			t.Errorf("request %d: got body %q, want %q", i, w.Body.String(), "ok")
		}
	}
}

func TestRateLimiter_Block(t *testing.T) {
	handler := NewRateLimiter(1, 1)(okHandler())

	// First request consumes the sole token — passes instantly.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("first request: got status %d, want %d", w.Code, http.StatusOK)
	}

	// Subsequent requests must wait ~1 s for a new token. Use a short
	// deadline so Wait(ctx) returns a context error → 429.
	var blocked int
	for range 5 {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		w := httptest.NewRecorder()
		r := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)

		if w.Code == http.StatusTooManyRequests {
			blocked++
			var body map[string]string
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode 429 body: %v", err)
			}
			if body["error"] != "rate limit exceeded" {
				t.Errorf("got error message %q, want %q", body["error"], "rate limit exceeded")
			}
		}
	}

	if blocked == 0 {
		t.Error("expected at least one 429 response")
	}
}

func TestRateLimiter_Disabled(t *testing.T) {
	handler := NewRateLimiter(0, 0)(okHandler())

	for range 100 {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("disabled limiter returned status %d", w.Code)
		}
	}
}

func TestRateLimiter_DefaultBurst(t *testing.T) {
	// With burst=0, the middleware internally sets burst=rps (=10).
	// Two immediate requests should both pass.
	handler := NewRateLimiter(10, 0)(okHandler())

	for i := range 10 {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("request %d with burst default: got status %d, want %d",
				i, w.Code, http.StatusOK)
		}
	}
}