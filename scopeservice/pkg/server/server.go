package server

import (
	"context"
	"fmt"
	"net"
	"strconv"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	scopecheckerV1 "github.com/NationalLibraryOfNorway/veidemann/api/scopechecker/v1"
	uricanonicalizerV1 "github.com/NationalLibraryOfNorway/veidemann/api/uricanonicalizer/v1"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/pkg/script"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/pkg/telemetry"
	otgrpc "github.com/opentracing-contrib/go-grpc"
	"github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
)

type GrpcServer struct {
	listenHost string
	listenPort int
	grpcServer *grpc.Server
}

func New(host string, port int) *GrpcServer {
	s := &GrpcServer{
		listenHost: host,
		listenPort: port,
	}
	return s
}

func (s *GrpcServer) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.listenHost, s.listenPort))
	if err != nil {
		log.Fatal().Msgf("failed to listen: %v", err)
	}

	tracer := opentracing.GlobalTracer()
	var opts = []grpc.ServerOption{
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)),
	}
	s.grpcServer = grpc.NewServer(opts...)
	scopecheckerV1.RegisterScopesCheckerServiceServer(s.grpcServer, &ScopeCheckerService{})
	uricanonicalizerV1.RegisterUriCanonicalizerServiceServer(s.grpcServer, &UriCanonicalizerService{})

	log.Info().Msgf("Scope Service listening on %s", lis.Addr())
	return s.grpcServer.Serve(lis)
}

func (s *GrpcServer) Shutdown() {
	log.Info().Msg("Shutting down Scope Service")
	s.grpcServer.GracefulStop()
}

type ScopeCheckerService struct {
	scopecheckerV1.UnimplementedScopesCheckerServiceServer
}

func (s *ScopeCheckerService) ScopeCheck(_ context.Context, request *scopecheckerV1.ScopeCheckRequest) (*scopecheckerV1.ScopeCheckResponse, error) {
	telemetry.ScopechecksTotal.Inc()
	result := script.RunScopeScript(request.ScopeScriptName, request.ScopeScript, request.QueuedUri, request.Debug)
	telemetry.ScopecheckResponseTotal.With(prometheus.Labels{"code": strconv.Itoa(int(result.ExcludeReason))}).Inc()
	return result, nil
}

type UriCanonicalizerService struct {
	uricanonicalizerV1.UnimplementedUriCanonicalizerServiceServer
}

func (u *UriCanonicalizerService) Canonicalize(_ context.Context, request *uricanonicalizerV1.CanonicalizeRequest) (*uricanonicalizerV1.CanonicalizeResponse, error) {
	telemetry.CanonicalizationsTotal.Inc()
	canonicalized, err := script.CrawlCanonicalizationProfile.Parse(request.Uri)
	if err == nil {
		return &uricanonicalizerV1.CanonicalizeResponse{
			Uri: &commonsV1.ParsedUri{
				Href:     canonicalized.String(),
				Scheme:   canonicalized.Scheme(),
				Host:     canonicalized.Hostname(),
				Port:     int32(canonicalized.DecodedPort()),
				Username: canonicalized.Username(),
				Password: canonicalized.Password(),
				Path:     canonicalized.Pathname(),
				Query:    canonicalized.Query(),
				Fragment: canonicalized.Fragment(),
			},
		}, nil
	}
	return &uricanonicalizerV1.CanonicalizeResponse{
		Uri: &commonsV1.ParsedUri{
			Href: request.Uri},
	}, err
}
