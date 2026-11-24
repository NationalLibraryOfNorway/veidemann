package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/chromedp/cdproto"
	jsonv2 "github.com/chromedp/cdproto/cdp/jsonv2"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/remote"
)

// remoteAllocator uses the upstream WebSocket transport and keeps the connection
// alive until chromedp has closed its targets on cancellation.
type remoteAllocator struct {
	endpoint  string
	events    targetEvents
	wg        sync.WaitGroup
	transport *eventTransport
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
	a.transport = &eventTransport{Transport: conn, events: &a.events}
	b, err := chromedp.NewBrowserTransport(connectionCtx, a.transport, opts...)
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

type harvestLogContextKey struct{}

type targetListener struct {
	logger *slog.Logger
	ctx    context.Context
	handle func(any)
}

type targetEvents struct {
	mu        sync.RWMutex
	listeners map[target.SessionID]*targetListener
}

func listenTarget(ctx context.Context, handle func(any)) error {
	return chromedp.Listen(ctx, handle)
}

// listenHarvestTarget initializes the target before registering its event handler.
// Handlers run on the transport reader and must dispatch protocol calls to a
// goroutine, so they cannot block responses needed to complete those calls.
func listenHarvestTarget(ctx context.Context, handle func(any)) error {
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
	log, ok := ctx.Value(harvestLogContextKey{}).(*slog.Logger)
	if !ok {
		log = slog.Default()
	}
	l := &targetListener{ctx: ctx, handle: handle, logger: log.With("cdpSessionId", id)}
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

	if msg.ID != 0 {
		return
	}
	listenerSession := msg.SessionID
	// Browser-level detachments name the affected session in the payload.
	// Route these to its handler too, so root teardown removes descendants.
	if listenerSession == "" && msg.Method == cdproto.EventTargetDetachedFromTarget {
		var detached target.EventDetachedFromTarget
		if err := jsonv2.Unmarshal(msg.Params, &detached); err != nil {
			return
		}
		listenerSession = detached.SessionID
	}
	if listenerSession == "" {
		return
	}
	e.mu.RLock()
	l := e.listeners[listenerSession]
	e.mu.RUnlock()
	if l == nil || l.ctx.Err() != nil {
		return
	}
	// Ignore unconsumed events before decoding, including events removed from
	// newer cdproto versions. Only malformed events we actually use are warnings.
	switch msg.Method {
	case cdproto.EventFetchRequestPaused, cdproto.EventNetworkRequestWillBeSent,
		cdproto.EventNetworkLoadingFinished, cdproto.EventNetworkDataReceived,
		cdproto.EventNetworkResponseReceived, cdproto.EventNetworkLoadingFailed,
		cdproto.EventPageLifecycleEvent, cdproto.EventPageFrameNavigated,
		cdproto.EventPageFileChooserOpened, cdproto.EventPageJavascriptDialogOpening,
		cdproto.EventTargetAttachedToTarget, cdproto.EventTargetDetachedFromTarget:
	default:
		return
	}
	ev, err := cdproto.UnmarshalMessage(msg, chromedp.DefaultUnmarshalOptions)
	if err != nil {
		l.logger.Warn("Malformed browser event", "method", msg.Method, "error", err)
		return
	}
	l.handle(ev)
}

type eventTransport struct {
	chromedp.Transport
	events  *targetEvents
	writes  chan wireWrite
	failed  chan struct{}
	mu      sync.Mutex
	nextID  int64
	pending map[int64]wireRoute
	err     error
}

type wireRoute struct {
	original int64
	reply    chan wireReply
	session  target.SessionID
}

type wireReply struct {
	message cdproto.Message
	err     error
}

var errChildDetached = errors.New("child session detached")

type wireWrite struct {
	ctx  context.Context
	msg  cdproto.Message
	done chan error
}

// All writers share one allocator, including chromedp's browser/root commands.
// The upstream WebSocket Write does not honor context cancellation. Keep it on
// one writer goroutine so child calls can still end at their setup deadline.
func (t *eventTransport) send(ctx context.Context, msg *cdproto.Message, route wireRoute) (id int64, err error) {
	t.mu.Lock()
	if t.err != nil {
		err = t.err
		t.mu.Unlock()
		return 0, err
	}
	if t.writes == nil {
		t.writes = make(chan wireWrite)
		t.failed = make(chan struct{})
		go t.writeLoop()
	}
	t.nextID++
	id = t.nextID
	if t.pending == nil {
		t.pending = make(map[int64]wireRoute)
	}
	t.pending[id] = route
	t.mu.Unlock()
	defer func() {
		if err != nil {
			t.mu.Lock()
			delete(t.pending, id)
			t.mu.Unlock()
		}
	}()
	wire := *msg
	wire.ID = id
	write := wireWrite{ctx: ctx, msg: wire, done: make(chan error, 1)}
	select {
	case t.writes <- write:
	case <-ctx.Done():
		return id, ctx.Err()
	case <-t.failed:
		return id, t.failure()
	}
	select {
	case err = <-write.done:
		return id, err
	case <-ctx.Done():
		return id, ctx.Err()
	case <-t.failed:
		return id, t.failure()
	}
}

func (t *eventTransport) failure() error { t.mu.Lock(); defer t.mu.Unlock(); return t.err }

func (t *eventTransport) writeLoop() {
	for {
		select {
		case <-t.failed:
			return
		case write := <-t.writes:
			if err := write.ctx.Err(); err != nil {
				write.done <- err
				continue
			}
			err := t.Transport.Write(context.WithoutCancel(write.ctx), &write.msg)
			write.done <- err
			if err != nil {
				t.fail(err)
				// Wake the reader and chromedp's root callers on write-only failures too.
				_ = t.Transport.Close()
				return
			}
		}
	}
}

func (t *eventTransport) Write(ctx context.Context, msg *cdproto.Message) error {
	_, err := t.send(ctx, msg, wireRoute{original: msg.ID, session: msg.SessionID})
	return err
}

func (t *eventTransport) fail(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.err == nil {
		t.err = err
		if t.failed != nil {
			close(t.failed)
		}
	}
	for id, route := range t.pending {
		if route.reply != nil {
			route.reply <- wireReply{err: err}
		}
		delete(t.pending, id)
	}
}

func (t *eventTransport) Close() error {
	t.fail(errors.New("browser connection closed"))
	return t.Transport.Close()
}

func (t *eventTransport) detach(id target.SessionID) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for wireID, route := range t.pending {
		if route.session == id && route.reply != nil {
			route.reply <- wireReply{err: errChildDetached}
			delete(t.pending, wireID)
		}
	}
}

func (t *eventTransport) call(ctx context.Context, session target.SessionID, method string, params, result any) error {
	raw, err := jsonv2.Marshal(params)
	if err != nil {
		return err
	}
	replies := make(chan wireReply, 1)
	id, err := t.send(ctx, &cdproto.Message{SessionID: session, Method: cdproto.MethodType(method), Params: raw}, wireRoute{reply: replies, session: session})
	if err != nil {
		return err
	}
	defer func() { t.mu.Lock(); delete(t.pending, id); t.mu.Unlock() }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case reply := <-replies:
		if reply.err != nil {
			return reply.err
		}
		msg := reply.message
		if msg.Error != nil {
			return msg.Error
		}
		return jsonv2.Unmarshal(msg.Result, result, chromedp.DefaultUnmarshalOptions)
	}
}

// Read is the single ordered event/reply reader. Child replies never enter
// chromedp's reply dispatcher, whose IDs are restored before forwarding.
func (t *eventTransport) Read(ctx context.Context, msg *cdproto.Message) error {
	for {
		// The upstream decoder merges into its destination. Each wire message
		// needs a fresh envelope when internal child replies are consumed here.
		*msg = cdproto.Message{}
		if err := t.Transport.Read(ctx, msg); err != nil {
			t.fail(err)
			return err
		}
		if msg.ID != 0 {
			t.mu.Lock()
			route, ok := t.pending[msg.ID]
			delete(t.pending, msg.ID)
			t.mu.Unlock()
			if !ok {
				continue
			} // A canceled child call's late reply.
			if route.reply != nil {
				route.reply <- wireReply{message: *msg}
				continue
			}
			msg.ID = route.original
		} else {
			t.events.dispatch(msg)
		}
		return nil
	}
}

func (t *eventTransport) SetDebugf(f func(string, ...any)) {
	if transport, ok := t.Transport.(interface{ SetDebugf(func(string, ...any)) }); ok {
		transport.SetDebugf(f)
	}
}
