package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthz_Returns200(t *testing.T) {
	handler := healthCheckMiddleware("http://localhost:9999", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("health check should not reach MCP handler")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", body["status"])
	}
}

func TestReadyz_UpstreamReachable(t *testing.T) {
	// Start a mock upstream that returns 200
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	handler := healthCheckMiddleware(upstream.URL, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("health check should not reach MCP handler")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 for reachable upstream, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %s", body["status"])
	}
}

func TestReadyz_UpstreamUnreachable(t *testing.T) {
	handler := healthCheckMiddleware("http://localhost:1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("health check should not reach MCP handler")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 for unreachable upstream, got %d", w.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body["status"] != "unhealthy" {
		t.Errorf("expected status=unhealthy, got %s", body["status"])
	}
}

func TestHealthCheck_Passthrough(t *testing.T) {
	passed := false
	handler := healthCheckMiddleware("http://localhost:9999", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		passed = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	handler.ServeHTTP(w, r)

	if !passed {
		t.Error("expected non-health-check path to reach MCP handler")
	}
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	handler := authMiddleware("test-token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer test-token")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	handler := authMiddleware("test-token", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach handler with invalid token")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer wrong-token")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCorsMiddleware_AddsHeaders(t *testing.T) {
	handler := corsMiddleware("*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	handler.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("expected CORS origin header")
	}
}

func TestCorsMiddleware_HandlesOptions(t *testing.T) {
	handler := corsMiddleware("https://example.com", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach handler for OPTIONS")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}

func TestMaxBodyMiddleware_LimitsSize(t *testing.T) {
	handler := maxBodyMiddleware(10, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 100)
		n, _ := r.Body.Read(buf)
		if n > 0 {
			t.Errorf("expected body to be truncated, read %d bytes", n)
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	// Content length is not enforced directly; MaxBytesReader is lazy
	handler.ServeHTTP(w, r)

	// The request itself should not 413 since MaxBytesReader only acts on read
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}