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

package oos

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"

	"github.com/NationalLibraryOfNorway/veidemann/api/ooshandler"
	"github.com/NationalLibraryOfNorway/veidemann/ooshandler/metrics"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// OosService is a service which handles Out of Scope URIs.
type Service struct {
	ooshandler.UnimplementedOosHandlerServer
	Port       int
	ln         net.Listener
	listenAddr net.Addr
	lnSetup    bool
	addr       string
	oosHandler *Handler
}

func (o *Service) SubmitUri(ctx context.Context, req *ooshandler.SubmitUriRequest) (*emptypb.Empty, error) {
	metrics.OosRequests.Inc()
	exists := o.oosHandler.Handle(req.GetUri().GetUri())
	if exists {
		metrics.OosDuplicate.Inc()
	}
	return &emptypb.Empty{}, nil
}

// NewOosService returns a new instance of OosService listening on the given port
func NewService(port int, oosHandler *Handler) *Service {
	met := &Service{
		Port:       port,
		addr:       fmt.Sprintf("0.0.0.0:%d", port),
		oosHandler: oosHandler,
	}

	return met
}

func (o *Service) Start() error {
	ln, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", o.Port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	o.ln = ln
	o.listenAddr = ln.Addr()
	o.lnSetup = true

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	ooshandler.RegisterOosHandlerServer(grpcServer, o)

	go func() {
		err := grpcServer.Serve(ln)
		if err != nil {
			slog.Error("Failed to serve", "err", err)
			os.Exit(1)
		}
	}()
	return nil
}
