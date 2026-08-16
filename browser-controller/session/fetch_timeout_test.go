package session

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	bcerrors "github.com/NationalLibraryOfNorway/veidemann/browser-controller/errors"
)

func TestClassifyFrameWaitErrorReturnsTimeoutWithInitialCrawlLog(t *testing.T) {
	err, returnNow := classifyFrameWaitError(false, "https://example.com", context.DeadlineExceeded)
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
	err, returnNow := classifyFrameWaitError(true, "https://example.com", errInitialRequestCached)
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
		{name: "idle timeout", waitErr: errCompletionIdleTimeout},
		{name: "load deadline", waitErr: context.DeadlineExceeded},
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

func TestClassifyFrameWaitErrorPreservesCancellation(t *testing.T) {
	err, returnNow := classifyFrameWaitError(false, "https://example.com", context.Canceled)
	if !returnNow {
		t.Fatal("context cancellation should stop fetch processing")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("frame wait error = %v, want %v", err, context.Canceled)
	}
}

func TestClassifyCompletionWaitErrorPreservesCancellation(t *testing.T) {
	err := classifyCompletionWaitError("https://example.com", context.Canceled)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("completion wait error = %v, want %v", err, context.Canceled)
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

func TestDurationFromMilliseconds(t *testing.T) {
	if got := durationFromMilliseconds(3_000); got != 3*time.Second {
		t.Fatalf("durationFromMilliseconds(3000) = %v, want %v", got, 3*time.Second)
	}
}

func TestScreenshotParentContext(t *testing.T) {
	browserCtx := context.WithValue(t.Context(), contextKey("context"), "browser")
	loadCtx, cancelLoad := context.WithCancel(context.WithValue(browserCtx, contextKey("context"), "load"))
	cancelLoad()

	if got := screenshotParentContext(browserCtx, loadCtx, nil); got != loadCtx {
		t.Fatal("successful fetch did not retain the load context for screenshot capture")
	}
	if got := screenshotParentContext(browserCtx, loadCtx, errors.New("fetch failed")); got != browserCtx {
		t.Fatal("failed fetch did not use the live browser context for recovery screenshot capture")
	}
	if err := screenshotParentContext(browserCtx, loadCtx, errors.New("fetch failed")).Err(); err != nil {
		t.Fatalf("recovery screenshot context is already expired: %v", err)
	}
}

type contextKey string
