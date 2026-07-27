package service

import (
	"context"

	commonsV1 "github.com/NationalLibraryOfNorway/veidemann/api/commons/v1"
	uricanonicalizerV1 "github.com/NationalLibraryOfNorway/veidemann/api/uricanonicalizer/v1"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/internal/script"
	"github.com/NationalLibraryOfNorway/veidemann/scopeservice/internal/telemetry"
)

type UriCanonicalizer struct {
	uricanonicalizerV1.UnimplementedUriCanonicalizerServiceServer
}

func (u UriCanonicalizer) Canonicalize(_ context.Context, request *uricanonicalizerV1.CanonicalizeRequest) (*uricanonicalizerV1.CanonicalizeResponse, error) {
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
