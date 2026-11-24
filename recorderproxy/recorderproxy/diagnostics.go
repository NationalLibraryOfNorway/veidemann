package recorderproxy

import (
	"time"

	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	recorderV1 "github.com/NationalLibraryOfNorway/veidemann/api/recorderproxy/v1"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
)

func harvestLog(spec *recorderV1.BeginHarvestRequest) *logger.Logger {
	return logger.Log.WithFields(logger.Fields{"harvestId": spec.GetAttemptId(), "slot": spec.GetProxyId(), "pageUrl": logger.DiagnosticURL(spec.GetRequestedUri())})
}

func logRecordingOutcome(spec *recorderV1.BeginHarvestRequest, recorderRequestID int64, cl *logV1.CrawlLog, cached bool, persistErr error) {
	outcome := "failed"
	switch {
	case persistErr != nil:
		outcome = "persistence_failed"
	case cl.StatusCode == int32(rperrors.CanceledByBrowser):
		outcome = "canceled"
	case cached:
		outcome = "cached"
	case cl.WarcId != "":
		outcome = "recorded"
	}
	log := harvestLog(spec).WithFields(logger.Fields{"phase": "recording", "recorderRequestId": recorderRequestID, "url": logger.DiagnosticURL(cl.RequestedUri), "method": cl.Method, "outcome": outcome,
		"fetchDurationMs": cl.FetchTimeMs, "bytes": cl.Size, "cached": cached, "crawlLogPersisted": persistErr == nil})
	if cl.StatusCode >= 100 && cl.StatusCode <= 599 {
		log = log.WithField("httpStatus", cl.StatusCode)
	}
	if cl.WarcId != "" {
		log = log.WithField("warcId", cl.WarcId)
	}
	if cl.Error != nil {
		log = log.WithFields(logger.Fields{"errorCode": cl.Error.Code, "error": cl.Error.Msg, "errorDetail": cl.Error.Detail})
	}
	if persistErr != nil {
		log = log.WithField("persistenceError", persistErr)
	}
	log.Debug("Resource recording finished")
}

func (a *harvestAttempt) logFinalization(reply *recorderV1.FinishHarvestReply, accepted uint64, results []transferOutcome, pageLog string, start time.Time) {
	failed, canceled, cached := 0, 0, 0
	for _, result := range results {
		if result.cached {
			cached++
		}
		if result.log.StatusCode == int32(rperrors.CanceledByBrowser) {
			canceled++
		} else if result.log.Error != nil || result.log.StatusCode >= 400 || result.log.StatusCode < 0 {
			failed++
		}
	}
	log := harvestLog(a.spec).WithFields(logger.Fields{"phase": "recorder_finalization", "accepted": accepted, "finished": len(results), "recordedResources": reply.UriCount, "recordedBytes": reply.BytesDownloaded,
		"resourceFailures": failed, "resourceCancellations": canceled, "cachedResources": cached, "pageLog": pageLog, "durationMs": time.Since(a.started).Milliseconds(), "finalizationDurationMs": time.Since(start).Milliseconds(), "reusable": reply.Reusable})
	if failure := reply.Failure; failure != nil {
		log.WithFields(logger.Fields{"outcome": "failure", "errorCode": failure.Code, "error": failure.Msg, "errorDetail": failure.Detail}).Error("Recording finished")
	} else {
		log.WithField("outcome", "success").Info("Recording finished")
	}
}
