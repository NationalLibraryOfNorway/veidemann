package connection

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

const (
	refreshBeforeExpiry = time.Minute
	refreshTimeout      = 30 * time.Second
)

// oidcCredentials refreshes and rotates OIDC credentials as needed. A single
// instance lives for the lifetime of a gRPC connection, so long-running ctl
// commands can refresh between RPCs.
type oidcCredentials struct {
	mu sync.Mutex

	clientID     string
	clientSecret string
	issuerURL    string
	idToken      string
	refreshToken string
	expiry       time.Time

	httpClient   *http.Client
	verifier     *oidc.IDTokenVerifier
	expiryReader *oidc.IDTokenVerifier
	oauthConfig  *oauth2.Config
}

func newOIDCCredentials(oidcConfig *config.OIDCConfig) (*oidcCredentials, error) {
	if oidcConfig == nil || oidcConfig.IdToken == "" {
		return nil, errors.New("OIDC session is missing an ID token; run veidemannctl login")
	}
	clientID := oidcConfig.ClientID
	if clientID == "" {
		clientID = "veidemann-cli"
	}
	if oidcConfig.IdpIssuerUrl == "" {
		return nil, errors.New("OIDC session is missing an issuer URL; run veidemannctl login")
	}

	httpClient := httpClientForRootCAs()
	ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
	defer cancel()
	providerCtx := oidc.ClientContext(ctx, httpClient)
	provider, err := oidc.NewProvider(providerCtx, oidcConfig.IdpIssuerUrl)
	if err != nil {
		return nil, fmt.Errorf("discover OIDC provider: %w", err)
	}

	verifier := provider.VerifierContext(providerCtx, &oidc.Config{ClientID: clientID})
	expiryReader := provider.VerifierContext(providerCtx, &oidc.Config{
		ClientID:        clientID,
		SkipExpiryCheck: true,
	})
	parsed, err := expiryReader.Verify(providerCtx, oidcConfig.IdToken)
	if err != nil && oidcConfig.RefreshToken == "" {
		return nil, fmt.Errorf("validate stored ID token: %w", err)
	}
	var expiry time.Time
	if parsed != nil {
		expiry = parsed.Expiry
	}

	return &oidcCredentials{
		clientID:     clientID,
		clientSecret: oidcConfig.ClientSecret,
		issuerURL:    oidcConfig.IdpIssuerUrl,
		idToken:      oidcConfig.IdToken,
		refreshToken: oidcConfig.RefreshToken,
		expiry:       expiry,
		httpClient:   httpClient,
		verifier:     verifier,
		expiryReader: expiryReader,
		oauthConfig: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: oidcConfig.ClientSecret,
			Endpoint:     provider.Endpoint(),
		},
	}, nil
}

func (oc *oidcCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	oc.mu.Lock()
	defer oc.mu.Unlock()

	if !oc.expiry.After(time.Now().Add(refreshBeforeExpiry)) {
		if err := oc.refresh(ctx); err != nil {
			return nil, err
		}
	}
	return map[string]string{"authorization": "Bearer " + oc.idToken}, nil
}

func (oc *oidcCredentials) RequireTransportSecurity() bool {
	return true
}

func (oc *oidcCredentials) refresh(parent context.Context) error {
	if oc.refreshToken == "" {
		return errors.New("OIDC session expired and is not renewable; run veidemannctl login --offline")
	}

	ctx, cancel := context.WithTimeout(parent, refreshTimeout)
	defer cancel()

	var nextIDToken string
	var nextRefreshToken string
	var nextExpiry time.Time
	err := config.UpdateAuthProvider(ctx, func(current *config.AuthProvider) (*config.AuthProvider, error) {
		if current == nil || current.Name != config.ProviderOIDC {
			return nil, errors.New("OIDC session changed while waiting to refresh; retry the command")
		}
		currentConfig, err := config.GetOIDCConfig()
		if err != nil {
			return nil, fmt.Errorf("decode current OIDC session: %w", err)
		}
		if currentConfig == nil || currentConfig.IdpIssuerUrl != oc.issuerURL ||
			normalizedClientID(currentConfig.ClientID) != oc.clientID || currentConfig.ClientSecret != oc.clientSecret {
			return nil, errors.New("OIDC provider changed while waiting to refresh; retry the command")
		}

		currentToken, validationErr := oc.expiryReader.Verify(oidc.ClientContext(ctx, oc.httpClient), currentConfig.IdToken)
		if validationErr == nil && currentToken.Expiry.After(time.Now().Add(refreshBeforeExpiry)) {
			nextIDToken = currentConfig.IdToken
			nextRefreshToken = currentConfig.RefreshToken
			nextExpiry = currentToken.Expiry
			return current, nil
		}
		if currentConfig.RefreshToken == "" {
			return nil, errors.New("OIDC session expired and is not renewable; run veidemannctl login --offline")
		}

		tokenSource := oc.oauthConfig.TokenSource(
			oidc.ClientContext(ctx, oc.httpClient),
			&oauth2.Token{RefreshToken: currentConfig.RefreshToken},
		)
		refreshed, err := tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("refresh OIDC session; run veidemannctl login --offline if the session is no longer valid: %w", err)
		}
		rawIDToken, ok := refreshed.Extra("id_token").(string)
		if !ok || rawIDToken == "" {
			return nil, errors.New("identity provider refresh response did not contain an ID token; run veidemannctl login --offline")
		}
		verified, err := oc.verifier.Verify(oidc.ClientContext(ctx, oc.httpClient), rawIDToken)
		if err != nil {
			return nil, fmt.Errorf("validate refreshed ID token: %w", err)
		}

		rotatedRefreshToken := refreshed.RefreshToken
		if rotatedRefreshToken == "" {
			rotatedRefreshToken = currentConfig.RefreshToken
		}
		updated := *currentConfig
		updated.IdToken = rawIDToken
		updated.RefreshToken = rotatedRefreshToken
		nextIDToken = rawIDToken
		nextRefreshToken = rotatedRefreshToken
		nextExpiry = verified.Expiry
		return &config.AuthProvider{Name: config.ProviderOIDC, Config: updated}, nil
	})
	if err != nil {
		return err
	}

	oc.idToken = nextIDToken
	oc.refreshToken = nextRefreshToken
	oc.expiry = nextExpiry
	return nil
}

func normalizedClientID(clientID string) string {
	if clientID == "" {
		return "veidemann-cli"
	}
	return clientID
}
