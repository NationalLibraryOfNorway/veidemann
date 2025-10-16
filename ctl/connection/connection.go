// Copyright © 2017 National Library of Norway.
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
	"crypto/x509"
	"fmt"

	"github.com/NationalLibraryOfNorway/veidemann/ctl/config"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Connect returns a connection to the server.
// If there is an auth provider configured, the connection will be authenticated with the configured credentials.
func Connect() (*grpc.ClientConn, error) {
	ap := config.GetAuthProvider()
	if ap == nil {
		return connect()
	}
	var creds credentials.PerRPCCredentials
	switch ap.Name {
	case config.ProviderApiKey:
		apiKeyConfig, err := config.GetApiKeyConfig()
		if err != nil {
			return nil, err
		}
		creds = apiKeyCredentials{apiKey: apiKeyConfig.ApiKey}
	case config.ProviderOIDC:
		oidcConfig, err := config.GetOIDCConfig()
		if err != nil {
			return nil, err
		}
		creds = oidcCredentials{idToken: oidcConfig.IdToken}
	default:
		return nil, fmt.Errorf("unknown auth provider: %s", ap.Name)
	}

	return connect(grpc.WithPerRPCCredentials(creds))
}

// connect returns a connection to the server.
func connect(opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	dialOptions := append(opts, grpc.WithDefaultCallOptions(grpc.WaitForReady(true)))

	if config.GetInsecure() {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		dialOptions = append(dialOptions, grpc.WithTransportCredentials(clientTransportCredentials()))
	}

	address := config.GetServer()

	return grpc.NewClient(address, dialOptions...)
}

// clientTransportCredentials returns the transport credentials to use for the client
func clientTransportCredentials() credentials.TransportCredentials {
	// Create a pool with systems CAs
	certPool, err := x509.SystemCertPool()
	if err != nil {
		log.Warn().Msg("Could not read system trusted certificates, using only the configured ones")
		certPool = x509.NewCertPool()
	}

	// Add configured CAs
	if config.GetRootCAs() != "" {
		rootCABytes := []byte(config.GetRootCAs())
		if !certPool.AppendCertsFromPEM(rootCABytes) {
			log.Warn().Msg("no certs found in root CA file")
		}
	}

	serverNameOverride := config.GetServerNameOverride()
	log.Debug().Msgf("Using server name override: %s", serverNameOverride)

	return credentials.NewClientTLSFromCert(certPool, serverNameOverride)
}
