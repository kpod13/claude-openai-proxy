package ratelimit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// echoHandler reads the request body and writes it back so we can confirm the body
// is still readable after the middleware restores it.
var (
	echoHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
)

func makeRequest(t *testing.T, body, bearerToken string) *http.Request {
	t.Helper()

	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(body),
	)

	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	}

	return req
}

func TestMiddleware_NoOp_WhenLimiterNil(t *testing.T) {
	handler := Middleware(nil)(echoHandler)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, makeRequest(t, `{"model":"x"}`, ""))

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Header().Get(headerRateLimitLimitRequests))
}

func TestMiddleware_NoOp_WhenLimitsZero(t *testing.T) {
	handler := Middleware(New(0, 0))(echoHandler)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, makeRequest(t, `{"model":"x"}`, ""))

	require.Equal(t, http.StatusOK, w.Code)
	require.Empty(t, w.Header().Get(headerRateLimitLimitRequests))
}

func TestMiddleware_HeadersPresent_WhenAllowed(t *testing.T) {
	handler := Middleware(New(10, 1000))(echoHandler)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, makeRequest(t, `{"model":"x","messages":[]}`, "token-abc"))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "10", w.Header().Get(headerRateLimitLimitRequests))
	require.Equal(t, "1000", w.Header().Get(headerRateLimitLimitTokens))
	require.NotEmpty(t, w.Header().Get(headerRateLimitRemainingRequests))
	require.NotEmpty(t, w.Header().Get(headerRateLimitRemainingTokens))
	require.NotEmpty(t, w.Header().Get(headerRateLimitResetRequests))
	require.NotEmpty(t, w.Header().Get(headerRateLimitResetTokens))
}

func TestMiddleware_OnlyRequestHeaders_WhenOnlyRPMConfigured(t *testing.T) {
	handler := Middleware(New(10, 0))(echoHandler)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, makeRequest(t, `{}`, ""))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "10", w.Header().Get(headerRateLimitLimitRequests))
	require.Empty(t, w.Header().Get(headerRateLimitLimitTokens), "token headers omitted when TPM not configured")
}

func TestMiddleware_Returns429_WhenRPMExceeded(t *testing.T) {
	handler := Middleware(New(1, 0))(echoHandler)

	// First request consumes the quota.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, makeRequest(t, `{}`, "k"))
	require.Equal(t, http.StatusOK, w.Code)

	// Second request should be rejected.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, makeRequest(t, `{}`, "k"))
	require.Equal(t, http.StatusTooManyRequests, w.Code)

	retryAfter := w.Header().Get(headerRetryAfter)
	require.NotEmpty(t, retryAfter)
	require.NotEqual(t, "0", retryAfter)

	var errBody rateLimitError

	err := json.NewDecoder(w.Body).Decode(&errBody)
	require.NoError(t, err)
	require.Equal(t, "requests", errBody.Error.Type)
	require.Equal(t, "rate_limit_exceeded", errBody.Error.Code)
	require.Nil(t, errBody.Error.Param)
	require.NotEmpty(t, errBody.Error.Message)
}

func TestMiddleware_Returns429_WhenTPMExceeded(t *testing.T) {
	// Use a tiny TPM limit so a single non-trivial body exhausts it.
	// Body of 8 bytes → estimateTokens returns 2; limit of 1 triggers 429 on first call.
	handler := Middleware(New(0, 1))(echoHandler)

	// Body estimateTokens: len(`{"a":"b"}`) = 9 bytes → 9/4 = 2 tokens > limit 1 → 429.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, makeRequest(t, `{"a":"b"}`, "k"))
	require.Equal(t, http.StatusTooManyRequests, w.Code)

	var errBody rateLimitError

	err := json.NewDecoder(w.Body).Decode(&errBody)
	require.NoError(t, err)
	require.Equal(t, "tokens", errBody.Error.Type)
	require.Equal(t, "rate_limit_exceeded", errBody.Error.Code)
}

func TestMiddleware_BodyRestoredForInnerHandler(t *testing.T) {
	handler := Middleware(New(10, 0))(echoHandler)

	body := `{"model":"sonnet","messages":[]}`
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, makeRequest(t, body, ""))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, body, w.Body.String())
}

func TestMiddleware_BearerToken_Extracted(t *testing.T) {
	// key-a and key-b should have independent counters.
	handler := Middleware(New(1, 0))(echoHandler)

	wA := httptest.NewRecorder()
	handler.ServeHTTP(wA, makeRequest(t, `{}`, "key-a"))
	require.Equal(t, http.StatusOK, wA.Code)

	// key-a exhausted — key-b still fine.
	wB := httptest.NewRecorder()
	handler.ServeHTTP(wB, makeRequest(t, `{}`, "key-b"))
	require.Equal(t, http.StatusOK, wB.Code)

	// key-a second request rejected.
	wA2 := httptest.NewRecorder()
	handler.ServeHTTP(wA2, makeRequest(t, `{}`, "key-a"))
	require.Equal(t, http.StatusTooManyRequests, wA2.Code)
}
