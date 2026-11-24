package session

import (
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"testing"
)

func TestEqualRequestIDsInDifferentTargetsRemainIndependent(t *testing.T) {
	root, cancelRoot := chromedp.NewContext(t.Context())
	defer cancelRoot()
	child, cancelChild := chromedp.NewContext(t.Context())
	defer cancelChild()
	chromedp.FromContext(root).Target = &chromedp.Target{SessionID: "root"}
	chromedp.FromContext(child).Target = &chromedp.Target{SessionID: "child"}
	sess := newDefaultSession()
	request := &network.EventRequestWillBeSent{RequestID: "same-id", FrameID: "frame", Type: network.ResourceTypeImage, Request: &network.Request{URL: "https://example.com/image"}}
	sess.onNetworkEventRequestWillBeSent(root, request, 1)
	sess.onNetworkEventRequestWillBeSent(child, request, 2)
	sess.onNetworkEventLoadingFinished(root, &network.EventLoadingFinished{RequestID: "same-id"}, 1)
	if got := sess.networkTracker.snapshot().requiredPending; got != 1 {
		t.Fatalf("child request lost: pending=%d", got)
	}
	sess.onNetworkEventLoadingFinished(child, &network.EventLoadingFinished{RequestID: "same-id"}, 2)
	if got := sess.networkTracker.snapshot().requiredPending; got != 0 {
		t.Fatalf("child request retained: pending=%d", got)
	}
}
