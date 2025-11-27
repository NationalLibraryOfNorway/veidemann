// Copyright 2018 National Library of Norway
//
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

// Package aggregator contains an aggregator service client
package frontier

import (
	"context"

	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Client struct {
	frontierV1.FrontierClient
}

func New(conn *grpc.ClientConn) *Client {
	return &Client{
		FrontierClient: frontierV1.NewFrontierClient(conn),
	}
}

func (f *Client) QueueCountTotal(ctx context.Context) (int64, error) {
	res, err := f.FrontierClient.QueueCountTotal(ctx, &emptypb.Empty{})
	if err != nil {
		return 0, err
	}
	return res.GetCount(), nil
}
