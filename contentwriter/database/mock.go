/*
 * Copyright 2021 National Library of Norway.
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

package database

import (
	"context"
	"time"

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config"
	"github.com/NationalLibraryOfNorway/veidemann/api/contentwriter"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

type MockConnection struct {
	*RethinkDbConnection
}

// NewMockConnection creates a new mocked RethinkDbConnection object
func NewMockConnection() *MockConnection {
	return &MockConnection{
		RethinkDbConnection: &RethinkDbConnection{
			connectOpts: r.ConnectOpts{
				NumRetries: 10,
			},
			session:      r.NewMock(),
			batchSize:    200,
			queryTimeout: 5 * time.Second,
		},
	}
}

func (c *MockConnection) Close() error {
	return nil
}

func (c *MockConnection) GetMock() *r.Mock {
	return c.session.(*r.Mock)
}

func (c *MockConnection) GetConfigObject(ctx context.Context, ref *configV1.ConfigRef) (*configV1.ConfigObject, error) {
	return c.RethinkDbConnection.GetConfigObject(ctx, ref)
}

func (c *MockConnection) HasCrawledContent(ctx context.Context, revisitKey string) (*contentwriter.CrawledContent, error) {
	return c.RethinkDbConnection.HasCrawledContent(ctx, revisitKey)
}

func (c *MockConnection) WriteCrawledContent(ctx context.Context, crawledContent *contentwriter.CrawledContent) error {
	return c.RethinkDbConnection.WriteCrawledContent(ctx, crawledContent)
}
