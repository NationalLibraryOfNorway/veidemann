package recorderproxy

import (
	"context"
	"net"
	"testing"
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestRecoverSlotAfterControllerLoss(t *testing.T) {
	for _, timedOut := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-finish", true: "after-finish-timeout"}[timedOut], func(t *testing.T) {
			c, p := testControl()
			p.finalizationTimeout = 20 * time.Millisecond
			spec := beginSpec("old-controller-attempt-0001")
			if _, err := c.BeginHarvest(t.Context(), spec); err != nil {
				t.Fatal(err)
			}
			a := p.attempt
			rc := &rpcontext.RecordContext{}
			a.admit(rc, func() { a.lifecycle.addRecord(rc) })
			finish := &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: spec.AttemptId}
			var published *recorderV1.FinishHarvestReply
			if timedOut {
				var err error
				published, err = c.FinishPage(t.Context(), finish, nil)
				if err != nil || published.Reusable || published.GetFailure().GetCode() != -5 {
					t.Fatalf("finish = %v, %v", published, err)
				}
			}

			// A new controller keeps the recorder sidecar and reconciles startup.
			recover := &recorderV1.RecoverSlotRequest{ProxyId: 1}
			if _, err := c.RecoverSlot(t.Context(), recover); status.Code(err) != codes.DeadlineExceeded {
				t.Fatalf("recovered undrained work: %v", err)
			}
			next := beginSpec("new-controller-attempt-0002")
			next.AdmissionToken = p.nextToken
			if _, err := c.BeginHarvest(t.Context(), next); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("began before drainage: %v", err)
			}
			rc.Complete(&logV1.CrawlLog{RequestedUri: spec.RequestedUri, WarcId: "late"}, false, nil)
			rc.CloseFunc()
			if _, err := c.RecoverSlot(t.Context(), recover); err != nil {
				t.Fatal(err)
			}
			if published != nil {
				replay, err := c.FinishPage(t.Context(), finish, nil)
				if err != nil || !proto.Equal(published, replay) || replay.Reusable {
					t.Fatalf("recovery changed published reply: %v, %v", replay, err)
				}
			}
			logs := c.logs.(*testutil.LogWriterMock)
			if len(logs.CrawlLogs()) != 1 || len(logs.PageLogs()) != 0 {
				t.Fatal("lost late crawl log or invented page capture")
			}
			if _, err := c.BeginHarvest(t.Context(), next); err != nil {
				t.Fatal(err)
			}
			if _, err := c.BeginHarvest(t.Context(), spec); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("old attempt revived: %v", err)
			}
			if _, err := c.FinishPage(t.Context(), finish, nil); status.Code(err) != codes.FailedPrecondition {
				t.Fatalf("stale finish accepted: %v", err)
			}
		})
	}
}

func TestRecoverSlotClosesConnectionsAndWaitsForHandlers(t *testing.T) {
	c, p := testControl()
	_, _ = c.BeginHarvest(t.Context(), beginSpec("orphaned-attempt-0001"))
	a := p.attempt
	client, peer := net.Pipe()
	defer peer.Close()
	conn := WrapConn(client, "test", false)
	connectionCtx, cancel := context.WithCancel(t.Context())
	defer cancel()
	conn.cancelFunc = cancel
	a.lifecycle.addConnection(conn)
	done := make(chan error, 1)
	go func() {
		_, err := c.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 1})
		done <- err
	}()
	select {
	case <-connectionCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("recovery did not close the orphan connection")
	}
	select {
	case err := <-done:
		t.Fatalf("recovery did not wait for handler exit: %v", err)
	default:
	}
	a.lifecycle.removeConnection(conn)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if a.lifecycle.addConnection(conn) || a.admit(&rpcontext.RecordContext{}, func() {}) {
		t.Fatal("old attempt reopened")
	}
	next := beginSpec("recovered-attempt-0002")
	next.AdmissionToken = p.nextToken
	if _, err := c.BeginHarvest(t.Context(), next); err != nil {
		t.Fatal(err)
	}
}

func TestRecoverSlotEmptyAndUnknown(t *testing.T) {
	c, _ := testControl()
	for range 2 {
		if _, err := c.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := c.RecoverSlot(t.Context(), &recorderV1.RecoverSlotRequest{ProxyId: 2}); status.Code(err) != codes.NotFound {
		t.Fatal(err)
	}
}
