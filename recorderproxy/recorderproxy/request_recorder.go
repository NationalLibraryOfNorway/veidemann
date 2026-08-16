package recorderproxy

import (
	"bytes"
	"context"
	"net/http"

	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	logV1 "github.com/NationalLibraryOfNorway/veidemann/api/log/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/constants"
	rpcontext "github.com/NationalLibraryOfNorway/veidemann/recorderproxy/context"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/errors"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type requestRecorder struct {
	ctx context.Context
	rc  *rpcontext.RecordContext
}

func newRequestRecorder(ctx context.Context, rc *rpcontext.RecordContext) requestRecorder {
	return requestRecorder{
		ctx: ctx,
		rc:  rc,
	}
}

func (r requestRecorder) Wrap(req *http.Request) (*http.Request, error) {
	if req == nil {
		return nil, http.ErrAbortHandler
	}

	// WIN: request setup is outside RecorderFilter.
	// The filter remains chain orchestration, not recording bookkeeping.
	prolog, err := requestProlog(req)
	if err != nil {
		return req, errors.WrapInternalError(
			err,
			errors.RuntimeException,
			"Unable to write request headers",
			err.Error(),
		)
	}

	r.prepareRequest(req)
	r.prepareMeta()

	body, err := WrapRequestBody(
		r.ctx,
		r.rc,
		req.Body,
		req.Header.Get("Content-Type"),
		prolog,
	)
	if err != nil {
		return req, errors.WrapInternalError(
			err,
			errors.RuntimeException,
			"Veidemann proxy lost connection to GRPC services",
			err.Error(),
		)
	}

	req.Body = body
	return req, nil
}

func (r requestRecorder) prepareRequest(req *http.Request) {
	req.Header.Set(constants.HeaderAcceptEncoding, "identity")
	req.Header.Set(constants.HeaderCrawlExecutionId, r.rc.CrawlExecutionId)
	req.Header.Set(constants.HeaderJobExecutionId, r.rc.JobExecutionId)
}

func (r requestRecorder) prepareMeta() {
	r.rc.IpAddress = rpcontext.GetIp(r.ctx)

	r.rc.Meta = &contentwriterV1.WriteRequest_Meta{
		Meta: &contentwriterV1.WriteRequestMeta{
			RecordMeta:     map[int32]*contentwriterV1.WriteRequestMeta_RecordMeta{},
			TargetUri:      r.rc.Uri.String(),
			ExecutionId:    r.rc.CrawlExecutionId,
			IpAddress:      r.rc.IpAddress,
			CollectionRef:  r.rc.CollectionRef,
			FetchTimeStamp: timestamppb.New(r.rc.FetchTimesTamp),
		},
	}

	r.rc.UpdateCrawlLog(func(cl *logV1.CrawlLog) {
		cl.RequestedUri = r.rc.Uri.String()
	})
}

func requestProlog(req *http.Request) ([]byte, error) {
	var prolog bytes.Buffer

	if err := writeRequestProlog(req, &prolog); err != nil {
		return nil, err
	}

	return prolog.Bytes(), nil
}
