package ratelimit

import (
	"sync"
	"time"
)

// Info carries rate limit state for a single request, used to populate response headers.
// Fields with a zero LimitX value indicate that dimension is unconfigured.
type Info struct {
	LimitRequests     int
	LimitTokens       int
	RemainingRequests int
	RemainingTokens   int
	ResetRequests     time.Duration
	ResetTokens       time.Duration
	// ExceededBy is "requests" or "tokens" when Allow returns false; empty otherwise.
	ExceededBy string
}

type window struct {
	minute   time.Time
	requests int
	tokens   int
}

// Limiter enforces per-key RPM and TPM limits using fixed 1-minute UTC windows.
// Keys are derived from the request's bearer token; a zero key covers unauthenticated requests.
type Limiter struct {
	mu                sync.Mutex
	requestsPerMinute int
	tokensPerMinute   int
	windows           map[string]*window
}

// New creates a Limiter with the given limits. A zero value means unlimited for that dimension.
func New(requestsPerMinute, tokensPerMinute int) *Limiter {
	return &Limiter{
		requestsPerMinute: requestsPerMinute,
		tokensPerMinute:   tokensPerMinute,
		windows:           make(map[string]*window),
	}
}

// Enabled reports whether at least one limit dimension is configured.
func (l *Limiter) Enabled() bool {
	return l.requestsPerMinute > 0 || l.tokensPerMinute > 0
}

// Allow checks whether a request from key with estimatedTokens prompt tokens is within
// limits. If allowed it increments counters and returns (Info, true). If a limit would
// be exceeded it returns (Info, false) without modifying counters; Info.ExceededBy names
// the exceeded dimension.
func (l *Limiter) Allow(key string, estimatedTokens int) (Info, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	minute := now.Truncate(time.Minute)
	resetIn := time.Minute - now.Sub(minute)

	w := l.windowFor(key, minute)

	// Check RPM first.
	if l.requestsPerMinute > 0 && w.requests >= l.requestsPerMinute {
		return Info{
			LimitRequests:     l.requestsPerMinute,
			LimitTokens:       l.tokensPerMinute,
			RemainingRequests: 0,
			RemainingTokens:   max(0, l.tokensPerMinute-w.tokens),
			ResetRequests:     resetIn,
			ResetTokens:       resetIn,
			ExceededBy:        "requests",
		}, false
	}

	// Check TPM.
	if l.tokensPerMinute > 0 && w.tokens+estimatedTokens > l.tokensPerMinute {
		return Info{
			LimitRequests:     l.requestsPerMinute,
			LimitTokens:       l.tokensPerMinute,
			RemainingRequests: max(0, l.requestsPerMinute-w.requests),
			RemainingTokens:   0,
			ResetRequests:     resetIn,
			ResetTokens:       resetIn,
			ExceededBy:        "tokens",
		}, false
	}

	w.requests++
	w.tokens += estimatedTokens

	return Info{
		LimitRequests:     l.requestsPerMinute,
		LimitTokens:       l.tokensPerMinute,
		RemainingRequests: remainingOrUnlimited(l.requestsPerMinute, w.requests),
		RemainingTokens:   remainingOrUnlimited(l.tokensPerMinute, w.tokens),
		ResetRequests:     resetIn,
		ResetTokens:       resetIn,
	}, true
}

// windowFor returns the window for key, resetting it if the minute has advanced.
// It opportunistically evicts windows from past minutes so the map does not grow
// unboundedly with distinct keys (e.g. rotating bearer tokens).
func (l *Limiter) windowFor(key string, minute time.Time) *window {
	w, ok := l.windows[key]
	if !ok {
		l.evictStale(minute)

		w = &window{minute: minute}
		l.windows[key] = w

		return w
	}

	if !w.minute.Equal(minute) {
		w.minute = minute
		w.requests = 0
		w.tokens = 0
	}

	return w
}

// evictStale removes windows belonging to a minute older than the current one.
// Callers must hold l.mu.
func (l *Limiter) evictStale(minute time.Time) {
	for k, w := range l.windows {
		if w.minute.Before(minute) {
			delete(l.windows, k)
		}
	}
}

// remainingOrUnlimited returns limit-used when limit > 0, or -1 (unconfigured) otherwise.
func remainingOrUnlimited(limit, used int) int {
	if limit == 0 {
		return -1
	}

	return limit - used
}
