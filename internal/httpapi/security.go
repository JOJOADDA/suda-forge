package httpapi

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// fixedWindowLimiter is intentionally process-local for the single-binary deployment.
// A shared store can replace it when the API is horizontally scaled.
type fixedWindowLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	items  map[string]rateWindow
}

type rateWindow struct {
	count int
	until time.Time
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{limit: limit, window: window, items: make(map[string]rateWindow)}
}

func (l *fixedWindowLimiter) allow(key string, now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.items[key]
	if !ok || !now.Before(entry.until) {
		l.items[key] = rateWindow{count: 1, until: now.Add(l.window)}
		return true, 0
	}
	if entry.count >= l.limit {
		return false, entry.until.Sub(now)
	}
	entry.count++
	l.items[key] = entry
	return true, 0
}

var (
	loginLimiter     = newFixedWindowLimiter(10, time.Minute)
	bootstrapLimiter = newFixedWindowLimiter(3, time.Hour)
)

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// sameOriginGuard protects cookie-authenticated state changes while keeping
// non-browser clients usable: requests without Origin are allowed, but an
// explicit cross-origin browser request is rejected.
func sameOriginGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && !requestOriginMatches(r, origin) {
			writeError(w, http.StatusForbidden, errCrossOriginRequest)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var errCrossOriginRequest = errorString("cross-origin request rejected")

type errorString string

func (e errorString) Error() string { return string(e) }

func requestOriginMatches(r *http.Request, origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	forwardedProto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])
	scheme := forwardedProto
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	forwardedHost := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Host"), ",")[0])
	host := forwardedHost
	if host == "" {
		host = r.Host
	}
	return strings.EqualFold(parsed.Scheme, scheme) && strings.EqualFold(parsed.Host, host)
}

func enforceRateLimit(w http.ResponseWriter, limiter *fixedWindowLimiter, key string) bool {
	allowed, retryAfter := limiter.allow(key, time.Now())
	if allowed {
		return true
	}
	seconds := int(retryAfter.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	w.Header().Set("Retry-After", strconvItoa(seconds))
	writeError(w, http.StatusTooManyRequests, errorString("too many requests; retry later"))
	return false
}

func strconvItoa(value int) string {
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	for value > 0 {
		pos--
		buf[pos] = byte('0' + value%10)
		value /= 10
	}
	return string(buf[pos:])
}
