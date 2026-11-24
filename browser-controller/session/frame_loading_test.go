package session

import (
	"context"
	"errors"
	"testing"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/page"
)

func TestPageLoadAcceptsEarlyAndLateRootLoad(t *testing.T) {
	for _, early := range []bool{true, false} {
		sess := newDefaultSession()
		sess.pageLoad.start("root")
		event := &page.EventLifecycleEvent{FrameID: "root", LoaderID: "navigation", Name: "load"}
		if early {
			sess.pageLoad.observe(event)
		}
		sess.pageLoad.selectNavigation("root", "navigation")
		if !early {
			sess.pageLoad.observe(event)
		}
		if err := sess.waitForPageLoad(t.Context()); err != nil {
			t.Fatal(err)
		}
		sess.pageLoad.observe(event) // repeated delivery must not close twice
	}
}

func TestPageLoadIgnoresOtherFramesAndOldDocuments(t *testing.T) {
	sess := newDefaultSession()
	sess.pageLoad.start("root")
	sess.pageLoad.selectNavigation("root", "current")
	for _, event := range []*page.EventLifecycleEvent{
		{FrameID: "child", LoaderID: "current", Name: "load"},
		{FrameID: "root", LoaderID: "previous", Name: "load"},
		{FrameID: "root", LoaderID: "current", Name: "DOMContentLoaded"},
	} {
		sess.pageLoad.observe(event)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := sess.waitForPageLoad(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestPageLoadFollowsClientRedirectBeforeInitialLoad(t *testing.T) {
	sess := newDefaultSession()
	sess.pageLoad.start("root")
	sess.pageLoad.navigated(&cdp.Frame{ID: "root", LoaderID: "first"})
	sess.pageLoad.navigated(&cdp.Frame{ID: "root", LoaderID: "redirected"})
	sess.pageLoad.selectNavigation("root", "first")
	sess.pageLoad.observe(&page.EventLifecycleEvent{FrameID: "root", LoaderID: "first", Name: "load"})
	select {
	case <-sess.pageLoad.done:
		t.Fatal("old document completed navigation")
	default:
	}
	sess.pageLoad.observe(&page.EventLifecycleEvent{FrameID: "root", LoaderID: "redirected", Name: "load"})
	if err := sess.waitForPageLoad(t.Context()); err != nil {
		t.Fatal(err)
	}
}

func TestCommittedRootDocumentIdentity(t *testing.T) {
	sess := newDefaultSession()
	sess.pageLoad.start("root")
	listener := sess.listenFunc(t.Context(), 1)
	frame := &cdp.Frame{ID: "root", LoaderID: "first", URL: "https://example.com/index"}
	if _, ok := sess.pageLoad.documentURL(frame); ok {
		t.Fatal("unobserved document accepted")
	}
	listener(&page.EventFrameNavigated{Frame: frame})
	for _, url := range []string{"https://example.com/path", "https://example.com/index?q=1", "https://example.com/index#fragment"} {
		listener(&page.EventNavigatedWithinDocument{FrameID: "root", URL: url})
		current := *frame
		current.URL = url
		if got, ok := sess.pageLoad.documentURL(&current); !ok || got != frame.URL {
			t.Fatalf("history changed commit: %q %v", got, ok)
		}
	}
	listener(&page.EventFrameNavigated{Frame: &cdp.Frame{ID: "child", ParentID: "root", LoaderID: "child", URL: "https://example.com/child"}})
	if got, ok := sess.pageLoad.documentURL(frame); !ok || got != frame.URL {
		t.Fatal("child replaced root")
	}
	for _, next := range []*cdp.Frame{
		{ID: "root", LoaderID: "redirect", URL: "https://example.com/final"},
		{ID: "root", LoaderID: "reload", URL: "https://example.com/final"},
	} {
		listener(&page.EventFrameNavigated{Frame: next})
		if _, ok := sess.pageLoad.documentURL(frame); ok {
			t.Fatal("old loader accepted")
		}
		if got, ok := sess.pageLoad.documentURL(next); !ok || got != next.URL {
			t.Fatal("new commit missing")
		}
		frame = next
	}
	for _, bad := range []*cdp.Frame{{ID: "root"}, {ID: "other", LoaderID: frame.LoaderID}, {ID: "root", LoaderID: "unknown"}} {
		if _, ok := sess.pageLoad.documentURL(bad); ok {
			t.Fatalf("invalid identity accepted: %v", bad)
		}
	}
	listener(&page.EventFrameNavigated{Frame: &cdp.Frame{ID: "root", URL: "https://example.com/no-loader"}})
	sess.pageLoad.selectNavigation("root", frame.LoaderID)
	if _, ok := sess.pageLoad.documentURL(frame); ok {
		t.Fatal("missing loader retained old commit")
	}
}
