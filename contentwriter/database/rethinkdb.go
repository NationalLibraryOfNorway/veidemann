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

	configV1 "github.com/NationalLibraryOfNorway/veidemann/api/config/v1"
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

type Options struct {
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
	waitForReads
	reconnect
)

// NewRethinkDbConnection creates a new RethinkDbConnection object
func NewRethinkDbConnection(opts Options) *RethinkDbConnection {
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

// GetConfigObject fetches a config.ConfigObject referenced by a config.ConfigRef
func (c *RethinkDbConnection) GetConfigObject(ctx context.Context, ref *configV1.ConfigRef) (*configV1.ConfigObject, error) {
	term := r.Table("config").Get(ref.Id)
	res, err := c.execRead(ctx, "get-config-object", &term)
	if err != nil {
		return nil, err
	}
	var result configV1.ConfigObject
	err = res.One(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// execRead executes the given read term with a timeout
func (c *RethinkDbConnection) execRead(ctx context.Context, name string, term *r.Term) (*r.Cursor, error) {
	q := func(ctx context.Context) (*r.Cursor, error) {
		runOpts := r.RunOpts{
			Context: ctx,
		}
		return term.Run(c.session, runOpts)
	}
	return c.execWithRetry(ctx, name, q)
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
		case waitForReads:
			log.Warn("Waiting for database to be ready for reads before retrying",
				"attempt", attempts,
				"max_attempts", maxAttempts,
			)
			if waitErr := c.wait(ctx); waitErr != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					err = ctxErr
					break retryLoop
				}
				log.Warn("Failed waiting for database to be ready for reads; retrying query",
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
		return waitForReads
	case errors.Is(err, r.ErrConnectionClosed):
		return reconnect
	case isPrimaryReplicaUnavailable(err):
		return waitForReads
	default:
		return doNotRetry
	}
}

func isPrimaryReplicaUnavailable(err error) bool {
	var opFailedErr r.RQLOpFailedError
	return errors.As(err, &opFailedErr) && strings.Contains(strings.ToLower(err.Error()), "primary replica")
}

// exec executes the given query with a timeout
func (c *RethinkDbConnection) exec(ctx context.Context, q func(ctx context.Context) (*r.Cursor, error)) (*r.Cursor, error) {
	ctx, cancel := context.WithTimeout(ctx, c.queryTimeout)
	defer cancel()
	return q(ctx)
}

// wait waits for database to be ready for reads
func (c *RethinkDbConnection) wait(ctx context.Context) error {
	waitOpts := r.WaitOpts{
		WaitFor: "ready_for_reads",
		Timeout: c.waitTimeout.Seconds(),
	}
	cursor, err := r.DB(c.connectOpts.Database).Wait(waitOpts).Run(c.session, r.RunOpts{Context: ctx})
	if err != nil {
		return err
	}
	return cursor.Close()
}
