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
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	controllerV1 "github.com/NationalLibraryOfNorway/veidemann/api/controller"
	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

const (
	// manualRedirectURI is the redirect URI used when manual login is used.
	manualRedirectURI = "urn:ietf:wg:oauth:2.0:oob"
	// autoRedirectURI is the redirect URI used when automatic login is used.
	autoRedirectURI = "http://localhost:9876"
)

// Provider is the type of authentication provider.
type Provider string

// Login logs in using the configured authentication provider.
// If manualLogin is true, the user will be given a URL to paste in a browser window,
// else a browser window will be opened automatically.
func Login(manualLogin bool) error {
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
		claims, err := loginOIDC(c, manualLogin)
		if err != nil {
			return err
		}
		if claims == nil {
			return nil
		}
		fmt.Printf("Hello, %s!\n", claims.Name)
	case config.ProviderApiKey:
		// no login procedure for apikey
	}
	return nil
}

// loginOIDC logs in using the OIDC authentication flow.
// If manualLogin is true, the user will be given a URL to paste in a browser window,
// else a browser window will be opened automatically.
func loginOIDC(oidcConfig *config.OIDCConfig, manualLogin bool) (*claims, error) {
	clientID := oidcConfig.ClientID
	if clientID == "" {
		clientID = "veidemann-cli"
	}
	clientSecret := oidcConfig.ClientSecret

	// Does the provider use "offline_access" scope to request a refresh token
	// or does it use "access_type=offline" (e.g. Google)?
	offlineAsScope := false
	scopes := []string{oidc.ScopeOpenID, "profile", "email", "groups", "audience:server:client_id:veidemann-api"}
	if offlineAsScope {
		scopes = append(scopes, "offline_access")
	}
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

	log.Debug().Msgf("Using identity provider: %s", idpIssuerUrl)

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

	// Set the auth provider in the config
	err = config.SetAuthProvider(&config.AuthProvider{
		Name: config.ProviderOIDC,
		Config: config.OIDCConfig{
			ClientID:     o.clientID,
			ClientSecret: o.clientSecret,
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
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	reply, err := controllerV1.NewControllerClient(conn).GetOpenIdConnectIssuer(ctx, &empty.Empty{})
	if err != nil {
		return "", fmt.Errorf("failed to get oidc issuer: %w", err)
	}

	idp := reply.GetOpenIdConnectIssuer()
	if idp == "" {
		log.Warn().Msg("Server is configured without an identity provider - proceeding without authentication.")
	} else {
		log.Debug().Msgf(`Using idp issuer address : "%v"`, idp)
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
	p, err := oidc.NewProvider(ctx, op.idpIssuerUrl)
	if err != nil {
		return nil, fmt.Errorf("could not connect to identity provider \"%s\": %w", op.idpIssuerUrl, err)
	}
	oc := oidc.Config{
		ClientID: op.clientID,
	}
	idTokenVerifier = p.Verifier(&oc)

	var redirectURI string
	if manual {
		redirectURI = manualRedirectURI
	} else {
		redirectURI = autoRedirectURI
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
	codeVerifier, codeChallenge := pkceChallenge()

	nonce := randStringBytesMaskImprSrc(16)
	state := randStringBytesMaskImprSrc(16)
	authCodeURL := oauth2Config.AuthCodeURL(state,
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)

	var code string

	if manual {
		code, err = manualFlow(authCodeURL)
	} else {
		code, err = openBrowserFlow(authCodeURL, state)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get authorization code: %w", err)
	}

	oauth2Token, err := oauth2Config.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
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

// pkceChallenge generates a code verifier and a code challenge.
func pkceChallenge() (codeVerifier string, codeChallenge string) {
	codeVerifier = randStringBytesMaskImprSrc(64)
	// sha256 hash code verifier
	h := sha256.New()
	h.Write([]byte(codeVerifier))
	codeChallenge = base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return
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

func openBrowserFlow(authCodeURL string, state string) (string, error) {
	err := openBrowser(authCodeURL)
	if err != nil {
		return "", err
	}

	code, gotState, err := listenAndWaitForAuthorizationCode(autoRedirectURI)
	if err != nil {
		return "", err
	}
	if gotState != state {
		return "", fmt.Errorf("state did not match")
	}
	return code, nil
}

// oidcCredentials implements credentials.PerRPCCredentials for oidc authentication.
type oidcCredentials struct {
	idToken string
}

// GetRequestMetadata implements credentials.PerRPCCredentials.
func (oc oidcCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer" + " " + oc.idToken,
	}, nil
}

// RequireTransportSecurity implements credentials.PerRPCCredentials.
func (oc oidcCredentials) RequireTransportSecurity() bool {
	return true
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
		log.Warn().Msg("Could not read system trusted certificates, using only the configured ones")
		certPool = x509.NewCertPool()
	}
	tlsConfig := tls.Config{RootCAs: certPool}

	// Add CAs from config
	if config.GetRootCAs() != "" {
		rootCABytes := []byte(config.GetRootCAs())
		if !tlsConfig.RootCAs.AppendCertsFromPEM(rootCABytes) {
			log.Warn().Msg("No certs found in root CA file")
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

// letterBytes is used to generate a random string.
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// src is a source of random numbers
var src = rand.NewSource(time.Now().UnixNano())

// randStringBytesMaskImprSrc generates a random string of n characters.
func randStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
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
