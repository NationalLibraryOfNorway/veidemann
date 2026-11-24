package context

import (
	"net/http/httptest"
	"testing"

	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
)

func TestDocumentNavigationMarker(t *testing.T) {
	for _, scheme := range []string{"http", "https"} {
		for _, marker := range []string{"", "1", "old-attempt", "current-attempt"} {
			t.Run(scheme+"/"+marker, func(t *testing.T) {
				req := httptest.NewRequest("GET", scheme+"://example.com/", nil)
				req = req.WithContext(RecordProxyDataAware(t.Context()))
				req.Header.Set(constants.HeaderMainDocument, marker)
				req.Header.Set("Sec-Fetch-Dest", "document")
				req.Header.Set("Sec-Fetch-Mode", "navigate")
				req.Header.Set("Upgrade-Insecure-Requests", "1")
				req.Header.Set("Accept", "text/html")
				rc := NewRecordContext()
				rc.Harvest = &recorderV1.BeginHarvestRequest{AttemptId: "current-attempt"}
				rc.Init(1, nil, req, req.URL)
				defer rc.SaveCrawlLog()
				if rc.MainDocument != (marker == "current-attempt") {
					t.Fatalf("main=%v marker=%q", rc.MainDocument, marker)
				}
				if req.Header.Get(constants.HeaderMainDocument) != "" {
					t.Fatal("marker not removed")
				}
			})
		}
	}
}
