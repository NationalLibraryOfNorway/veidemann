package archivingcache

import (
	"reflect"
	"testing"

	"github.com/coredns/caddy"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		input     string
		shouldErr bool
		addresses []string
		dmap      string
	}{
		{`archivingcache`, false, []string{defaultOlricAddress}, defaultOlricDMap},
		{`archivingcache {
				olricAddress olric-client:3320
			}`, false, []string{"olric-client:3320"}, defaultOlricDMap},
		{`archivingcache {
				olricAddress olric-a:3320
				olricAddress olric-b:3320
				olricDmap dns-cache
			}`, false, []string{"olric-a:3320", "olric-b:3320"}, "dns-cache"},
		{`archivingcache {
				olricAddress olric-a:3320,olric-b:3320
			}`, false, []string{"olric-a:3320", "olric-b:3320"}, defaultOlricDMap},
		{`archivingcache {
				contentWriterHost cwHost
			}`, false, []string{defaultOlricAddress}, defaultOlricDMap},

		// fails
		{`archivingcache {
				eviction 15s
			}`, true, nil, ""},
		{`archivingcache example.nl {
				olricAddress
			}`, true, nil, ""},
		{`archivingcache {
				olricDmap
			}`, true, nil, ""},
		{`archivingcache {
				positive 0
			}`, true, nil, ""},
		{`archivingcache
		  archivingcache`, true, nil, ""},
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
		if !reflect.DeepEqual(a.olricAddresses, test.addresses) {
			t.Errorf("Test %v: expected addresses %v, got %v", i, test.addresses, a.olricAddresses)
		}
		if a.olricDMap != test.dmap {
			t.Errorf("Test %v: expected dmap %q, got %q", i, test.dmap, a.olricDMap)
		}
	}
}
