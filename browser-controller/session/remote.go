package session

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/remote"
)

// remoteAllocator uses the upstream WebSocket transport and keeps the connection
// alive until chromedp has closed its targets on cancellation.
type remoteAllocator struct {
	endpoint string
	wg       sync.WaitGroup
}

func newRemoteAllocator(parent context.Context, endpoint string) (context.Context, context.CancelFunc) {
	return chromedp.NewAllocatorContext(parent, &remoteAllocator{endpoint: endpoint})
}

func (a *remoteAllocator) Allocate(ctx context.Context, opts ...chromedp.BrowserOption) (*chromedp.Browser, error) {
	dialCtx, cancelDial := context.WithTimeout(ctx, 10*time.Second)
	defer cancelDial()
	conn, err := remote.DialContext(dialCtx, a.endpoint)
	if err != nil {
		return nil, fmt.Errorf("connect to remote browser: %w", err)
	}

	// Keep the connection alive while chromedp closes the targets on cancellation.
	connectionCtx, cancelConnection := context.WithCancel(context.Background())
	b, err := chromedp.NewBrowserTransport(connectionCtx, conn, opts...)
	if err != nil {
		cancelConnection()
		return nil, errors.Join(err, conn.Close())
	}
	a.wg.Go(func() {
		<-ctx.Done()
		_ = chromedp.Cancel(ctx)
		cancelConnection()
	})
	return b, nil
}

func (a *remoteAllocator) Wait()          { a.wg.Wait() }
func (a *remoteAllocator) Attaches() bool { return true }

// listenTarget registers an ordered event handler after initializing the target.
// Handlers run in the target's message loop and must dispatch protocol calls to
// a goroutine, so they cannot block responses needed to complete those calls.
func listenTarget(ctx context.Context, handle func(any)) error {
	return chromedp.Listen(ctx, handle)
}
