package archivingcache

import (
	"reflect"
	"testing"
	"time"

	"github.com/miekg/dns"
)

func TestCacheEntryRoundTrip(t *testing.T) {
	storedAt := time.Date(2026, time.August, 3, 12, 0, 0, 123, time.UTC)
	entry := &CacheEntry{
		StoredAt:  storedAt,
		ExpiresAt: storedAt.Add(10 * time.Minute),
		ProxyAddr: "127.0.0.1",
		Msg: new(dns.Msg).SetQuestion(
			"example.org.",
			dns.TypeA,
		),
	}

	packed, err := entry.pack()
	if err != nil {
		t.Fatalf("pack() error = %v", err)
	}

	got := new(CacheEntry)
	if err := got.unpack(packed); err != nil {
		t.Fatalf("unpack() error = %v", err)
	}
	if !got.StoredAt.Equal(entry.StoredAt) {
		t.Errorf("StoredAt = %v, want %v", got.StoredAt, entry.StoredAt)
	}
	if !got.ExpiresAt.Equal(entry.ExpiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, entry.ExpiresAt)
	}
	if got.ProxyAddr != entry.ProxyAddr {
		t.Errorf("ProxyAddr = %q, want %q", got.ProxyAddr, entry.ProxyAddr)
	}
	if got.Msg == nil || !reflect.DeepEqual(got.Msg, entry.Msg) {
		t.Errorf("Msg = %v, want %v", got.Msg, entry.Msg)
	}
}

func TestCacheEntryRejectsUnsupportedValue(t *testing.T) {
	unsupported := append([]byte(nil), cacheEntryMagic[:]...)
	unsupported[3] = 2
	entry := new(CacheEntry)
	if err := entry.unpack(unsupported); err == nil {
		t.Fatal("unpack() expected an error for an unsupported value")
	}
}

func TestCacheEntryRequiresMessage(t *testing.T) {
	if _, err := new(CacheEntry).pack(); err == nil {
		t.Fatal("pack() expected an error without a DNS message")
	}
}
