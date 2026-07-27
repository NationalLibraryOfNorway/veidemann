package session

import (
	"strings"
	"testing"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	bcerrors "github.com/NationalLibraryOfNorway/veidemann/browser-controller/errors"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/syncx"
)

func TestClassifyFrameWaitErrorReturnsTimeoutWithInitialCrawlLog(t *testing.T) {
	initialRequest := &requests.Request{CrawlLog: &logV1.CrawlLog{}}

	err, returnNow := classifyFrameWaitError(initialRequest, "https://example.com", syncx.ErrExceededMaxTime)
	if returnNow {
		t.Fatal("frame timeout should not short-circuit browser-side finalization")
	}
	if err == nil {
		t.Fatal("frame timeout should return a fetch error")
	}

	commonsErr := bcerrors.CommonsError(err)
	if commonsErr.Code != -5004 {
		t.Fatalf("unexpected error code: got %d want -5004", commonsErr.Code)
	}
	if !strings.Contains(commonsErr.Detail, "frames to finish loading") {
		t.Fatalf("unexpected error detail: %q", commonsErr.Detail)
	}
}

func TestClassifyFrameWaitErrorReturnsCacheHitImmediately(t *testing.T) {
	initialRequest := &requests.Request{FromCache: true}

	err, returnNow := classifyFrameWaitError(initialRequest, "https://example.com", syncx.ErrCancelled)
	if !returnNow {
		t.Fatal("cache hit cancellation should return immediately")
	}
	if err == nil {
		t.Fatal("cache hit cancellation should return an error")
	}

	commonsErr := bcerrors.CommonsError(err)
	if commonsErr.Code != -4100 {
		t.Fatalf("unexpected error code: got %d want -4100", commonsErr.Code)
	}
}

func TestClassifyCompletionWaitErrorReturnsTimeout(t *testing.T) {
	tests := []struct {
		name    string
		waitErr error
	}{
		{name: "idle timeout", waitErr: syncx.ErrIdleTimeout},
		{name: "max time", waitErr: syncx.ErrExceededMaxTime},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := classifyCompletionWaitError("https://example.com", tc.waitErr)
			if err == nil {
				t.Fatal("completion timeout should return a fetch error")
			}

			commonsErr := bcerrors.CommonsError(err)
			if commonsErr.Code != -5004 {
				t.Fatalf("unexpected error code: got %d want -5004", commonsErr.Code)
			}
			if !strings.Contains(commonsErr.Detail, "outstanding requests to complete") {
				t.Fatalf("unexpected error detail: %q", commonsErr.Detail)
			}
		})
	}
}

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
