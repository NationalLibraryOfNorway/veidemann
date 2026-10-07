package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/remote"
)

// remoteAllocator uses the upstream WebSocket transport and observes events in
// wire order. Independent per-method iterators would let loadingFinished run
// before requestWillBeSent, or frameStoppedLoading before frameStartedLoading.
type remoteAllocator struct {
	endpoint string
	events   targetEvents
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
	b, err := chromedp.NewBrowserTransport(connectionCtx, &eventTransport{Transport: conn, events: &a.events}, opts...)
	if err != nil {
		cancelConnection()
		conn.Close()
		return nil, err
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

type targetListener struct {
	ctx    context.Context
	handle func(any)
}

type targetEvents struct {
	mu        sync.RWMutex
	listeners map[target.SessionID]*targetListener
}

// listenTarget initializes the target before registering its event handler.
// Handlers run on the transport reader and must dispatch protocol calls to a
// goroutine, so they cannot block responses needed to complete those calls.
func listenTarget(ctx context.Context, handle func(any)) error {
	if err := chromedp.Do(ctx); err != nil {
		return err
	}
	c := chromedp.FromContext(ctx)
	a, ok := c.Allocator.(*remoteAllocator)
	if !ok {
		return fmt.Errorf("target requires a remote browser allocator")
	}
	a.events.listen(ctx, c.Target.SessionID, handle)
	return nil
}

func (e *targetEvents) listen(ctx context.Context, id target.SessionID, handle func(any)) {
	l := &targetListener{ctx: ctx, handle: handle}
	e.mu.Lock()
	if e.listeners == nil {
		e.listeners = make(map[target.SessionID]*targetListener)
	}
	e.listeners[id] = l
	e.mu.Unlock()
	context.AfterFunc(ctx, func() {
		e.mu.Lock()
		if e.listeners[id] == l {
			delete(e.listeners, id)
		}
		e.mu.Unlock()
	})
}

func (e *targetEvents) dispatch(msg *cdproto.Message) {
	if msg.ID != 0 || msg.SessionID == "" {
		return
	}
	e.mu.RLock()
	l := e.listeners[msg.SessionID]
	e.mu.RUnlock()
	if l == nil || l.ctx.Err() != nil {
		return
	}
	// Only domains consumed by session listeners need a second decoding pass.
	switch msg.Method.Domain() {
	case "Network", "Page", "Fetch", "Target":
	default:
		return
	}
	ev, err := cdproto.UnmarshalMessage(msg, chromedp.DefaultUnmarshalOptions)
	if err != nil {
		slog.Debug("Could not decode remote browser event", "method", msg.Method, "error", err)
		return
	}
	l.handle(ev)
}

type eventTransport struct {
	chromedp.Transport
	events *targetEvents
}

func (t *eventTransport) Read(ctx context.Context, msg *cdproto.Message) error {
	if err := t.Transport.Read(ctx, msg); err != nil {
		return err
	}
	t.events.dispatch(msg)
	return nil
}

func (t *eventTransport) SetDebugf(f func(string, ...any)) {
	if transport, ok := t.Transport.(interface{ SetDebugf(func(string, ...any)) }); ok {
		transport.SetDebugf(f)
	}
}
