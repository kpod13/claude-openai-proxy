package ratelimit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLimiter_Enabled(t *testing.T) {
	require.False(t, New(0, 0).Enabled())
	require.True(t, New(60, 0).Enabled())
	require.True(t, New(0, 1000).Enabled())
	require.True(t, New(60, 1000).Enabled())
}

func TestLimiter_RPM_Enforced(t *testing.T) {
	l := New(3, 0)

	for i := range 3 {
		info, ok := l.Allow("k", 10)
		require.True(t, ok, "request %d should be allowed", i+1)
		require.Equal(t, 3, info.LimitRequests)
		require.Equal(t, 3-i-1, info.RemainingRequests)
	}

	info, ok := l.Allow("k", 10)
	require.False(t, ok)
	require.Equal(t, "requests", info.ExceededBy)
	require.Equal(t, 0, info.RemainingRequests)
}

func TestLimiter_TPM_Enforced(t *testing.T) {
	l := New(0, 100)

	info, ok := l.Allow("k", 80)
	require.True(t, ok)
	require.Equal(t, 20, info.RemainingTokens)

	// 80+30 > 100 → denied
	info, ok = l.Allow("k", 30)
	require.False(t, ok)
	require.Equal(t, "tokens", info.ExceededBy)
	require.Equal(t, 0, info.RemainingTokens)
}

func TestLimiter_PerKeyIsolation(t *testing.T) {
	l := New(1, 0)

	_, ok := l.Allow("key-a", 0)
	require.True(t, ok)

	// key-a exhausted
	_, ok = l.Allow("key-a", 0)
	require.False(t, ok)

	// key-b unaffected
	_, ok = l.Allow("key-b", 0)
	require.True(t, ok)
}

func TestLimiter_AnonymousBucket(t *testing.T) {
	l := New(1, 0)

	_, ok := l.Allow("", 0)
	require.True(t, ok)

	_, ok = l.Allow("", 0)
	require.False(t, ok)
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

func TestLimiter_UnlimitedDimension_ReturnsNegativeOne(t *testing.T) {
	// RPM unlimited, TPM configured.
	l := New(0, 100)

	info, ok := l.Allow("k", 10)
	require.True(t, ok)
	require.Equal(t, -1, info.RemainingRequests, "unlimited dimension should be -1")
	require.Equal(t, 90, info.RemainingTokens)
}

func TestLimiter_ResetDuration_WithinOneMinute(t *testing.T) {
	l := New(10, 0)

	info, ok := l.Allow("k", 0)
	require.True(t, ok)
	require.Positive(t, info.ResetRequests)
	require.LessOrEqual(t, info.ResetRequests, time.Minute)
}
