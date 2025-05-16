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

package scylla

import (
	"log"
	"time"

	"github.com/gocql/gocql"
)

type Client struct {
	*gocql.Session
	config *gocql.ClusterConfig
}

func NewClient(hosts ...string) *Client {
	return &Client{config: createCluster(hosts...)}
}

func createCluster(hosts ...string) *gocql.ClusterConfig {
	retryPolicy := &gocql.ExponentialBackoffRetryPolicy{
		Min:        time.Second,
		Max:        10 * time.Second,
		NumRetries: 5,
	}
	cluster := gocql.NewCluster(hosts...)
	cluster.Timeout = 5 * time.Second
	cluster.RetryPolicy = retryPolicy
	cluster.Consistency = gocql.LocalQuorum
	cluster.PoolConfig.HostSelectionPolicy = gocql.TokenAwareHostPolicy(gocql.RoundRobinHostPolicy())
	return cluster
}

// Connect establishes a connection to a ScyllaDB cluster.
func (c *Client) Connect() error {
	sess, err := gocql.NewSession(*c.config)
	if err != nil {
		return err
	}
	c.Session = sess
	return nil
}

// Disconnect closes the connection with the scylla cluster
func (c *Client) Disconnect() {
	c.Close()
}

func (c *Client) Drop(keyspace string) {
	log.Printf("Dropping scylla keyspace: %v", keyspace)
	err := c.Query("drop keyspace " + keyspace).Exec()
	if err != nil {
		log.Println(err)
	}
}
