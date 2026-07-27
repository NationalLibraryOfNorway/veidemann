package screenshotwriter

import "testing"

func TestScreenshotTargetURI(t *testing.T) {
	requestedURI := "https://foo.bar/path?q=1"
	got := screenshotTargetURI(requestedURI)
	want := "screenshot:https://foo.bar/path?q=1"

	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
