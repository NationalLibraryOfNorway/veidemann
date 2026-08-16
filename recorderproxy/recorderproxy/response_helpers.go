package recorderproxy

import (
	"errors"
	"net/http"

	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	rperrors "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	proxy "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/internal/proxy"
)

// handleRequestError creates a short circuit response for requests that fail before or in request handling.
// Only CrawlLog is sent, nothing is written to content writer.
func handleRequestError(
	cs *proxy.State,
	req *http.Request,
	reqErr error,
) (*http.Response, *proxy.State, error) {
	ctx := filterContext(cs, req)
	l := rpcontext.LogWithContextAndRequest(ctx, req, "REQH")
	l.WithError(reqErr).Debug("handling request error")

	rc := rpcontext.GetRecordContext(ctx)
	if rc == nil {
		// Important: no crawl resource exists, so do not send crawl log.
		return denyResponse(cs, req, reqErr)
	}

	e := rc.SendRequestError(ctx, reqErr)

	resp, nextCS := errorResponse(cs, req, e)
	if resp != nil {
		resp.Close = true
	}

	// Expected terminal request outcome, do not propagate as proxy failure.
	return resp, nextCS, nil
}

func Deny(
	cs *proxy.State,
	req *http.Request,
	status int,
	msg string,
) (*http.Response, *proxy.State, error) {
	resp, nextCS, _ := proxy.Fail(cs, req, status, errors.New(msg))
	if resp != nil {
		resp.Close = true
	}
	return resp, nextCS, nil
}

func denyResponse(
	cs *proxy.State,
	req *http.Request,
	err error,
) (*http.Response, *proxy.State, error) {
	resp, nextCS := errorResponse(cs, req, err)
	if resp != nil {
		resp.Close = true
	}
	return resp, nextCS, nil
}

// errorResponse creates a response from an error and populates Veidemann specific headers
func errorResponse(cs *proxy.State, req *http.Request, err error) (*http.Response, *proxy.State) {
	resp, nextCS, _ := proxy.Fail(cs, req, rperrors.HttpStatusCode(err), err)
	if resp != nil {
		resp.Header.Add(constants.HeaderProxyErrorCode, rperrors.Code(err).String())
		resp.Header.Add(constants.HeaderProxyError, rperrors.Message(err))
	}
	return resp, nextCS
}
