/*
 * Copyright 2019 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package serviceconnections

import (
	"errors"
	"fmt"

	browsercontrollerV2 "github.com/NationalLibraryOfNorway/veidemann/api/browsercontroller/v2"
	contentwriterV1 "github.com/NationalLibraryOfNorway/veidemann/api/contentwriter/v1"
	dnsresolverV1 "github.com/NationalLibraryOfNorway/veidemann/api/dnsresolver/v1"
	"github.com/NationalLibraryOfNorway/veidemann/recorderproxy/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Connections holds the clients for external grpc services
type Connections struct {
	contentWriterOptions        *ConnectionOptions
	dnsOptions                  *ConnectionOptions
	browserControllerOptions    *ConnectionOptions
	contentWriterClientConn     *grpc.ClientConn
	contentWriterClient         contentwriterV1.ContentWriterClient
	dnsResolverClientConn       *grpc.ClientConn
	dnsResolverClient           dnsresolverV1.DnsResolverClient
	browserControllerClientConn *grpc.ClientConn
	browserControllerClient     browsercontrollerV2.BrowserControllerClient
}

func NewConnections(contentWriterOptions, dnsOptions, browserControllerOptions *ConnectionOptions) *Connections {
	return &Connections{
		contentWriterOptions:     contentWriterOptions,
		dnsOptions:               dnsOptions,
		browserControllerOptions: browserControllerOptions,
	}
}

func (c *Connections) Connect() error {
	var err error

	// Set up ContentWriterClient
	c.contentWriterClientConn, err = c.contentWriterOptions.connectService()
	if err != nil {
		return err
	}
	c.contentWriterClient = contentwriterV1.NewContentWriterClient(c.contentWriterClientConn)
	logger.LogWithComponent("gRPC:CWR").Print("Connected to contentwriter")

	// Set up DnsResolverClient
	c.dnsResolverClientConn, err = c.dnsOptions.connectService()
	if err != nil {
		return err
	}
	c.dnsResolverClient = dnsresolverV1.NewDnsResolverClient(c.dnsResolverClientConn)
	logger.LogWithComponent("gRPC:DNS").Print("Connected to dns resolver")

	// Set up BrowserControllerClient
	c.browserControllerClientConn, err = c.browserControllerOptions.connectService()
	if err != nil {
		return err
	}
	c.browserControllerClient = browsercontrollerV2.NewBrowserControllerClient(c.browserControllerClientConn)
	logger.LogWithComponent("gRPC:BC").Print("Connected to browser controller")

	return nil
}

func (opts *ConnectionOptions) connectService() (*grpc.ClientConn, error) {
	logger.LogWithComponent("PROXY").Printf("Connecting %s at: %s", opts.serviceName, opts.Addr())

	dialOpts := append(opts.dialOptions,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	clientConn, err := grpc.NewClient(opts.Addr(), dialOpts...)
	if err != nil {
		logger.LogWithComponent("PROXY").Errorf("fail to dial %s: %v", opts.serviceName, err)
	}

	return clientConn, err
}

func (c *Connections) Close() error {
	cwErr := c.contentWriterClientConn.Close()
	if cwErr != nil {
		cwErr = fmt.Errorf("failed to close content writer connection: %w", cwErr)
	}
	dnsErr := c.dnsResolverClientConn.Close()
	if dnsErr != nil {
		dnsErr = fmt.Errorf("failed to close dns resolver connection: %w", dnsErr)
	}
	bcErr := c.browserControllerClientConn.Close()
	if bcErr != nil {
		bcErr = fmt.Errorf("failed to close browser controller connection: %w", bcErr)
	}

	return errors.Join(cwErr, dnsErr, bcErr)
}

func (c *Connections) ContentWriterClient() contentwriterV1.ContentWriterClient {
	return c.contentWriterClient
}

func (c *Connections) DnsResolverClient() dnsresolverV1.DnsResolverClient {
	return c.dnsResolverClient
}

func (c *Connections) BrowserControllerClient() browsercontrollerV2.BrowserControllerClient {
	return c.browserControllerClient
}
