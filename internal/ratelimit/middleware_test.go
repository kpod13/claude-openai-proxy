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

func TestMiddleware_NoOp(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		limiter *Limiter
	}{
		{name: "limiter nil", limiter: nil},
		{name: "limits zero", limiter: New(0, 0)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := Middleware(tc.limiter)(echoHandler)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, makeRequest(t, `{"model":"x"}`, ""))

			require.Equal(t, http.StatusOK, w.Code)
			require.Empty(t, w.Header().Get(headerRateLimitLimitRequests))
		})
	}
}

func TestMiddleware_Headers(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		rpm, tpm    int
		body        string
		bearerToken string
		check       func(t *testing.T, h http.Header)
	}{
		{
			name:        "all headers present when allowed",
			rpm:         10,
			tpm:         1000,
			body:        `{"model":"x","messages":[]}`,
			bearerToken: "token-abc",
			check: func(t *testing.T, h http.Header) {
				t.Helper()
				require.Equal(t, "10", h.Get(headerRateLimitLimitRequests))
				require.Equal(t, "1000", h.Get(headerRateLimitLimitTokens))
				require.NotEmpty(t, h.Get(headerRateLimitRemainingRequests))
				require.NotEmpty(t, h.Get(headerRateLimitRemainingTokens))
				require.NotEmpty(t, h.Get(headerRateLimitResetRequests))
				require.NotEmpty(t, h.Get(headerRateLimitResetTokens))
			},
		},
		{
			name: "only request headers when only RPM configured",
			rpm:  10,
			tpm:  0,
			body: `{}`,
			check: func(t *testing.T, h http.Header) {
				t.Helper()
				require.Equal(t, "10", h.Get(headerRateLimitLimitRequests))
				require.Empty(t, h.Get(headerRateLimitLimitTokens), "token headers omitted when TPM not configured")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := Middleware(New(tc.rpm, tc.tpm))(echoHandler)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, makeRequest(t, tc.body, tc.bearerToken))

			require.Equal(t, http.StatusOK, w.Code)
			tc.check(t, w.Header())
		})
	}
}

func TestMiddleware_Returns429(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		setup    func(t *testing.T) (http.Handler, *http.Request)
		wantType string
	}{
		{
			name: "RPM exceeded",
			setup: func(t *testing.T) (http.Handler, *http.Request) {
				t.Helper()

				h := Middleware(New(1, 0))(echoHandler)
				// First request consumes the quota.
				h.ServeHTTP(httptest.NewRecorder(), makeRequest(t, `{}`, "k-rpm"))

				return h, makeRequest(t, `{}`, "k-rpm")
			},
			wantType: "requests",
		},
		{
			name: "TPM exceeded",
			setup: func(t *testing.T) (http.Handler, *http.Request) {
				t.Helper()

				h := Middleware(New(0, 1))(echoHandler)
				// 9 bytes → 9/4 = 2 tokens > limit 1 → 429 on first call.
				return h, makeRequest(t, `{"a":"b"}`, "k-tpm")
			},
			wantType: "tokens",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler, req := tc.setup(t)

			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			require.Equal(t, http.StatusTooManyRequests, w.Code)

			retryAfter := w.Header().Get(headerRetryAfter)
			require.NotEmpty(t, retryAfter)
			require.NotEqual(t, "0", retryAfter)

			var errBody rateLimitError

			err := json.NewDecoder(w.Body).Decode(&errBody)
			require.NoError(t, err)
			require.Equal(t, tc.wantType, errBody.Error.Type)
			require.Equal(t, "rate_limit_exceeded", errBody.Error.Code)
			require.Nil(t, errBody.Error.Param)
			require.NotEmpty(t, errBody.Error.Message)
		})
	}
}

func TestMiddleware_BodyRestoredForInnerHandler(t *testing.T) {
	t.Parallel()

	handler := Middleware(New(10, 0))(echoHandler)

	body := `{"model":"sonnet","messages":[]}`
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, makeRequest(t, body, ""))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, body, w.Body.String())
}

func TestMiddleware_BearerToken_Extracted(t *testing.T) {
	t.Parallel()

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
