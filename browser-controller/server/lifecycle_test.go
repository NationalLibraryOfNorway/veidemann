package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	browsercontrollerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	robotsevaluatorV1 "github.com/NationalLibraryOfNorway/veidemann/api/robotsevaluator/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/requests"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/testutil"
)

type mutableSessionLookup struct {
	mu       sync.RWMutex
	sessions map[int]*session.Session
}

func newMutableSessionLookup(sessions ...*session.Session) *mutableSessionLookup {
	lookup := &mutableSessionLookup{sessions: make(map[int]*session.Session, len(sessions))}
	for _, sess := range sessions {
		lookup.sessions[sess.Id] = sess
	}
	return lookup
}

func (l *mutableSessionLookup) GetActive(id int) *session.Session {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.sessions[id]
}

func (l *mutableSessionLookup) setActive(id int, sess *session.Session) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if sess == nil {
		delete(l.sessions, id)
		return
	}
	l.sessions[id] = sess
}

func serverSession(t *testing.T, id int, executionID string) *session.Session {
	t.Helper()
	return &session.Session{
		Id: id,
		RequestedUrl: &frontierV1.QueuedUri{
			ExecutionId:    executionID,
			JobExecutionId: "job-1",
		},
		CrawlConfig: &configV1.CrawlConfig{},
	}
}

func TestRegisterResourceCancelsReleasedSession(t *testing.T) {
	server := NewApiServer(session.NewRegistry(2), nil, nil)
	reply, err := server.RegisterResource(context.Background(), &browsercontrollerV2.RegisterResourceRequest{
		ProxyId:   1,
		RequestId: "network-1",
		Method:    "GET",
		Uri:       "https://example.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.GetCancel() != cancelledByBrowserController {
		t.Fatalf("cancel reason = %q, want %q", reply.GetCancel(), cancelledByBrowserController)
	}
}

func TestCompleteResourceAcceptsLateBrowserCancellationForReleasedSession(t *testing.T) {
	server := NewApiServer(session.NewRegistry(2), nil, nil)
	_, err := server.CompleteResource(context.Background(), &browsercontrollerV2.CompleteResourceRequest{
		ProxyId:   1,
		RequestId: "network-1",
		CrawlLog: &logV1.CrawlLog{
			StatusCode:   canceledByBrowserStatusCode,
			Method:       "GET",
			RequestedUri: "https://example.com/events",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestResourceRPCsTreatInitializingSessionAsUnavailable(t *testing.T) {
	registry := session.NewRegistry(2)
	initializing, err := registry.GetNextAvailable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Release(initializing)

	server := NewApiServer(registry, nil, nil)
	reply, err := server.RegisterResource(context.Background(), &browsercontrollerV2.RegisterResourceRequest{
		ProxyId:   int32(initializing.Id),
		RequestId: "network-1",
		Method:    http.MethodGet,
		Uri:       "https://example.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.GetCancel() != cancelledByBrowserController {
		t.Fatalf("cancel reason = %q, want %q", reply.GetCancel(), cancelledByBrowserController)
	}

	if _, err := server.CompleteResource(context.Background(), &browsercontrollerV2.CompleteResourceRequest{
		ProxyId:   int32(initializing.Id),
		RequestId: "network-1",
		CrawlLog:  &logV1.CrawlLog{StatusCode: 200},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestResourceRPCsTreatInvalidProxyIDsAsUnavailable(t *testing.T) {
	server := NewApiServer(session.NewRegistry(2), nil, nil)
	for _, proxyID := range []int32{-1, 2, 1000} {
		t.Run(fmt.Sprintf("proxy-%d", proxyID), func(t *testing.T) {
			reply, err := server.RegisterResource(context.Background(), &browsercontrollerV2.RegisterResourceRequest{
				ProxyId:   proxyID,
				RequestId: "network-1",
				Method:    http.MethodGet,
			})
			if err != nil {
				t.Fatal(err)
			}
			if reply.GetCancel() != cancelledByBrowserController {
				t.Fatalf("cancel reason = %q, want %q", reply.GetCancel(), cancelledByBrowserController)
			}

			if _, err := server.CompleteResource(context.Background(), &browsercontrollerV2.CompleteResourceRequest{
				ProxyId:  proxyID,
				CrawlLog: &logV1.CrawlLog{},
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRegisterResourceSupportsActiveConnectWithoutExecutionID(t *testing.T) {
	active := serverSession(t, 1, "active")
	server := NewApiServer(newMutableSessionLookup(active), nil, nil)

	reply, err := server.RegisterResource(context.Background(), &browsercontrollerV2.RegisterResourceRequest{
		ProxyId: int32(active.Id),
		Method:  http.MethodConnect,
		Uri:     "example.com:443",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.GetRegistered().GetCrawlExecutionId() != "active" {
		t.Fatalf("execution ID = %q, want active", reply.GetRegistered().GetCrawlExecutionId())
	}
}

func TestRegisterResourceSupportsActiveOptions(t *testing.T) {
	active := serverSession(t, 1, "active")
	request := &requests.Request{ID: "options-1", URL: "https://example.com/"}
	active.ObserveRequest(*request)
	server := NewApiServer(newMutableSessionLookup(active), nil, nil)

	reply, err := server.RegisterResource(context.Background(), &browsercontrollerV2.RegisterResourceRequest{
		ProxyId: int32(active.Id),
		Method:  http.MethodOptions,
		Uri:     request.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.GetRegistered() == nil {
		t.Fatalf("reply = %v, want registered", reply)
	}
	snapshot, found := active.RequestSnapshot(request.ID)
	if !found || !snapshot.GotComplete {
		t.Fatal("OPTIONS request was not marked complete")
	}
	if request.GotComplete {
		t.Fatal("OPTIONS completion mutated the caller-owned fixture")
	}
}

func TestRegisterResourceAcceptsMatchingExecution(t *testing.T) {
	active := serverSession(t, 1, "active")
	request := &requests.Request{ID: "network-1", URL: "https://example.com/", ResourceType: "Other"}
	active.ObserveRequest(*request)
	server := NewApiServer(newMutableSessionLookup(active), &testutil.RobotsEvaluatorMock{}, nil)

	reply, err := server.RegisterResource(context.Background(), &browsercontrollerV2.RegisterResourceRequest{
		ProxyId:          int32(active.Id),
		RequestId:        request.ID,
		Method:           http.MethodGet,
		Uri:              request.URL,
		CrawlExecutionId: "active",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.GetRegistered() == nil {
		t.Fatalf("reply = %v, want registered", reply)
	}
	snapshot, found := active.RequestSnapshot(request.ID)
	if !found || !snapshot.GotNew {
		t.Fatal("request was not marked new")
	}
	if request.GotNew {
		t.Fatal("registration mutated the caller-owned fixture")
	}
}

func TestRegisterResourceRejectsStaleExecution(t *testing.T) {
	active := serverSession(t, 1, "active")
	server := NewApiServer(newMutableSessionLookup(active), nil, nil)
	reply, err := server.RegisterResource(context.Background(), &browsercontrollerV2.RegisterResourceRequest{
		ProxyId:          int32(active.Id),
		RequestId:        "network-1",
		Method:           "GET",
		Uri:              "https://example.com/",
		CrawlExecutionId: "stale",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.GetCancel() != cancelledByBrowserController {
		t.Fatalf("cancel reason = %q, want %q", reply.GetCancel(), cancelledByBrowserController)
	}
}

func TestCompleteResourceRejectsStaleExecution(t *testing.T) {
	active := serverSession(t, 1, "active")
	request := &requests.Request{ID: "network-1", URL: "https://example.com/"}
	active.ObserveRequest(*request)

	server := NewApiServer(newMutableSessionLookup(active), nil, nil)
	_, err := server.CompleteResource(context.Background(), &browsercontrollerV2.CompleteResourceRequest{
		ProxyId:   int32(active.Id),
		RequestId: request.ID,
		CrawlLog: &logV1.CrawlLog{
			ExecutionId:  "stale",
			StatusCode:   200,
			Method:       "GET",
			RequestedUri: request.URL,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, found := active.RequestSnapshot(request.ID)
	if !found {
		t.Fatal("active request was not found after stale completion")
	}
	if snapshot.CrawlLog != nil {
		t.Fatal("stale completion mutated the active session")
	}
}

func TestCompleteResourceAcceptsMatchingExecution(t *testing.T) {
	active := serverSession(t, 1, "active")
	request := &requests.Request{ID: "network-1", URL: "https://example.com/"}
	active.ObserveRequest(*request)
	server := NewApiServer(newMutableSessionLookup(active), nil, nil)

	crawlLog := &logV1.CrawlLog{ExecutionId: "active", StatusCode: 200}
	if _, err := server.CompleteResource(context.Background(), &browsercontrollerV2.CompleteResourceRequest{
		ProxyId:   int32(active.Id),
		RequestId: request.ID,
		CrawlLog:  crawlLog,
	}); err != nil {
		t.Fatal(err)
	}
	snapshot, found := active.RequestSnapshot(request.ID)
	if !found || snapshot.CrawlLog == nil || snapshot.CrawlLog.GetStatusCode() != 200 {
		t.Fatal("matching completion did not update the session snapshot")
	}
	if request.CrawlLog != nil || snapshot.CrawlLog == crawlLog {
		t.Fatal("matching completion aliased caller-owned state")
	}
}

func TestRegisterResourceRevalidatesSessionAfterRobotsEvaluation(t *testing.T) {
	oldSession := serverSession(t, 1, "old")
	oldRequest := &requests.Request{ID: "network-1", URL: "https://example.com/", ResourceType: "Other"}
	oldSession.ObserveRequest(*oldRequest)
	newSession := serverSession(t, 1, "new")
	newRequest := &requests.Request{ID: oldRequest.ID, URL: oldRequest.URL, ResourceType: "Other"}
	newSession.ObserveRequest(*newRequest)

	lookup := newMutableSessionLookup(oldSession)
	entered := make(chan struct{})
	resume := make(chan struct{})
	robots := &testutil.RobotsEvaluatorMock{IsAllowedFunc: func(_ *robotsevaluatorV1.IsAllowedRequest) bool {
		close(entered)
		<-resume
		return true
	}}
	server := NewApiServer(lookup, robots, nil)

	type result struct {
		reply *browsercontrollerV2.RegisterResourceReply
		err   error
	}
	resultCh := make(chan result, 1)
	go func() {
		reply, err := server.RegisterResource(context.Background(), &browsercontrollerV2.RegisterResourceRequest{
			ProxyId:          1,
			RequestId:        oldRequest.ID,
			Method:           http.MethodGet,
			Uri:              oldRequest.URL,
			CrawlExecutionId: "old",
		})
		resultCh <- result{reply: reply, err: err}
	}()

	select {
	case <-entered:
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	lookup.setActive(1, newSession)
	close(resume)

	var got result
	select {
	case got = <-resultCh:
	case <-t.Context().Done():
		t.Fatal(t.Context().Err())
	}
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.reply.GetCancel() != cancelledByBrowserController {
		t.Fatalf("cancel reason = %q, want %q", got.reply.GetCancel(), cancelledByBrowserController)
	}
	oldSnapshot, oldFound := oldSession.RequestSnapshot(oldRequest.ID)
	newSnapshot, newFound := newSession.RequestSnapshot(newRequest.ID)
	if !oldFound || !newFound {
		t.Fatalf("request snapshots not found: old=%v new=%v", oldFound, newFound)
	}
	if oldSnapshot.GotNew || newSnapshot.GotNew {
		t.Fatalf("registration mutated a session: old=%v new=%v", oldSnapshot.GotNew, newSnapshot.GotNew)
	}
}
