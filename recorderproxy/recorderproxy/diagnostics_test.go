package recorderproxy

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/testutil"
)

type recordingLogCapture struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (c *recordingLogCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buffer.Write(p)
}
func (c *recordingLogCapture) records(t *testing.T) []map[string]any {
	t.Helper()
	c.mu.Lock()
	raw := append([]byte(nil), c.buffer.Bytes()...)
	c.mu.Unlock()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	var rows []map[string]any
	for {
		var rawRecord json.RawMessage
		err := decoder.Decode(&rawRecord)
		if err == io.EOF {
			return rows
		}
		if err != nil {
			t.Fatal(err)
		}
		keys := json.NewDecoder(bytes.NewReader(rawRecord))
		if _, err := keys.Token(); err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for keys.More() {
			key, err := keys.Token()
			if err != nil {
				t.Fatal(err)
			}
			field := key.(string)
			if seen[field] {
				t.Fatalf("duplicate JSON key %q in %s", field, rawRecord)
			}
			seen[field] = true
			var value json.RawMessage
			if err := keys.Decode(&value); err != nil {
				t.Fatal(err)
			}
		}
		var row map[string]any
		if err := json.Unmarshal(rawRecord, &row); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
}
func (c *recordingLogCapture) matching(t *testing.T, message string) []map[string]any {
	var found []map[string]any
	for _, row := range c.records(t) {
		if row["msg"] == message {
			found = append(found, row)
		}
	}
	return found
}
func captureRecordingLogs(t *testing.T, level slog.Level) *recordingLogCapture {
	t.Helper()
	capture := &recordingLogCapture{}
	previous := logger.Log
	logger.Log = logger.New(slog.New(slog.NewJSONHandler(capture, &slog.HandlerOptions{Level: level})))
	t.Cleanup(func() { logger.Log = previous })
	return capture
}

func TestRecordingSummariesAndTerminalOutcomes(t *testing.T) {
	for _, scenario := range []string{"stored", "resource-failure", "canceled", "page-persistence-failure", "crawl-persistence-failure", "cached", "http-error"} {
		t.Run(scenario, func(t *testing.T) {
			capture := captureRecordingLogs(t, slog.LevelDebug)
			control, proxy := testControl()
			spec := beginSpec("logging-attempt-00001")
			if _, err := control.BeginHarvest(t.Context(), spec); err != nil {
				t.Fatal(err)
			}
			if _, err := control.BeginHarvest(t.Context(), spec); err != nil {
				t.Fatal(err)
			}
			cl := &logV1.CrawlLog{RequestedUri: spec.RequestedUri, Method: "GET", StatusCode: 200, WarcId: "confirmed-warc", Size: 120, FetchTimeMs: 42}
			switch scenario {
			case "http-error":
				cl.StatusCode = 404
			case "resource-failure":
				cl.WarcId = ""
				cl.StatusCode = -5002
				cl.Error = &commonsV1.Error{Code: -5002, Msg: "origin refused"}
			case "canceled":
				cl.WarcId = ""
				cl.StatusCode = int32(rperrors.CanceledByBrowser)
				cl.Error = &commonsV1.Error{Code: int32(rperrors.CanceledByBrowser), Msg: "canceled"}
			case "page-persistence-failure":
				control.logs.(*testutil.LogWriterMock).PageError = errors.New("page log unavailable")
			case "crawl-persistence-failure":
				control.logs.(*testutil.LogWriterMock).CrawlError = errors.New("crawl log unavailable")
			}
			rc := &rpcontext.RecordContext{MainDocument: true}
			if !proxy.attempt.admit(rc, func() { proxy.attempt.lifecycle.addRecord(rc) }) {
				t.Fatal("admission rejected")
			}
			rc.Complete(cl, scenario == "cached", nil)
			rc.CloseFunc()
			request := &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: spec.AttemptId, PageUri: &spec.RequestedUri}
			for range 2 {
				if _, err := control.FinishPage(t.Context(), request, nil); err != nil {
					t.Fatal(err)
				}
			}
			starts := capture.matching(t, "Recording started")
			summaries := capture.matching(t, "Recording finished")
			terminals := capture.matching(t, "Resource recording finished")
			if len(starts) != 1 || len(summaries) != 1 || len(terminals) != 1 {
				t.Fatalf("starts=%v summaries=%v terminal=%v", starts, summaries, terminals)
			}
			for _, row := range []map[string]any{starts[0], summaries[0], terminals[0]} {
				if row["component"] != "recorderproxy" || row["harvestId"] != spec.AttemptId || row["slot"] != float64(1) || row["pageUrl"] != spec.RequestedUri || row["duration"] != nil {
					t.Fatal(row)
				}
			}
			summary, terminal := summaries[0], terminals[0]
			if summary["accepted"] != float64(1) || summary["finished"] != float64(1) || summary["durationMs"] == nil || terminal["fetchDurationMs"] != float64(42) || terminal["durationMs"] != nil || summary["finalizationDurationMs"] == nil {
				t.Fatal(summary, terminal)
			}
			if terminal["attempt"] != nil || terminal["uri"] != nil || terminal["transferId"] != nil || terminal["statusCode"] != nil {
				t.Fatal("legacy log fields", terminal)
			}
			if cl.StatusCode < 0 && terminal["httpStatus"] != nil {
				t.Fatal("recorder code used as HTTP status", terminal)
			}
			if scenario == "http-error" && (terminal["outcome"] != "recorded" || terminal["httpStatus"] != float64(404)) {
				t.Fatal(terminal)
			}
			if scenario == "cached" && terminal["outcome"] != "cached" {
				t.Fatal(terminal)
			}
			if scenario == "stored" && (summary["pageLog"] != "stored" || summary["recordedResources"] != float64(1) || terminal["outcome"] != "recorded") {
				t.Fatal(summary, terminal)
			}
			if scenario == "resource-failure" && (summary["resourceFailures"] != float64(1) || terminal["outcome"] != "failed") {
				t.Fatal(summary, terminal)
			}
			if scenario == "canceled" && (summary["resourceCancellations"] != float64(1) || terminal["outcome"] != "canceled") {
				t.Fatal(summary, terminal)
			}
			if scenario == "crawl-persistence-failure" && (terminal["outcome"] != "persistence_failed" || terminal["crawlLogPersisted"] != false || summary["level"] != "ERROR") {
				t.Fatal(summary, terminal)
			}
			if scenario == "page-persistence-failure" && (summary["pageLog"] != "failed" || summary["level"] != "ERROR") {
				t.Fatal(summary)
			}
		})
	}
}

func TestRecordingDebugEventsAreHiddenAtInfo(t *testing.T) {
	capture := captureRecordingLogs(t, slog.LevelInfo)
	logRecordingOutcome(beginSpec("logging-attempt-00002"), 1, &logV1.CrawlLog{RequestedUri: "https://example.com/", WarcId: "recorded"}, false, nil)
	if len(capture.records(t)) != 0 {
		t.Fatal("INFO produced request detail")
	}
}

func TestRecordingSeparatesHarvestAndFinalizationDurations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		capture := captureRecordingLogs(t, slog.LevelInfo)
		control, _ := testControl()
		spec := beginSpec("logging-duration-00003")
		if _, err := control.BeginHarvest(t.Context(), spec); err != nil {
			t.Fatal(err)
		}
		time.Sleep(12 * time.Second)
		if _, err := control.FinishPage(t.Context(), &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: spec.AttemptId}, nil); err != nil {
			t.Fatal(err)
		}
		rows := capture.matching(t, "Recording finished")
		if len(rows) != 1 || rows[0]["durationMs"] != float64(12000) || rows[0]["finalizationDurationMs"] != float64(0) {
			t.Fatal(rows)
		}
	})
}

func TestCompactDebugHasOneResultPerTransfer(t *testing.T) {
	capture := captureRecordingLogs(t, slog.LevelDebug)
	control, proxy := testControl()
	spec := beginSpec("logging-volume-00004")
	if _, err := control.BeginHarvest(t.Context(), spec); err != nil {
		t.Fatal(err)
	}
	for range 85 {
		rc := &rpcontext.RecordContext{}
		if !proxy.attempt.admit(rc, func() { proxy.attempt.lifecycle.addRecord(rc) }) {
			t.Fatal("admission rejected")
		}
		rc.Complete(&logV1.CrawlLog{RequestedUri: spec.RequestedUri, Method: "GET", StatusCode: 200, WarcId: "confirmed", Size: 120}, false, nil)
		rc.CloseFunc()
	}
	if _, err := control.FinishPage(t.Context(), &recorderV1.FinishHarvestHeader{ProxyId: 1, AttemptId: spec.AttemptId}, nil); err != nil {
		t.Fatal(err)
	}
	results := capture.matching(t, "Resource recording finished")
	if len(results) != 85 {
		t.Fatal("unexpected resource result count", len(results))
	}
	for _, row := range capture.records(t) {
		if row["level"] == "DEBUG" && row["msg"] != "Resource recording finished" {
			t.Fatal("intermediate DEBUG event", row)
		}
	}
	summary := capture.matching(t, "Recording finished")
	if len(summary) != 1 || summary[0]["accepted"] != float64(85) || summary[0]["finished"] != float64(85) {
		t.Fatal(summary)
	}
}
