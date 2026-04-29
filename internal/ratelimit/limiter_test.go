package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLimiter_Enabled(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		rpm  int
		tpm  int
		want bool
	}{
		{"both zero", 0, 0, false},
		{"RPM only", 60, 0, true},
		{"TPM only", 0, 1000, true},
		{"both set", 60, 1000, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tc.want, New(tc.rpm, tc.tpm).Enabled())
		})
	}
}

func TestLimiter_Enforced(t *testing.T) {
	t.Parallel()

	type call struct {
		tokens int
		wantOK bool
		check  func(t *testing.T, info Info)
	}

	cases := []struct {
		name     string
		rpm, tpm int
		key      string
		calls    []call
	}{
		{
			name: "RPM enforced",
			rpm:  3,
			key:  "k",
			calls: []call{
				{tokens: 10, wantOK: true, check: func(t *testing.T, info Info) {
					t.Helper()
					require.Equal(t, 3, info.LimitRequests)
					require.Equal(t, 2, info.RemainingRequests)
				}},
				{tokens: 10, wantOK: true, check: func(t *testing.T, info Info) {
					t.Helper()
					require.Equal(t, 1, info.RemainingRequests)
				}},
				{tokens: 10, wantOK: true, check: func(t *testing.T, info Info) {
					t.Helper()
					require.Equal(t, 0, info.RemainingRequests)
				}},
				{tokens: 10, wantOK: false, check: func(t *testing.T, info Info) {
					t.Helper()
					require.Equal(t, "requests", info.ExceededBy)
					require.Equal(t, 0, info.RemainingRequests)
				}},
			},
		},
		{
			name: "TPM enforced",
			tpm:  100,
			key:  "k",
			calls: []call{
				{tokens: 80, wantOK: true, check: func(t *testing.T, info Info) {
					t.Helper()
					require.Equal(t, 20, info.RemainingTokens)
				}},
				// 80+30 > 100 → denied
				{tokens: 30, wantOK: false, check: func(t *testing.T, info Info) {
					t.Helper()
					require.Equal(t, "tokens", info.ExceededBy)
					require.Equal(t, 0, info.RemainingTokens)
				}},
			},
		},
		{
			name: "unlimited dimension reports -1",
			rpm:  0,
			tpm:  100,
			key:  "k",
			calls: []call{
				{tokens: 10, wantOK: true, check: func(t *testing.T, info Info) {
					t.Helper()
					require.Equal(t, -1, info.RemainingRequests, "unlimited dimension should be -1")
					require.Equal(t, 90, info.RemainingTokens)
				}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := New(tc.rpm, tc.tpm)

			for i, c := range tc.calls {
				info, ok := l.Allow(tc.key, c.tokens)
				require.Equal(t, c.wantOK, ok, "call %d", i+1)

				if c.check != nil {
					c.check(t, info)
				}
			}
		})
	}
}

func TestLimiter_Buckets(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		// keys are consumed in order; the test asserts the second call to the
		// last key is denied while intermediate keys remain allowed.
		keys []string
	}{
		{"per-key isolation", []string{"key-a", "key-b", "key-a"}},
		{"anonymous bucket", []string{"", ""}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			l := New(1, 0)

			// First time we see each key it must succeed; the second time it must fail.
			seen := map[string]bool{}

			for i, k := range tc.keys {
				_, ok := l.Allow(k, 0)

				if seen[k] {
					require.False(t, ok, "call %d (key=%q) should be rejected on repeat", i+1, k)
				} else {
					require.True(t, ok, "call %d (key=%q) should be allowed first time", i+1, k)
					seen[k] = true
				}
			}
		})
	}
}

func TestLimiter_WindowReset(t *testing.T) {
	l := New(1, 0)

	// Manually place a window in the previous minute.
	past := time.Now().UTC().Truncate(time.Minute).Add(-time.Minute)
	l.windows["k"] = &window{minute: past, requests: 1}

	// First request in new minute should succeed.
	_, ok := l.Allow("k", 0)
	require.True(t, ok)
}

func TestLimiter_ResetDuration_WithinOneMinute(t *testing.T) {
	t.Parallel()

	l := New(10, 0)

	info, ok := l.Allow("k", 0)
	require.True(t, ok)
	require.Positive(t, info.ResetRequests)
	require.LessOrEqual(t, info.ResetRequests, time.Minute)
}
