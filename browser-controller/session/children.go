package session

import (
	"context"
	"fmt"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
)

const childSetupTimeout = 10 * time.Second

type childSession struct {
	id, parent target.SessionID
	info       target.Info
	paused     bool
	ctx        context.Context
	cancel     context.CancelFunc
	transport  *eventTransport
	detached   bool // guarded by childrenMu
}
type childContextKey struct{}

func sessionIDFromContext(ctx context.Context) target.SessionID {
	if child, ok := ctx.Value(childContextKey{}).(*childSession); ok {
		return child.id
	}
	if c := chromedp.FromContext(ctx); c != nil && c.Target != nil {
		return c.Target.SessionID
	}
	return ""
}

func callTarget[P, R any](ctx context.Context, command cdp.Command[P, R], params P) (R, error) {
	if child, ok := ctx.Value(childContextKey{}).(*childSession); ok {
		var result R
		err := child.transport.call(ctx, child.id, command.Method, params, &result)
		if err != nil {
			err = fmt.Errorf("%s: %w", command.Method, err)
		}
		return result, err
	}
	return chromedp.Call(ctx, command, params)
}

// Runs on the ordered reader. Register before any configuration can emit events.
func (sess *Session) onTargetAttached(ctx context.Context, ev *target.EventAttachedToTarget, _ int64) {
	if ev.TargetInfo == nil || ev.SessionID == "" {
		return
	}
	allocator := chromedp.FromContext(ctx).Allocator.(*remoteAllocator)
	sess.childrenMu.Lock()
	if sess.childrenSealed || sess.children[ev.SessionID] != nil || ev.SessionID == sessionIDFromContext(ctx) {
		sess.childrenMu.Unlock()
		return
	}
	childCtx, cancel := context.WithCancel(ctx)
	child := &childSession{id: ev.SessionID, parent: sessionIDFromContext(ctx), info: *ev.TargetInfo, paused: ev.WaitingForDebugger, cancel: cancel, transport: allocator.transport}
	child.ctx = context.WithValue(childCtx, childContextKey{}, child)
	sess.children[child.id] = child
	sess.childSetups++
	allocator.events.listen(child.ctx, child.id, sess.listenFunc(child.ctx, sess.listenerSeq.Add(1)))
	sess.childrenMu.Unlock()
	sess.diagnostics.mu.Lock()
	sess.diagnostics.children[child.info.Type]++
	sess.diagnostics.mu.Unlock()
	sess.logger.Log(context.Background(), LevelTrace, "Child session attached", "targetId", child.info.TargetID, "cdpSessionId", child.id, "parentCdpSessionId", child.parent, "targetType", child.info.Type, "waitingForDebugger", child.paused)
	sess.networkTracker.notify()
	go sess.setupChild(child)
}

func childStep[P, R any](ctx context.Context, command cdp.Command[P, R], params P) error {
	_, err := callTarget(ctx, command, params)
	return err
}

func (sess *Session) configureChild(ctx context.Context, child *childSession) error {
	if err := childStep(ctx, network.Enable, network.EnableParams{}); err != nil {
		return err
	}
	if err := childStep(ctx, network.SetCacheDisabled, network.SetCacheDisabledParams{CacheDisabled: true}); err != nil {
		return err
	}
	if err := childStep(ctx, network.SetExtraHTTPHeaders, network.SetExtraHTTPHeadersParams{Headers: network.Headers{constants.HeaderAttemptId: sess.attemptID}}); err != nil {
		return err
	}
	if err := childStep(ctx, target.SetAutoAttach, harvestAutoAttach()); err != nil {
		return err
	}
	if child.info.Type == "iframe" || child.info.Type == "page" {
		if err := childStep(ctx, page.Enable, page.EnableParams{}); err != nil {
			return err
		}
		if err := childStep(ctx, page.SetLifecycleEventsEnabled, page.SetLifecycleEventsEnabledParams{Enabled: true}); err != nil {
			return err
		}
	}
	if child.paused {
		return childStep(ctx, runtime.RunIfWaitingForDebugger, cdp.Empty{})
	}
	return nil
}

func (sess *Session) setupChild(child *childSession) {
	ctx, cancel := context.WithTimeout(child.ctx, childSetupTimeout)
	defer cancel()
	err := sess.configureChild(ctx, child)
	if err == nil {
		sess.diagnostics.mu.Lock()
		sess.diagnostics.configured++
		if child.paused {
			sess.diagnostics.resumed++
		}
		sess.diagnostics.mu.Unlock()
		sess.logger.Log(context.Background(), LevelTrace, "Child session configured", "targetId", child.info.TargetID, "cdpSessionId", child.id, "parentCdpSessionId", child.parent, "targetType", child.info.Type, "paused", false, "resumed", child.paused)
	}
	sess.childrenMu.Lock()
	// A detached target and parent shutdown are normal. A live setup deadline or
	// command failure is fatal, and is published before the setup counter drains.
	if err != nil && !child.detached && child.ctx.Err() == nil {
		err = fmt.Errorf("configure child target %s session %s type %s: %w", child.info.TargetID, child.id, child.info.Type, err)
		if sess.childErr == nil {
			sess.childErr = err
			sess.failPhase("child_setup")
		}
		if sess.browserCancel != nil {
			sess.browserCancel(err)
		}
	}
	sess.childSetups--
	sess.childrenMu.Unlock()
	sess.networkTracker.notify()
	select {
	case sess.childUpdates <- struct{}{}:
	default:
	}
}

func (sess *Session) detachChild(id target.SessionID) {
	sess.childrenMu.Lock()
	detached := sess.detachChildLocked(id)
	sess.childrenMu.Unlock()
	sess.logChildDetachments(detached, "target_removed")
}

func (sess *Session) logChildDetachments(children []*childSession, reason string) {
	for _, child := range children {
		sess.logger.Log(context.Background(), LevelTrace, "Child session detached", "targetId", child.info.TargetID, "cdpSessionId", child.id, "parentCdpSessionId", child.parent, "targetType", child.info.Type, "reason", reason)
	}
}

func (sess *Session) detachChildLocked(id target.SessionID) (detached []*childSession) {
	for childID, child := range sess.children {
		if child.parent == id {
			detached = append(detached, sess.detachChildLocked(childID)...)
		}
	}
	if child := sess.children[id]; child != nil {
		detached = append(detached, child)
		child.detached = true
		child.cancel()
		child.transport.events.mu.Lock()
		delete(child.transport.events.listeners, id)
		child.transport.events.mu.Unlock()
		child.transport.detach(id)
		sess.networkTracker.removeSession(id)
		sess.forgetRequests(id)
		sess.diagnostics.mu.Lock()
		sess.diagnostics.detached++
		sess.diagnostics.mu.Unlock()
		delete(sess.children, id)
	}
	return detached
}

// Seal admission before the final drain so no setup can race recorder Finish.
func (sess *Session) drainChildren(ctx context.Context, seal bool) error {
	sess.childrenMu.Lock()
	if seal {
		sess.childrenSealed = true
	}
	for sess.childSetups != 0 {
		sess.childrenMu.Unlock()
		select {
		case <-ctx.Done():
			return fmt.Errorf("drain child setup: %w", ctx.Err())
		case <-sess.childUpdates:
		}
		sess.childrenMu.Lock()
	}
	err := sess.childErr
	sess.childrenMu.Unlock()
	return err
}

func (sess *Session) stopChildren() error {
	sess.childrenMu.Lock()
	sess.childrenSealed = true
	var detached []*childSession
	for id := range sess.children {
		detached = append(detached, sess.detachChildLocked(id)...)
	}
	sess.childrenMu.Unlock()
	sess.logChildDetachments(detached, "harvest_teardown")
	// Cancellation unblocks every child call; setup goroutines never await CDP
	// teardown. Wait for their failure publication before finalizing the recorder.
	return sess.drainChildren(context.Background(), true)
}
