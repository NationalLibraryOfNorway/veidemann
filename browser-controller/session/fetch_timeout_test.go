package session

import (
	"testing"
	"time"
)

func TestNetworkSettleIdleTime(t *testing.T) {
	tests := []struct {
		name        string
		maxIdleTime time.Duration
		want        time.Duration
	}{
		{name: "uses configured idle time when larger", maxIdleTime: 2 * time.Second, want: 2 * time.Second},
		{name: "keeps one second minimum", maxIdleTime: 500 * time.Millisecond, want: time.Second},
		{name: "falls back to one second when unset", maxIdleTime: 0, want: time.Second},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := networkSettleIdleTime(tc.maxIdleTime); got != tc.want {
				t.Fatalf("networkSettleIdleTime(%v) = %v, want %v", tc.maxIdleTime, got, tc.want)
			}
		})
	}
}

func TestDurationFromMilliseconds(t *testing.T) {
	if got := durationFromMilliseconds(3_000); got != 3*time.Second {
		t.Fatalf("durationFromMilliseconds(3000) = %v, want %v", got, 3*time.Second)
	}
}
