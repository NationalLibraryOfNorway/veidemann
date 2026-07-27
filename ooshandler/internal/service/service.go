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

package service

import (
	"context"

	ooshandlerV1 "github.com/NationalLibraryOfNorway/veidemann/api/ooshandler/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewOutOfScopeHandler(oosHandler *Handler) *OOSHandler {
	return &OOSHandler{
		oosHandler: oosHandler,
	}
}

// OosService is a service which handles Out of Scope URIs.
type OOSHandler struct {
	ooshandlerV1.UnimplementedOosHandlerServer
	oosHandler *Handler
}

func (o *OOSHandler) SubmitUri(ctx context.Context, req *ooshandlerV1.SubmitUriRequest) (*emptypb.Empty, error) {
	OosRequests.Inc()
	exists := o.oosHandler.Handle(req.GetUri().GetUri())
	if exists {
		OosDuplicate.Inc()
	}
	return &emptypb.Empty{}, nil
}
