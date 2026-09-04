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
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// RethinkDbConnection holds the database connection
type RethinkDbConnection struct {
	connectOpts  r.ConnectOpts
	session      r.QueryExecutor
	maxRetries   int
	waitTimeout  time.Duration
	queryTimeout time.Duration
	batchSize    int
	logger       *slog.Logger
}

type RethinkDbOptions struct {
	Username           string
	Password           string
	Database           string
	UseOpenTracing     bool
	Address            string
	QueryTimeout       time.Duration
	MaxRetries         int
	MaxOpenConnections int
}

type retryAction int

const (
	doNotRetry retryAction = iota
	waitForWrites
	reconnect
)

// NewRethinkDbConnection creates a new RethinkDbConnection object
func NewRethinkDbConnection(opts RethinkDbOptions) *RethinkDbConnection {
	return &RethinkDbConnection{
		connectOpts: r.ConnectOpts{
			Address:        opts.Address,
			Username:       opts.Username,
			Password:       opts.Password,
			Database:       opts.Database,
			InitialCap:     2,
			MaxOpen:        opts.MaxOpenConnections,
			UseOpentracing: opts.UseOpenTracing,
			NumRetries:     10,
			Timeout:        10 * time.Second,
		},
		maxRetries:   opts.MaxRetries,
		waitTimeout:  60 * time.Second,
		queryTimeout: opts.QueryTimeout,
		batchSize:    200,
		logger:       slog.Default().With("component", "rethinkdb"),
	}
}

func (c *RethinkDbConnection) loggerOrDefault() *slog.Logger {
	if c == nil || c.logger == nil {
		return slog.Default().With("component", "rethinkdb")
	}
	return c.logger
}

// Connect establishes connections
func (c *RethinkDbConnection) Connect() error {
	log := c.loggerOrDefault()
	var err error
	// Set up database RethinkDbConnection
	c.session, err = r.Connect(c.connectOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to RethinkDB at %s: %w", c.connectOpts.Address, err)
	}
	log.Info("Connected to RethinkDB", "address", c.connectOpts.Address)
	return nil
}

// Close closes the RethinkDbConnection
func (c *RethinkDbConnection) Close() error {
	log := c.loggerOrDefault()
	log.Info("Closing connection to RethinkDB")
	return c.session.(*r.Session).Close()
}

// execWrite executes the given write term with a timeout
func (c *RethinkDbConnection) execWrite(ctx context.Context, name string, term *r.Term) (writeResponse r.WriteResponse, err error) {
	q := func(ctx context.Context) (*r.Cursor, error) {
		runOpts := r.RunOpts{
			Context:    ctx,
			Durability: "soft",
		}
		writeResponse, err = (*term).RunWrite(c.session, runOpts)
		return nil, err
	}
	_, err = c.execWithRetry(ctx, name, q)
	return
}

// execWithRetry executes given query function repeatedly until successful or max retry limit is reached
func (c *RethinkDbConnection) execWithRetry(ctx context.Context, name string, q func(ctx context.Context) (*r.Cursor, error)) (cursor *r.Cursor, err error) {
	maxAttempts := c.maxRetries + 1
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	attempts := 0
	log := c.loggerOrDefault().With("operation", name)
retryLoop:
	for attempts < maxAttempts {
		attempts++
		cursor, err = c.exec(ctx, q)
		if err == nil {
			return
		}

		if ctxErr := ctx.Err(); ctxErr != nil {
			err = ctxErr
			break
		}

		action := retryActionFor(err)
		retryable := action != doNotRetry
		log.Warn("Failed to execute query",
			"error", err,
			"attempt", attempts,
			"max_attempts", maxAttempts,
			"retryable", retryable,
		)
		if !retryable || attempts == maxAttempts {
			break
		}

		switch action {
		case waitForWrites:
			log.Warn("Waiting for database to be ready for writes before retrying",
				"attempt", attempts,
				"max_attempts", maxAttempts,
			)
			if waitErr := c.wait(ctx); waitErr != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					err = ctxErr
					break retryLoop
				}
				log.Warn("Failed waiting for database to be ready for writes; retrying query",
					"error", waitErr,
					"attempt", attempts,
					"max_attempts", maxAttempts,
				)
			}
		case reconnect:
			log.Warn("Reconnecting to database before retrying",
				"attempt", attempts,
				"max_attempts", maxAttempts,
			)
			if connectErr := c.Connect(); connectErr != nil {
				log.Warn("Failed to reconnect database; retrying query",
					"error", connectErr,
					"attempt", attempts,
					"max_attempts", maxAttempts,
				)
			}
		}
	}
	return nil, fmt.Errorf("failed to %s after %d of %d attempts: %w", name, attempts, maxAttempts, err)
}

func retryActionFor(err error) retryAction {
	switch {
	case errors.Is(err, r.ErrQueryTimeout):
		return waitForWrites
	case errors.Is(err, r.ErrConnectionClosed):
		return reconnect
	case isPrimaryReplicaUnavailable(err):
		return waitForWrites
	default:
		return doNotRetry
	}
}

func isPrimaryReplicaUnavailable(err error) bool {
	var opFailedErr r.RQLOpFailedError
	return errors.As(err, &opFailedErr) && strings.Contains(strings.ToLower(err.Error()), "primary replica")
}

// exec the given query with a timeout
func (c *RethinkDbConnection) exec(ctx context.Context, q func(ctx context.Context) (*r.Cursor, error)) (*r.Cursor, error) {
	ctx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()
	return q(ctx)
}

// wait for database to be ready for writes
func (c *RethinkDbConnection) wait(ctx context.Context) error {
	waitOpts := r.WaitOpts{
		WaitFor: "ready_for_writes",
		Timeout: c.waitTimeout.Seconds(),
	}
	cursor, err := r.DB(c.connectOpts.Database).Wait(waitOpts).Run(c.session, r.RunOpts{Context: ctx})
	if err != nil {
		return err
	}
	return cursor.Close()
}
