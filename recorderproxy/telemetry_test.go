package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMetricsServer(t *testing.T) {
	server := newMetricsServer(":0", "/metrics")
	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !strings.Contains(recorder.Body.String(), "go_goroutines") {
		t.Fatal("metrics response does not contain Go runtime metrics")
	}
	if !strings.Contains(recorder.Body.String(), "recorderproxy_idle_connection_timeouts_total") {
		t.Fatal("metrics response does not contain recorderproxy idle connection timeout metric")
	}

	profileRecorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(profileRecorder, httptest.NewRequest(http.MethodGet, "/debug/pprof/heap", nil))
	if profileRecorder.Code != http.StatusNotFound {
		t.Fatalf("profile status on metrics server = %d, want %d", profileRecorder.Code, http.StatusNotFound)
	}
}

func TestProfilingServer(t *testing.T) {
	server := newProfilingServer(":0")
	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/debug/pprof/heap", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("heap profile status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/octet-stream" {
		t.Fatalf("heap profile content type = %q, want application/octet-stream", contentType)
	}
}
