// Copyright © 2017 National Library of Norway
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package connection

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	controllerV1 "github.com/NationalLibraryOfNorway/veidemann/api/controller/v1"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

// manualRedirectURI is the redirect URI used when manual login is used.
const manualRedirectURI = "urn:ietf:wg:oauth:2.0:oob"

// Provider is the type of authentication provider.
type Provider string

// Login logs in using the configured authentication provider.
// If manualLogin is true, the user will be given a URL to paste in a browser window,
// else a browser window will be opened automatically.
func Login(manualLogin bool, offlineAccess bool) error {
	p := config.GetAuthProviderName()
	if p == "" {
		p = config.ProviderOIDC
	}
	switch p {
	case config.ProviderOIDC:
		c, err := config.GetOIDCConfig()
		if err != nil {
			return err
		}
		if c == nil {
			c = &config.OIDCConfig{}
		}
		claims, err := loginOIDC(c, manualLogin, offlineAccess)
		if err != nil {
			return err
		}
		if claims == nil {
			return nil
		}
		fmt.Printf("Hello, %s!\n", claims.Name)
		if offlineAccess {
			fmt.Println("Offline access enabled; this context is ready for renewable unattended use.")
		}
	case config.ProviderApiKey:
		// no login procedure for apikey
		if offlineAccess {
			return errors.New("offline login requires an OIDC context; remove the configured API key first")
		}
	}
	return nil
}

// loginOIDC logs in using the OIDC authentication flow.
// If manualLogin is true, the user will be given a URL to paste in a browser window,
// else a browser window will be opened automatically.
func loginOIDC(oidcConfig *config.OIDCConfig, manualLogin bool, offlineAccess bool) (*claims, error) {
	clientID := oidcConfig.ClientID
	if clientID == "" {
		clientID = "veidemann-cli"
	}
	clientSecret := oidcConfig.ClientSecret

	scopes := oidcScopes(oidcConfig.Scopes, offlineAccess)
	idpIssuerUrl := oidcConfig.IdpIssuerUrl
	if idpIssuerUrl == "" {
		idp, err := getIdpIssuer()
		if err != nil {
			return nil, err
		} else if idp == "" {
			return nil, nil
		} else {
			idpIssuerUrl = idp
		}
	}

	slog.Debug("Using identity provider", "issuer", idpIssuerUrl)

	o := oidcProvider{
		idpIssuerUrl: idpIssuerUrl,
		clientID:     clientID,
		clientSecret: clientSecret,
		scopes:       scopes,
	}

	claims, err := o.login(manualLogin)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}
	o.refreshToken, err = refreshTokenForLogin(o.refreshToken, offlineAccess)
	if err != nil {
		return nil, err
	}

	// Set the auth provider in the config
	err = config.SetAuthProvider(&config.AuthProvider{
		Name: config.ProviderOIDC,
		Config: config.OIDCConfig{
			ClientID:     o.clientID,
			ClientSecret: o.clientSecret,
			Scopes:       oidcConfig.Scopes,
			IdToken:      o.idToken,
			RefreshToken: o.refreshToken,
			IdpIssuerUrl: o.idpIssuerUrl,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to save auth provider: %w", err)
	}

	return claims, nil
}

func refreshTokenForLogin(refreshToken string, offlineAccess bool) (string, error) {
	if offlineAccess && refreshToken == "" {
		return "", errors.New("login failed: the identity provider did not issue a refresh token for offline access")
	}
	if !offlineAccess {
		return "", nil
	}
	return refreshToken, nil
}

func oidcScopes(configured []string, offlineAccess bool) []string {
	scopes := configured
	if scopes == nil {
		scopes = []string{oidc.ScopeOpenID, "profile", "email", "groups", "audience:server:client_id:veidemann-api"}
	}
	result := make([]string, 0, len(scopes)+1)
	for _, scope := range scopes {
		if scope != "offline_access" {
			result = append(result, scope)
		}
	}
	if offlineAccess {
		result = append(result, "offline_access")
	}
	return result
}

// Logout removes the auth provider from the config. Effectively logging out.
func Logout() error {
	return config.SetAuthProvider(nil)
}

// getIdpIssuer resolves the OIDC issuer from the server.
func getIdpIssuer() (string, error) {
	conn, err := connect()
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reply, err := controllerV1.NewControllerClient(conn).GetOpenIdConnectIssuer(ctx, &empty.Empty{})
	if err != nil {
		return "", fmt.Errorf("failed to get oidc issuer: %w", err)
	}

	idp := reply.GetOpenIdConnectIssuer()
	if idp == "" {
		slog.Warn("Server is configured without an identity provider - proceeding without authentication.")
	} else {
		slog.Debug("Using idp issuer address", "issuer", idp)
	}

	return idp, nil
}

// oidcProvider implements the oidc authentication flow.
type oidcProvider struct {
	clientID     string
	clientSecret string
	idToken      string
	refreshToken string
	idpIssuerUrl string
	scopes       []string
}

// Login using OIDC Authorization Code flow with PKCE.
// If manual is true, the user will be given a URL to paste in a browser window,
// else a browser window will be opened automatically.
func (op *oidcProvider) login(manual bool) (*claims, error) {
	// get http client with configured CAs
	client := httpClientForRootCAs()
	if client == nil {
		client = http.DefaultClient
	}

	// initialize OIDC ID Token verifier
	var idTokenVerifier *oidc.IDTokenVerifier
	ctx := oidc.ClientContext(context.Background(), client)
	discoveryCtx, cancelDiscovery := context.WithTimeout(ctx, 30*time.Second)
	p, err := oidc.NewProvider(discoveryCtx, op.idpIssuerUrl)
	cancelDiscovery()
	if err != nil {
		return nil, fmt.Errorf("could not connect to identity provider \"%s\": %w", op.idpIssuerUrl, err)
	}
	oc := oidc.Config{
		ClientID: op.clientID,
	}
	idTokenVerifier = p.Verifier(&oc)

	var redirectURI string
	var loopback *loopbackServer
	if manual {
		redirectURI = manualRedirectURI
	} else {
		loopback, err = newLoopbackServer()
		if err != nil {
			return nil, fmt.Errorf("could not start login callback server: %w", err)
		}
		defer loopback.Close()
		redirectURI = loopback.RedirectURI()
	}

	// Authorization code flow with PKCE
	oauth2Config := &oauth2.Config{
		ClientID:     op.clientID,
		ClientSecret: op.clientSecret,
		Endpoint:     p.Endpoint(),
		Scopes:       op.scopes,
		RedirectURL:  redirectURI,
	}

	// PKCE requires a code verifier and a code challenge.
	codeVerifier := oauth2.GenerateVerifier()

	nonce, err := secureRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	state, err := secureRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("generate state: %w", err)
	}
	authCodeURL := oauth2Config.AuthCodeURL(state,
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(codeVerifier),
	)

	var code string

	if manual {
		code, err = manualFlow(authCodeURL)
	} else {
		loginCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		code, err = openBrowserFlow(loginCtx, authCodeURL, state, loopback)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization code: %w", err)
	}

	exchangeCtx, cancelExchange := context.WithTimeout(ctx, 30*time.Second)
	defer cancelExchange()
	oauth2Token, err := oauth2Config.Exchange(exchangeCtx, code, oauth2.VerifierOption(codeVerifier))
	if err != nil {
		return nil, err
	}
	op.refreshToken = oauth2Token.RefreshToken

	// Extract the ID Token from OAuth2 token.
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("token not found")
	}
	op.idToken = rawIDToken

	// Verify ID Token
	idToken, err := idTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, err
	}

	// Verify nonce
	if idToken.Nonce != nonce {
		return nil, errors.New("nonce did not match")
	}

	claims := new(claims)
	if err := idToken.Claims(claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func secureRandomString(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func manualFlow(authCodeURL string) (string, error) {
	var code string
	fmt.Printf("Open the following link in your browser:\n%v\n", authCodeURL)
	fmt.Printf("Paste the code from the browser: ")
	if _, err := fmt.Scan(&code); err != nil {
		return "", err
	}
	return code, nil
}

func openBrowserFlow(ctx context.Context, authCodeURL string, state string, server *loopbackServer) (string, error) {
	result := server.Wait(ctx, state)
	err := openBrowser(authCodeURL)
	if err != nil {
		return "", err
	}
	return result()
}

// claims represent custom claims.
type claims struct {
	Email    string   `json:"email"`
	Verified bool     `json:"email_verified"`
	Groups   []string `json:"groups"`
	Name     string   `json:"name"`
}

// openBrowser tries to open the URL in a browser.
func openBrowser(authCodeURL string) error {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", authCodeURL).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", authCodeURL).Start()
	case "darwin":
		err = exec.Command("open", authCodeURL).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}

	return nil
}

// httpClientForRootCAs returns an HTTP client that trusts the provided root CAs.
func httpClientForRootCAs() *http.Client {
	// Create a certificate pool with systems CAs
	certPool, err := x509.SystemCertPool()
	if err != nil {
		slog.Warn("Could not read system trusted certificates, using only the configured ones")
		certPool = x509.NewCertPool()
	}
	tlsConfig := tls.Config{RootCAs: certPool}

	// Add CAs from config
	if config.GetRootCAs() != "" {
		rootCABytes := []byte(config.GetRootCAs())
		if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCABytes) {
			slog.Warn("No certs found in root CA file")
		}
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

// apiKeyCredentials implements credentials.PerRPCCredentials for apikey authentication.
type apiKeyCredentials struct {
	apiKey string
}

// GetRequestMetadata implements PerRPCCredentials
func (a apiKeyCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "ApiKey" + " " + a.apiKey,
	}, nil
}

// RequireTransportSecurity implements PerRPCCredentials
func (a apiKeyCredentials) RequireTransportSecurity() bool {
	return true
}
