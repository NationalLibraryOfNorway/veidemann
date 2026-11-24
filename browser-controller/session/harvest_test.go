package session

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	bcerrors "github.com/NationalLibraryOfNorway/veidemann/browser-controller/errors"
	"google.golang.org/grpc"
)

type recorderReply struct {
	recorderV1.RecorderControlClient
	reply   *recorderV1.FinishHarvestReply
	err     error
	request *recorderV1.FinishHarvestHeader
	links   []string
	grpc.ClientStream
}

func (r *recorderReply) FinishHarvest(context.Context, ...grpc.CallOption) (recorderV1.RecorderControl_FinishHarvestClient, error) {
	return r, nil
}
func (r *recorderReply) Send(msg *recorderV1.FinishHarvestRequest) error {
	if header := msg.GetHeader(); header != nil {
		r.request = header
	} else {
		r.links = append(r.links, msg.GetOutlink())
	}
	return nil
}
func (r *recorderReply) CloseAndRecv() (*recorderV1.FinishHarvestReply, error) { return r.reply, r.err }

func TestFinishUsesRecorderOutcomeAndExplicitReusability(t *testing.T) {
	for _, scenario := range []string{"success", "failure", "undrained", "interrupted"} {
		t.Run(scenario, func(t *testing.T) {
			client := &recorderReply{reply: &recorderV1.FinishHarvestReply{Reusable: true}}
			if scenario == "failure" {
				client.reply.Failure = &commonsV1.Error{Code: -3, Msg: "origin refused"}
			}
			if scenario == "undrained" {
				client.reply.Reusable = false
				client.reply.Failure = &commonsV1.Error{Code: -5, Msg: "any diagnostic text"}
			}
			if scenario == "interrupted" {
				client.reply = nil
				client.err = context.DeadlineExceeded
			}
			sess := New(3, WithRecorder(client))
			sess.attemptID = "attempt"
			page := "https://example.com/#fragment"
			links := []string{"https://example.com/link"}
			_, err := sess.finishHarvest(t.Context(), &page, links)
			if (err != nil) != (scenario != "success") {
				t.Fatal(err)
			}
			if sess.Reusable() != (scenario == "success" || scenario == "failure") {
				t.Fatal("incorrect slot isolation")
			}
			if client.request.ProxyId != 3 || client.request.AttemptId != "attempt" || client.request.GetPageUri() != page || len(client.links) != 1 || client.links[0] != links[0] {
				t.Fatal(client.request)
			}
			if scenario == "failure" && bcerrors.CommonsError(preferFailure(errors.New("net::ERR_FAILED"), err)).Code != -3 {
				t.Fatal(err)
			}
		})
	}
}

func TestFinalizationFailureOverridesBrowsingTimeout(t *testing.T) {
	err := preferFailure(bcerrors.New(-5004, "RUNTIME_EXCEEDED", "no capture"), context.DeadlineExceeded)
	if bcerrors.CommonsError(err).Code != -5 {
		t.Fatal(err)
	}
}

func TestBrowserCleanupTimeoutPreventsSlotReuse(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sess := newDefaultSession()
		release := make(chan struct{})
		start := time.Now()
		err := sess.closeBrowser(func() { <-release })
		close(release)
		synctest.Wait()
		if err == nil || sess.Reusable() || time.Since(start) != 5*time.Second {
			t.Fatalf("cleanup=%v reusable=%v", err, sess.Reusable())
		}
	})
}

func TestSuccessfulCleanupPreservesBrowsingTimeout(t *testing.T) {
	original := bcerrors.New(-5004, "timeout", "no main document")
	if got := preferFailure(original, nil); got != original {
		t.Fatal(got)
	}
}

func TestBrowserSetupFailureStillProvidesCleanup(t *testing.T) {
	sess := newDefaultSession()
	sess.browserWsEndpoint = "://invalid"
	_, cleanup, err := sess.startBrowserSession(t.Context())
	if err == nil {
		t.Fatal("invalid browser endpoint accepted")
	}
	if err := sess.closeBrowser(cleanup); err != nil || !sess.Reusable() {
		t.Fatalf("setup cleanup: %v", err)
	}
}
