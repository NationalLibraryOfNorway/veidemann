package archivingcache

import (
	"reflect"
	"testing"
	"time"

	"github.com/coredns/caddy"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		input     string
		shouldErr bool
		eviction  time.Duration
		addresses []string
		dmap      string
	}{
		{`archivingcache`, false, defaultEviction, []string{defaultOlricAddress}, defaultOlricDMap},
		{`archivingcache {
				eviction 10s
			}`, false, 10 * time.Second, []string{defaultOlricAddress}, defaultOlricDMap},
		{`archivingcache {
				olricAddress olric-client:3320
			}`, false, defaultEviction, []string{"olric-client:3320"}, defaultOlricDMap},
		{`archivingcache {
				eviction 10m
				olricAddress olric-a:3320
				olricAddress olric-b:3320
				olricDmap dns-cache
			}`, false, 10 * 60 * time.Second, []string{"olric-a:3320", "olric-b:3320"}, "dns-cache"},
		{`archivingcache {
				olricAddress olric-a:3320,olric-b:3320
			}`, false, defaultEviction, []string{"olric-a:3320", "olric-b:3320"}, defaultOlricDMap},
		{`archivingcache {
				contentWriterHost cwHost
			}`, false, defaultEviction, []string{defaultOlricAddress}, defaultOlricDMap},

		// fails
		{`archivingcache example.nl {
				eviction
				olricAddress olric-client:3320
			}`, true, defaultEviction, nil, ""},
		{`archivingcache example.nl {
				eviction 15s
				olricAddress
			}`, true, defaultEviction, nil, ""},
		{`archivingcache example.nl {
				eviction aaa
				olricDmap dns-cache
			}`, true, defaultEviction, nil, ""},
		{`archivingcache {
				olricDmap
			}`, true, defaultEviction, nil, ""},
		{`archivingcache {
				positive 0
			}`, true, defaultEviction, nil, ""},
		{`archivingcache
		  archivingcache`, true, defaultEviction, nil, ""},
	}
	for i, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		a, err := parseArchivingCache(c)
		if test.shouldErr && err == nil {
			t.Errorf("Test %v: Expected error but found nil", i)
			continue
		} else if !test.shouldErr && err != nil {
			t.Errorf("Test %v: Expected no error but found error: %v", i, err)
			continue
		}
		if test.shouldErr {
			continue
		}
		if a.eviction != test.eviction {
			t.Errorf("Test %v: expected eviction %v, got %v", i, test.eviction, a.eviction)
		}
		if !reflect.DeepEqual(a.olricAddresses, test.addresses) {
			t.Errorf("Test %v: expected addresses %v, got %v", i, test.addresses, a.olricAddresses)
		}
		if a.olricDMap != test.dmap {
			t.Errorf("Test %v: expected dmap %q, got %q", i, test.dmap, a.olricDMap)
		}
	}
}
