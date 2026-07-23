package recorderproxy

import (
	"errors"
	"net/http"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"github.com/getlantern/proxy/v3/filters"
)

// handleRequestError creates a short circuit response for requests that fail before or in request handling.
// Only CrawlLog is sent, nothing is written to content writer.
func handleRequestError(
	cs *filters.ConnectionState,
	req *http.Request,
	reqErr error,
) (*http.Response, *filters.ConnectionState, error) {
	ctx := filterContext(cs, req)
	l := rpcontext.LogWithContextAndRequest(ctx, req, "REQH")
	l.WithError(reqErr).Debug("handling request error")

	rc := rpcontext.GetRecordContext(ctx)
	if rc == nil {
		// Important: no crawl resource exists, so do not send crawl log.
		return denyResponse(cs, req, reqErr)
	}

	e := rc.SendRequestError(ctx, reqErr)
	_ = rc.CancelContentWriter(rperrors.Detail(e))

	resp, nextCS := errorResponse(cs, req, e)
	if resp != nil {
		resp.Close = true
	}

	// Expected terminal request outcome, do not propagate as proxy failure.
	return resp, nextCS, nil
}

func Deny(
	cs *filters.ConnectionState,
	req *http.Request,
	status int,
	msg string,
) (*http.Response, *filters.ConnectionState, error) {
	resp, nextCS, _ := filters.Fail(cs, req, status, errors.New(msg))
	if resp != nil {
		resp.Close = true
	}
	return resp, nextCS, nil
}

func denyResponse(
	cs *filters.ConnectionState,
	req *http.Request,
	err error,
) (*http.Response, *filters.ConnectionState, error) {
	resp, nextCS := errorResponse(cs, req, err)
	if resp != nil {
		resp.Close = true
	}
	return resp, nextCS, nil
}

// errorResponse creates a response from an error and populates Veidemann specific headers
func errorResponse(cs *filters.ConnectionState, req *http.Request, err error) (*http.Response, *filters.ConnectionState) {
	resp, nextCS, _ := filters.Fail(cs, req, rperrors.HttpStatusCode(err), err)
	if resp != nil {
		resp.Header.Add(constants.HeaderProxyErrorCode, rperrors.Code(err).String())
		resp.Header.Add(constants.HeaderProxyError, rperrors.Message(err))
	}
	return resp, nextCS
}
