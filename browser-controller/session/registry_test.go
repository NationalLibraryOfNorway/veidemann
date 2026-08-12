package session

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
)

func TestRegistryGetActiveFollowsAcceptanceState(t *testing.T) {
	registry := NewRegistry(2)
	for _, id := range []int{-1, 0, 1, 2} {
		if got := registry.GetActive(id); got != nil {
			t.Fatalf("GetActive(%d) = %p before allocation, want nil", id, got)
		}
	}

	sess, err := registry.GetNextAvailable(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if got := registry.GetActive(sess.Id); got != nil {
		t.Fatalf("initializing session = %p, want nil", got)
	}

	sess.startAcceptingRequests()
	if got := registry.GetActive(sess.Id); got != sess {
		t.Fatalf("active session = %p, want %p", got, sess)
	}

	sess.stopAcceptingRequests()
	if got := registry.GetActive(sess.Id); got != nil {
		t.Fatalf("draining session = %p, want nil", got)
	}

	sess.startAcceptingRequests()
	registry.Release(sess)
	if sess.acceptingRequests() {
		t.Fatal("released session is still accepting requests")
	}
	if got := registry.GetActive(sess.Id); got != nil {
		t.Fatalf("released session = %p, want nil", got)
	}
}

func TestRegistryGetActiveConcurrentWithSlotReuse(t *testing.T) {
	registry := NewRegistry(2)
	done := make(chan struct{})
	errCh := make(chan error, 1)

	var readers sync.WaitGroup
	for range 4 {
		readers.Go(func() {
			for {
				select {
				case <-done:
					return
				default:
				}
				if sess := registry.GetActive(1); sess != nil && sess.Id != 1 {
					select {
					case errCh <- fmt.Errorf("GetActive(1) returned session %d", sess.Id):
					default:
					}
					return
				}
			}
		})
	}

	var previous *Session
	for range 250 {
		sess, err := registry.GetNextAvailable(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if sess == previous {
			t.Fatal("registry reused a Session object instead of only reusing its slot")
		}
		sess.startAcceptingRequests()
		runtime.Gosched()
		registry.Release(sess)
		previous = sess
	}

	close(done)
	readers.Wait()
	select {
	case err := <-errCh:
		t.Fatal(err)
	default:
	}
}
