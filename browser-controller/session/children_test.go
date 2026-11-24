package session

import (
	"context"
	"errors"
	"io"
	"reflect"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/chromedp/cdproto"
	"github.com/chromedp/cdproto/cdp"
	jsonv2 "github.com/chromedp/cdproto/cdp/jsonv2"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

type mockWire struct {
	commands chan cdproto.Message
	incoming chan cdproto.Message
	lost     chan struct{}
	once     sync.Once
}

func newMockWire() *mockWire {
	return &mockWire{commands: make(chan cdproto.Message, 100), incoming: make(chan cdproto.Message, 100), lost: make(chan struct{})}
}
func (w *mockWire) Write(ctx context.Context, msg *cdproto.Message) error {
	select {
	case w.commands <- *msg:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-w.lost:
		return io.EOF
	}
}
func (w *mockWire) Read(ctx context.Context, msg *cdproto.Message) error {
	select {
	case incoming := <-w.incoming:
		raw, err := jsonv2.Marshal(struct {
			ID        int64              `json:"id,omitempty"`
			SessionID target.SessionID   `json:"sessionId,omitempty"`
			Method    cdproto.MethodType `json:"method,omitempty"`
			Params    jsonv2.Value       `json:"params,omitzero"`
			Result    jsonv2.Value       `json:"result,omitzero"`
			Error     *cdproto.Error     `json:"error,omitzero"`
		}{incoming.ID, incoming.SessionID, incoming.Method, incoming.Params, incoming.Result, incoming.Error})
		if err != nil {
			return err
		}
		return jsonv2.Unmarshal(raw, msg, chromedp.DefaultUnmarshalOptions)
	case <-ctx.Done():
		return ctx.Err()
	case <-w.lost:
		return io.EOF
	}
}
func (w *mockWire) Close() error { w.once.Do(func() { close(w.lost) }); return nil }
func (w *mockWire) reply(cmd cdproto.Message) {
	w.incoming <- cdproto.Message{ID: cmd.ID, SessionID: cmd.SessionID, Result: jsonv2.Value(`{}`)}
}

func childHarness(t *testing.T) (*Session, context.Context, *mockWire, *eventTransport) {
	t.Helper()
	ctx, cancel := context.WithCancelCause(t.Context())
	wire := newMockWire()
	allocator, _ := newRemoteAllocator(ctx, "unused")
	root, _ := chromedp.NewContext(allocator)
	a := chromedp.FromContext(root).Allocator.(*remoteAllocator)
	transport := &eventTransport{Transport: wire, events: &a.events}
	a.transport = transport
	sess := newDefaultSession()
	sess.attemptID = "test-attempt"
	sess.browserCancel = cancel
	root = context.WithValue(root, childContextKey{}, &childSession{id: "root"})
	go func() {
		for {
			var msg cdproto.Message
			if transport.Read(ctx, &msg) != nil {
				return
			}
		}
	}()
	t.Cleanup(func() {
		_ = sess.stopChildren()
		cancel(nil)
		_ = wire.Close()
	})
	return sess, root, wire, transport
}
func attached(id target.SessionID, kind string) *target.EventAttachedToTarget {
	return &target.EventAttachedToTarget{SessionID: id, TargetInfo: &target.Info{TargetID: "same-target", Type: kind}, WaitingForDebugger: true}
}

func TestChildLifecycleConfigurationAndReattachment(t *testing.T) {
	for _, kind := range []string{"iframe", "worker"} {
		t.Run(kind, func(t *testing.T) {
			sess, root, wire, transport := childHarness(t)
			// Duplicate notifications do not configure or resume twice. No execution
			// context exists in this backend; every unexpected command fails the test.
			sess.onTargetAttached(root, attached("child", kind), 1)
			sess.onTargetAttached(root, attached("child", kind), 1)
			done := make(chan struct{})
			var methods []string
			go func() {
				defer close(done)
				for {
					cmd := <-wire.commands
					methods = append(methods, string(cmd.Method))
					transport.events.mu.RLock()
					listener := transport.events.listeners[cmd.SessionID]
					transport.events.mu.RUnlock()
					if listener == nil {
						t.Error("configuration preceded listener registration")
					}
					switch cmd.Method {
					case "Network.setExtraHTTPHeaders":
						var params network.SetExtraHTTPHeadersParams
						if err := jsonv2.Unmarshal(cmd.Params, &params); err != nil || params.Headers[constants.HeaderAttemptId] != sess.attemptID {
							t.Errorf("missing attempt identity: %s (%v)", cmd.Params, err)
						}
					case "Target.setAutoAttach":
						var params target.SetAutoAttachParams
						if err := jsonv2.Unmarshal(cmd.Params, &params); err != nil || !params.AutoAttach || !params.WaitForDebuggerOnStart || params.Flatten == nil || !*params.Flatten {
							t.Errorf("recursive attachment is not paused and flat: %s (%v)", cmd.Params, err)
						}
					}
					wire.reply(cmd)
					if cmd.Method == "Runtime.runIfWaitingForDebugger" {
						return
					}
				}
			}()
			if err := sess.drainChildren(t.Context(), false); err != nil {
				t.Fatal(err)
			}
			<-done
			want := []string{"Network.enable", "Network.setCacheDisabled", "Network.setExtraHTTPHeaders", "Target.setAutoAttach"}
			if kind == "iframe" {
				want = append(want, "Page.enable", "Page.setLifecycleEventsEnabled")
			}
			want = append(want, "Runtime.runIfWaitingForDebugger")
			if !reflect.DeepEqual(methods, want) {
				t.Fatalf("commands=%v", methods)
			}
			sess.childrenMu.Lock()
			child := sess.children["child"]
			sess.childrenMu.Unlock()
			// A nested attachment is registered on its parent's supplied session.
			sess.onTargetAttached(child.ctx, attached("nested", "worker"), 2)
			cmd := <-wire.commands
			if cmd.SessionID != "nested" {
				t.Fatal(cmd)
			}
			sess.networkTracker.noteObservedRequestStart(network.ResourceTypeImage, "https://example.com/x", true, "child:req")
			sess.networkTracker.noteObservedRequestStart(network.ResourceTypeImage, "https://example.com/y", true, "nested:req")
			sess.detachChild("child") // includes a nested child in flight during setup
			if err := sess.drainChildren(t.Context(), false); err != nil {
				t.Fatal(err)
			}
			if sess.networkTracker.snapshot().requiredPending != 0 {
				t.Fatal("detached requests survived")
			}
			sess.childrenMu.Lock()
			n := len(sess.children)
			sess.childrenMu.Unlock()
			if n != 0 {
				t.Fatal("descendants survived")
			}
			// Same target, new session: it must initialize again.
			sess.onTargetAttached(root, attached("reattached", kind), 1)
			cmd = <-wire.commands
			if cmd.SessionID != "reattached" || cmd.Method != "Network.enable" {
				t.Fatal(cmd)
			}
			sess.detachChild("reattached")
			if err := sess.drainChildren(t.Context(), false); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestChildLifecycleFailuresAndCancellation(t *testing.T) {
	for _, scenario := range []string{"command-error", "detach", "connection-loss", "cancel", "timeout", "unpaused"} {
		t.Run(scenario, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				sess, root, wire, transport := childHarness(t)
				ev := attached("child", "iframe")
				if scenario == "unpaused" {
					ev.WaitingForDebugger = false
				}
				sess.onTargetAttached(root, ev, 1)
				cmd := <-wire.commands
				if scenario == "timeout" {
					synctest.Wait()
					if sess.networkTracker.snapshot().requiredPending != 0 {
						t.Fatal("unexpected network request")
					}
					start := time.Now()
					err := sess.drainChildren(t.Context(), false)
					if time.Since(start) != childSetupTimeout || err == nil || !strings.Contains(err.Error(), "Network.enable") || !errors.Is(err, context.DeadlineExceeded) {
						t.Fatalf("timeout: %v after %v", err, time.Since(start))
					}
				} else {
					switch scenario {
					case "command-error":
						wire.incoming <- cdproto.Message{ID: cmd.ID, Error: &cdproto.Error{Code: -32000, Message: "original failure"}}
					case "connection-loss":
						_ = wire.Close()
					case "detach":
						sess.detachChild("child")
					case "cancel":
						sess.browserCancel(context.Canceled)
					case "unpaused":
						for {
							wire.reply(cmd)
							if cmd.Method == "Page.setLifecycleEventsEnabled" {
								break
							}
							cmd = <-wire.commands
							if cmd.Method == "Runtime.runIfWaitingForDebugger" {
								t.Fatal("resumed an unpaused session")
							}
						}
					}
					err := sess.drainChildren(t.Context(), false)
					fatal := scenario == "command-error" || scenario == "connection-loss"
					if (err != nil) != fatal {
						t.Fatalf("%s: %v", scenario, err)
					}
					if scenario == "command-error" {
						var protocol *cdproto.Error
						if !errors.As(err, &protocol) || protocol.Code != -32000 {
							t.Fatalf("lost original protocol error: %v", err)
						}
					}
					if scenario == "connection-loss" && !errors.Is(err, io.EOF) {
						t.Fatalf("lost connection error: %v", err)
					}
					if fatal && (!strings.Contains(err.Error(), "same-target session child type iframe") || !strings.Contains(err.Error(), "Network.enable")) {
						t.Fatal(err)
					}
				}
				_ = sess.stopChildren()
				transport.mu.Lock()
				pending := len(transport.pending)
				transport.mu.Unlock()
				if pending != 0 {
					t.Fatal("pending calls survived")
				}
			})
		})
	}
}

func TestSessionTransportConcurrentRepliesAndOrderedEvents(t *testing.T) {
	wire := newMockWire()
	var events targetEvents
	transport := &eventTransport{Transport: wire, events: &events}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	forwarded := make(chan cdproto.Message, 10)
	go func() {
		for {
			var msg cdproto.Message
			if transport.Read(ctx, &msg) != nil {
				return
			}
			forwarded <- msg
		}
	}()
	var order []network.RequestID
	events.listen(ctx, "child", func(ev any) { order = append(order, ev.(*network.EventLoadingFinished).RequestID) })
	root := &cdproto.Message{ID: 40, Method: cdproto.CommandBrowserGetVersion}
	if err := transport.Write(ctx, root); err != nil {
		t.Fatal(err)
	}
	first := <-wire.commands
	childDone := make(chan error, 1)
	go func() {
		var result cdp.Empty
		childDone <- transport.call(ctx, "child", "Network.enable", cdp.Empty{}, &result)
	}()
	second := <-wire.commands
	if first.ID == second.ID || root.ID != 40 {
		t.Fatal("wire ID collision or mutated caller ID")
	}
	if err := transport.Write(ctx, &cdproto.Message{ID: 1, SessionID: "root", Method: cdproto.CommandNetworkEnable}); err != nil {
		t.Fatal(err)
	}
	third := <-wire.commands
	if third.ID == first.ID || third.ID == second.ID {
		t.Fatal("shared allocator reused an ID")
	}
	// Interleave ordered events and reverse the replies, including the root's
	// browser-level reply, which has no session ID.
	for _, id := range []string{"one", "two", "three"} {
		wire.incoming <- cdproto.Message{SessionID: "child", Method: cdproto.EventNetworkLoadingFinished, Params: jsonv2.Value(`{"requestId":"` + id + `"}`)}
	}
	wire.reply(third)
	wire.reply(second)
	wire.reply(first)
	if err := <-childDone; err != nil {
		t.Fatal(err)
	}
	for range 3 {
		<-forwarded
	}
	pageReply := <-forwarded
	if pageReply.ID != 1 || pageReply.SessionID != "root" {
		t.Fatalf("root-page reply was not restored: %v", pageReply)
	}
	reply := <-forwarded
	if reply.ID != 40 || reply.SessionID != "" || !reflect.DeepEqual(order, []network.RequestID{"one", "two", "three"}) {
		t.Fatalf("reply=%v order=%v", reply, order)
	}
	_ = wire.Close()
}

type blockedWire struct {
	*mockWire
	release chan struct{}
}

func (w *blockedWire) Write(ctx context.Context, msg *cdproto.Message) error {
	<-w.release
	return w.mockWire.Write(ctx, msg)
}

func TestChildTransportCancellationWhileSocketWriteBlocks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wire := &blockedWire{mockWire: newMockWire(), release: make(chan struct{})}
		transport := &eventTransport{Transport: wire}
		ctx, cancel := context.WithTimeout(t.Context(), childSetupTimeout)
		defer cancel()
		var result cdp.Empty
		start := time.Now()
		err := transport.call(ctx, "child", "Network.enable", cdp.Empty{}, &result)
		if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) != childSetupTimeout {
			t.Fatalf("write blocked setup: %v", err)
		}
		transport.mu.Lock()
		pending := len(transport.pending)
		transport.mu.Unlock()
		if pending != 0 {
			t.Fatal("canceled write retained reply route")
		}
		_ = transport.Close()
		close(wire.release)
		synctest.Wait()
	})
}

func TestBrowserLevelParentDetachRemovesDescendants(t *testing.T) {
	sess, root, wire, transport := childHarness(t)
	transport.events.listen(root, "root", sess.listenFunc(root, 1))
	sess.onTargetAttached(root, attached("child", "iframe"), 1)
	<-wire.commands
	transport.events.dispatch(&cdproto.Message{Method: cdproto.EventTargetDetachedFromTarget, Params: jsonv2.Value(`{"sessionId":"root"}`)})
	if err := sess.drainChildren(t.Context(), false); err != nil {
		t.Fatal(err)
	}
	sess.childrenMu.Lock()
	n := len(sess.children)
	sess.childrenMu.Unlock()
	if n != 0 {
		t.Fatal("parent teardown retained descendants")
	}
}

type writeFailureWire struct{ *mockWire }

func (w *writeFailureWire) Write(ctx context.Context, msg *cdproto.Message) error {
	if msg.Method == cdproto.CommandBrowserGetVersion {
		return io.ErrClosedPipe
	}
	return w.mockWire.Write(ctx, msg)
}

func TestSessionTransportWriteFailureUnblocksEveryCaller(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	wire := &writeFailureWire{newMockWire()}
	transport := &eventTransport{Transport: wire}
	readDone := make(chan error, 1)
	go func() { var msg cdproto.Message; readDone <- transport.Read(ctx, &msg) }()
	childDone := make(chan error, 1)
	go func() {
		var result cdp.Empty
		childDone <- transport.call(ctx, "child", "Network.enable", cdp.Empty{}, &result)
	}()
	<-wire.commands // child awaits a reply when the browser-level write fails
	err := transport.Write(ctx, &cdproto.Message{ID: 50, Method: cdproto.CommandBrowserGetVersion})
	if !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("root lost write error: %v", err)
	}
	if err := <-childDone; !errors.Is(err, io.ErrClosedPipe) {
		t.Fatalf("child lost write error: %v", err)
	}
	if err := <-readDone; !errors.Is(err, io.EOF) {
		t.Fatalf("write failure did not unblock reader: %v", err)
	}
}
