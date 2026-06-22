package middleware

import (
	"net/http"
	"strings"
)

// defaultOrigins are always allowed, matching the Python backend's dev defaults.
var defaultOrigins = []string{
	"http://localhost:3000",
	"http://localhost:5173",
	"http://localhost:4173",
}

// CORS implements the cross-origin policy for the chi router. It mirrors the
// Python CORSMiddleware configuration: explicit origins, credentials allowed,
// a fixed method/header set, and a 10-minute preflight cache.
type CORS struct {
	allowed map[string]struct{}
}

// NewCORS returns a CORS middleware configured with the default localhost
// origins plus any comma-separated values from frontendURL (typically the
// FRONTEND_URL environment variable). Trailing slashes are stripped so env
// values match the Origin header sent by browsers.
func NewCORS(frontendURL string) *CORS {
	allowed := make(map[string]struct{}, len(defaultOrigins))
	for _, o := range defaultOrigins {
		allowed[o] = struct{}{}
	}

	for _, o := range strings.Split(frontendURL, ",") {
		o = strings.TrimSpace(o)
		o = strings.TrimSuffix(o, "/")
		if o == "" {
			continue
		}
		allowed[o] = struct{}{}
	}

	return &CORS{allowed: allowed}
}

// Handler returns an http.Handler that answers CORS preflight OPTIONS requests
// and adds the required access-control headers to regular requests.
func (c *CORS) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && c.isAllowed(origin)

		w.Header().Add("Vary", "Origin")

		if r.Method == http.MethodOptions {
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization, Cache-Control, X-Publish-Token")
				w.Header().Set("Access-Control-Max-Age", "600")
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Range, X-Publish-Token")
		}
		next.ServeHTTP(w, r)
	})
}

// isAllowed reports whether origin is in the configured allow-list. The empty
// string is never allowed.
func (c *CORS) isAllowed(origin string) bool {
	if origin == "" {
		return false
	}
	_, ok := c.allowed[origin]
	return ok
}
