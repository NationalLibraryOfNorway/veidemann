package controller

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/database"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
)

type workerFrontier struct {
	next     atomic.Int32
	reported chan error
}

func (f *workerFrontier) GetNextPage(context.Context) (*frontierV1.PageHarvestSpec, error) {
	id := f.next.Add(1)
	return &frontierV1.PageHarvestSpec{
		QueuedUri:   &frontierV1.QueuedUri{Uri: fmt.Sprintf("https://example.com/%d", id)},
		CrawlConfig: &configV1.ConfigObject{Spec: &configV1.ConfigObject_CrawlConfig{CrawlConfig: &configV1.CrawlConfig{}}},
	}, nil
}
func (f *workerFrontier) PageCompleted(ctx context.Context, _ *frontierV1.PageHarvestSpec, _ *frontier.RenderResult, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	f.reported <- err
	return nil
}

type blockingConfig struct {
	database.ConfigAdapter
	entered chan struct{}
	release chan struct{}
}

func (c *blockingConfig) GetConfigObject(context.Context, *configV1.ConfigRef) (*configV1.ConfigObject, error) {
	c.entered <- struct{}{}
	<-c.release
	return nil, errors.New("configuration unavailable")
}

func TestWorkersBoundConcurrencyAndReportActiveHarvestsOnShutdown(t *testing.T) {
	const slots = 3
	config := &blockingConfig{entered: make(chan struct{}, slots+1), release: make(chan struct{})}
	frontier := &workerFrontier{reported: make(chan error, slots+1)}
	controller := New(make([]string, slots), frontier, time.Minute, time.Minute, session.WithConfigAdapter(config))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- controller.Run(ctx) }()
	deadline := time.After(3 * time.Second)
	for range slots {
		select {
		case <-config.entered:
		case <-deadline:
			t.Fatal("workers did not start")
		}
	}
	cancel()
	select {
	case err := <-done:
		t.Fatalf("abandoned active harvests: %v", err)
	default:
	}
	close(config.release)
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-deadline:
		t.Fatal("workers did not stop")
	}
	if frontier.next.Load() != slots || len(frontier.reported) != slots {
		t.Fatalf("started=%d reported=%d", frontier.next.Load(), len(frontier.reported))
	}
	for range slots {
		if err := <-frontier.reported; err == nil {
			t.Fatal("lost fetch failure")
		}
	}
}
