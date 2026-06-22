// Package middleware provides HTTP middleware for the lyrike-studio-tui Go
// backend: per-IP token-bucket rate limiting and CORS handling.
package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// GetRealIP returns the best-effort client IP for request r using the same
// precedence as the Python backend: CF-Connecting-IP, then the first entry of
// X-Forwarded-For, then the host portion of RemoteAddr.
func GetRealIP(r *http.Request) string {
	if cf := r.Header.Get("CF-Connecting-IP"); cf != "" {
		return strings.TrimSpace(cf)
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// routeConfig describes the token-bucket parameters for a single route group.
type routeConfig struct {
	name  string
	limit rate.Limit
	burst int
}

// RateLimiter enforces per-IP token-bucket limits for the route groups the
// TUI backend exposes. It is safe for concurrent use.
type RateLimiter struct {
	mu         sync.Mutex
	limiters   map[string]*rate.Limiter
	fetch      routeConfig
	transcribe routeConfig
	cache      routeConfig
}

// NewRateLimiter builds a RateLimiter with the production defaults from the
// Python backend: 60/min for /local-api/fetch, 5/min for /local-api/transcribe,
// and 120/min for /cache/*.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		fetch: routeConfig{
			name:  "fetch",
			limit: rate.Every(time.Minute / 60),
			burst: 60,
		},
		transcribe: routeConfig{
			name:  "transcribe",
			limit: rate.Every(time.Minute / 5),
			burst: 5,
		},
		cache: routeConfig{
			name:  "cache",
			limit: rate.Every(time.Minute / 120),
			burst: 120,
		},
	}
}

// Handler returns an http.Handler that rate-limits requests before forwarding
// them to next. Routes outside the configured groups pass through untouched.
func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := rl.configForPath(r.URL.Path)
		if cfg.name == "" {
			next.ServeHTTP(w, r)
			return
		}

		ip := GetRealIP(r)
		limiter := rl.getLimiter(ip, cfg)
		if !limiter.Allow() {
			writeRateLimitResponse(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// configForPath returns the routeConfig for path, or a zero value if the path
// is not covered by any per-route limit.
func (rl *RateLimiter) configForPath(path string) routeConfig {
	switch {
	case path == "/local-api/fetch":
		return rl.fetch
	case path == "/local-api/transcribe":
		return rl.transcribe
	case strings.HasPrefix(path, "/cache/"):
		return rl.cache
	default:
		return routeConfig{}
	}
}

// getLimiter returns the token bucket for the given IP and route, creating it
// on first use. Each (IP, route) pair gets its own independent bucket.
func (rl *RateLimiter) getLimiter(ip string, cfg routeConfig) *rate.Limiter {
	key := ip + ":" + cfg.name

	rl.mu.Lock()
	defer rl.mu.Unlock()

	if l, ok := rl.limiters[key]; ok {
		return l
	}

	l := rate.NewLimiter(cfg.limit, cfg.burst)
	rl.limiters[key] = l
	return l
}

// writeRateLimitResponse writes the JSON 429 response expected by the TUI
// contract, including a Retry-After header of 60 seconds.
func writeRateLimitResponse(w http.ResponseWriter) {
	body := map[string]string{
		"error":       "rate_limit_exceeded",
		"detail":      "Too many requests. Please slow down.",
		"retry_after": "60",
	}
	data, _ := json.Marshal(body)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", "60")
	w.WriteHeader(http.StatusTooManyRequests)
	_, _ = w.Write(data)
}
