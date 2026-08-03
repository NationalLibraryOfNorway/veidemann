package archivingcache

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/olric-data/olric"
	"github.com/olric-data/olric/config"
)

func TestOlricCache(t *testing.T) {
	address, shutdown := startTestOlric(t)
	defer shutdown()

	cache, err := NewOlricCache([]string{address}, "archivingcache-test")
	if err != nil {
		t.Fatalf("NewOlricCache() error = %v", err)
	}
	defer func() {
		if err := cache.Close(context.Background()); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	ctx := context.Background()

	t.Run("miss maps to ErrKeyNotFound", func(t *testing.T) {
		_, err := cache.Get(ctx, "missing")
		if !errors.Is(err, ErrKeyNotFound) {
			t.Fatalf("Get() error = %v, want %v", err, ErrKeyNotFound)
		}
	})

	t.Run("set get and expiry", func(t *testing.T) {
		value := []byte("cached value")
		if err := cache.Set(ctx, "example.org.A", value, 100*time.Millisecond); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		got, err := cache.Get(ctx, "example.org.A")
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(got) != string(value) {
			t.Fatalf("Get() = %q, want %q", got, value)
		}

		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			_, err = cache.Get(ctx, "example.org.A")
			if errors.Is(err, ErrKeyNotFound) {
				return
			}
			if err != nil {
				t.Fatalf("Get() during TTL wait error = %v", err)
			}
			time.Sleep(10 * time.Millisecond)
		}

		t.Fatal("expected entry to expire from Olric cache")
	})
}

func startTestOlric(t *testing.T) (string, func()) {
	t.Helper()

	port := freePort(t)
	cfg := config.New("local")
	cfg.BindAddr = "127.0.0.1"
	cfg.BindPort = port
	cfg.MemberlistConfig.BindPort = 0

	started := make(chan struct{})
	startErr := make(chan error, 1)
	cfg.Started = func() {
		close(started)
	}

	if err := cfg.Sanitize(); err != nil {
		t.Fatalf("Sanitize() error = %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	db, err := olric.New(cfg)
	if err != nil {
		t.Fatalf("olric.New() error = %v", err)
	}

	go func() {
		startErr <- db.Start()
	}()

	select {
	case <-started:
	case err := <-startErr:
		t.Fatalf("Olric failed to start: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("Olric did not start in time")
	}

	address := net.JoinHostPort(cfg.BindAddr, fmt.Sprintf("%d", cfg.BindPort))

	return address, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := db.Shutdown(ctx); err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
		select {
		case err := <-startErr:
			if err != nil {
				t.Fatalf("Olric start loop returned error during shutdown: %v", err)
			}
		case <-time.After(2 * time.Second):
		}
	}
}

func freePort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	defer func() {
		err = listener.Close()
		if err != nil {
			t.Errorf("listener.Close() error = %v", err)
		}
	}()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCPAddr, got %T", listener.Addr())
	}
	return addr.Port
}
