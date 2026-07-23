package recorderproxy

import (
	"bytes"
	"context"
	"net/http"
	"strings"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
)

type responseRecorder struct {
	ctx context.Context
	rc  *rpcontext.RecordContext
}

func newResponseRecorder(ctx context.Context, rc *rpcontext.RecordContext) responseRecorder {
	return responseRecorder{
		ctx: ctx,
		rc:  rc,
	}
}

func (r responseRecorder) Wrap(resp *http.Response) (*http.Response, error) {
	if resp == nil {
		panic(http.ErrAbortHandler)
	}

	if shouldSkipResponseRecording(r.rc) {
		return resp, nil
	}

	if isFromCache(resp) {
		rpcontext.LogWithRecordContext(r.rc, "FLT:rec").Info("Loaded from cache")
		r.rc.FoundInCache = true
	}

	// WIN: prolog is exactly the response being passed through.
	// No replacement headers. No synthetic content length. No content-type guess.
	prolog, err := responseProlog(resp)
	if err != nil {
		return resp, errors.WrapInternalError(
			err,
			errors.RuntimeException,
			"Unable to write response headers",
			err.Error(),
		)
	}

	body, err := WrapResponseBody(
		r.ctx,
		r.rc,
		resp.Body,
		int32(resp.StatusCode),
		resp.Header.Get("Content-Type"),
		contentwriterV1.RecordType_RESPONSE,
		prolog,
	)
	if err != nil {
		return nil, errors.WrapInternalError(
			err,
			errors.RuntimeException,
			"Veidemann proxy lost connection to GRPC services",
			err.Error(),
		)
	}

	// WIN: one body wrapper, one lifecycle owner, one stream.
	resp.Body = body
	return resp, nil
}

func shouldSkipResponseRecording(rc *rpcontext.RecordContext) bool {
	return rc.Error != nil &&
		strings.HasPrefix(rc.Error.Error(), "unknown error from browser controller")
}

func responseProlog(resp *http.Response) ([]byte, error) {
	var prolog bytes.Buffer

	if err := writeResponseProlog(resp, &prolog); err != nil {
		return nil, err
	}

	return prolog.Bytes(), nil
}

func isFromCache(resp *http.Response) bool {
	for _, v := range resp.Header.Values("X-Cache") {
		if strings.Contains(v, "HIT from veidemann_cache") {
			return true
		}
	}

	return false
}
