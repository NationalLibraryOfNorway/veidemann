package service

import (
	"context"
	"strconv"

	scopecheckerV1 "github.com/NationalLibraryOfNorway/veidemann/api/scopechecker/v1"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/internal/script"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/internal/telemetry"
	"github.com/prometheus/client_golang/prometheus"
)

type ScopeChecker struct {
	scopecheckerV1.UnimplementedScopesCheckerServiceServer
}

func (s ScopeChecker) ScopeCheck(_ context.Context, request *scopecheckerV1.ScopeCheckRequest) (*scopecheckerV1.ScopeCheckResponse, error) {
	telemetry.ScopechecksTotal.Inc()

	result := script.RunScopeScript(request.ScopeScriptName, request.ScopeScript, request.QueuedUri, request.Debug)

	telemetry.ScopecheckResponseTotal.With(prometheus.Labels{"code": strconv.Itoa(int(result.ExcludeReason))}).Inc()

	return result, nil
}
