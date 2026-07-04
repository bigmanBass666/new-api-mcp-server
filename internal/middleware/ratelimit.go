package middleware

import (
	"encoding/json"
	"net/http"

	"golang.org/x/time/rate"
)

// NewRateLimiter returns an HTTP middleware that limits request rate using a
// token-bucket algorithm. When rps <= 0, a no-op middleware is returned (no
// rate limiting applied). When burst <= 0, it defaults to rps.
func NewRateLimiter(rps, burst int) func(http.Handler) http.Handler {
	if rps <= 0 {
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	if burst <= 0 {
		burst = rps
	}

	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := limiter.Wait(r.Context()); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]string{"error": "rate limit exceeded"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}