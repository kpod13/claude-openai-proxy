package ratelimit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	headerRateLimitLimitRequests     = "X-Ratelimit-Limit-Requests"
	headerRateLimitLimitTokens       = "X-Ratelimit-Limit-Tokens"       //nolint:gosec // header name, not a credential
	headerRateLimitRemainingRequests = "X-Ratelimit-Remaining-Requests"
	headerRateLimitRemainingTokens   = "X-Ratelimit-Remaining-Tokens"   //nolint:gosec // header name, not a credential
	headerRateLimitResetRequests     = "X-Ratelimit-Reset-Requests"
	headerRateLimitResetTokens       = "X-Ratelimit-Reset-Tokens"       //nolint:gosec // header name, not a credential
	headerRetryAfter                 = "Retry-After"
)

// rateLimitError is the OpenAI-compatible error envelope for 429 responses.
type rateLimitError struct {
	Error struct {
		Message string  `json:"message"`
		Type    string  `json:"type"`
		Param   *string `json:"param"`
		Code    string  `json:"code"`
	} `json:"error"`
}

// Middleware returns an HTTP middleware that enforces rate limits using l.
// It reads the request body to estimate token count, then restores the body
// so the inner handler can read it normally.
// If l is nil or not enabled the middleware is a no-op.
func Middleware(l *Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if l == nil || !l.Enabled() {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read request body", http.StatusBadRequest)

				return
			}

			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			key := bearerToken(r)
			estimatedTokens := estimateTokens(bodyBytes)

			info, allowed := l.Allow(key, estimatedTokens)
			setRateLimitHeaders(w, info)

			if !allowed {
				writeRateLimitError(w, info)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// bearerToken extracts the token value from the Authorization header.
// Returns empty string when no bearer token is present.
func bearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if after, ok := strings.CutPrefix(auth, "Bearer "); ok {
		return strings.TrimSpace(after)
	}

	return ""
}

// estimateTokens returns a rough token count from raw request bytes (len/4).
func estimateTokens(body []byte) int {
	est := len(body) / 4
	if est < 1 && len(body) > 0 {
		return 1
	}

	return est
}

// setRateLimitHeaders writes x-ratelimit-* headers to w based on info.
// Headers for unconfigured dimensions (limit == 0) are omitted.
func setRateLimitHeaders(w http.ResponseWriter, info Info) {
	if info.LimitRequests > 0 {
		w.Header().Set(headerRateLimitLimitRequests, strconv.Itoa(info.LimitRequests))
		w.Header().Set(headerRateLimitRemainingRequests, strconv.Itoa(max(0, info.RemainingRequests)))
		w.Header().Set(headerRateLimitResetRequests, formatDuration(info.ResetRequests))
	}

	if info.LimitTokens > 0 {
		w.Header().Set(headerRateLimitLimitTokens, strconv.Itoa(info.LimitTokens))
		w.Header().Set(headerRateLimitRemainingTokens, strconv.Itoa(max(0, info.RemainingTokens)))
		w.Header().Set(headerRateLimitResetTokens, formatDuration(info.ResetTokens))
	}
}

// writeRateLimitError writes a 429 response with Retry-After and OpenAI-compatible JSON body.
func writeRateLimitError(w http.ResponseWriter, info Info) {
	retryAfterSecs := max(1, int(info.ResetRequests.Seconds()))
	w.Header().Set(headerRetryAfter, strconv.Itoa(retryAfterSecs))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	var msg string

	switch info.ExceededBy {
	case "requests":
		msg = fmt.Sprintf(
			"Rate limit reached: %d requests per minute. Please retry after %s.",
			info.LimitRequests, formatDuration(info.ResetRequests),
		)
	case "tokens":
		msg = fmt.Sprintf(
			"Rate limit reached: %d tokens per minute. Please retry after %s.",
			info.LimitTokens, formatDuration(info.ResetTokens),
		)
	default:
		msg = "Rate limit exceeded."
	}

	var body rateLimitError

	body.Error.Message = msg
	body.Error.Type = info.ExceededBy
	body.Error.Code = "rate_limit_exceeded"

	err := json.NewEncoder(w).Encode(body)
	if err != nil {
		// Response header already written; nothing useful we can do.
		_ = err
	}
}

// formatDuration formats a duration as a short human-readable string compatible
// with OpenAI's x-ratelimit-reset-* format (e.g. "30s", "1m0s", "200ms").
func formatDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)
	d = max(d, 0)

	return d.String()
}
