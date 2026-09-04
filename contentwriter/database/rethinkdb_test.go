/*
 * Copyright 2026 National Library of Norway.
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
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

const primaryReplicaUnavailableMessage = "rethinkdb: Cannot perform read: primary replica for shard not available"

type opFailedTestError string

func (e opFailedTestError) Error() string {
	return string(e)
}

func (e opFailedTestError) As(target any) bool {
	opFailed, ok := target.(*r.RQLOpFailedError)
	if !ok {
		return false
	}
	*opFailed = r.RQLOpFailedError{}
	return true
}

func newRetryTestConnection(maxRetries int) (*RethinkDbConnection, *r.Mock) {
	mock := r.NewMock()
	return &RethinkDbConnection{
		connectOpts: r.ConnectOpts{
			Database: "veidemann",
		},
		session:      mock,
		maxRetries:   maxRetries,
		waitTimeout:  60 * time.Second,
		queryTimeout: time.Second,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}, mock
}

func expectWaitForReads(mock *r.Mock, times int) *r.MockQuery {
	return mock.On(r.DB("veidemann").Wait(r.WaitOpts{
		WaitFor: "ready_for_reads",
		Timeout: float64(60),
	})).Return(map[string]any{"ready": 1}, nil).Times(times)
}

func TestExecWithRetryRetriesPrimaryReplicaFailure(t *testing.T) {
	conn, mock := newRetryTestConnection(5)
	waitQuery := expectWaitForReads(mock, 1)
	wantErr := opFailedTestError(primaryReplicaUnavailableMessage)
	attempts := 0

	_, err := conn.execWithRetry(context.Background(), "test-query", func(context.Context) (*r.Cursor, error) {
		attempts++
		if attempts == 1 {
			return nil, wantErr
		}
		return nil, nil
	})

	if err != nil {
		t.Fatalf("execWithRetry() error = %v, want nil", err)
	}
	if attempts != 2 {
		t.Fatalf("query attempts = %d, want 2", attempts)
	}
	mock.AssertNumberOfExecutions(t, waitQuery, 1)
	mock.AssertExpectations(t)
}

func TestExecWithRetryExhaustsPrimaryReplicaRetries(t *testing.T) {
	conn, mock := newRetryTestConnection(5)
	waitQuery := expectWaitForReads(mock, 5)
	wantErr := opFailedTestError(primaryReplicaUnavailableMessage)
	attempts := 0

	_, err := conn.execWithRetry(context.Background(), "test-query", func(context.Context) (*r.Cursor, error) {
		attempts++
		return nil, wantErr
	})

	if err == nil {
		t.Fatal("execWithRetry() error = nil, want an error")
	}
	if attempts != 6 {
		t.Errorf("query attempts = %d, want 6", attempts)
	}
	if !strings.Contains(err.Error(), "after 6 of 6 attempts") {
		t.Errorf("execWithRetry() error = %q, want attempt count", err)
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("execWithRetry() error = %v, want wrapped error %v", err, wantErr)
	}
	mock.AssertNumberOfExecutions(t, waitQuery, 5)
	mock.AssertExpectations(t)
}

func TestExecWithRetryDoesNotRetryUnrelatedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{name: "unrelated operation failure", err: opFailedTestError("rethinkdb: Table is not available")},
		{name: "ordinary error", err: errors.New("invalid query")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			conn, _ := newRetryTestConnection(5)
			attempts := 0

			_, err := conn.execWithRetry(context.Background(), "test-query", func(context.Context) (*r.Cursor, error) {
				attempts++
				return nil, test.err
			})

			if err == nil {
				t.Fatal("execWithRetry() error = nil, want an error")
			}
			if attempts != 1 {
				t.Errorf("query attempts = %d, want 1", attempts)
			}
			if !strings.Contains(err.Error(), "after 1 of 6 attempts") {
				t.Errorf("execWithRetry() error = %q, want attempt count", err)
			}
		})
	}
}

func TestRetryActionFor(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want retryAction
	}{
		{name: "query timeout", err: fmt.Errorf("wrapped: %w", r.ErrQueryTimeout), want: waitForReads},
		{name: "closed connection", err: fmt.Errorf("wrapped: %w", r.ErrConnectionClosed), want: reconnect},
		{name: "primary replica unavailable", err: opFailedTestError(primaryReplicaUnavailableMessage), want: waitForReads},
		{name: "case insensitive message", err: opFailedTestError("rethinkdb: PRIMARY REPLICA unavailable"), want: waitForReads},
		{name: "untyped primary message", err: errors.New(primaryReplicaUnavailableMessage), want: doNotRetry},
		{name: "unrelated operation failure", err: opFailedTestError("rethinkdb: Database does not exist"), want: doNotRetry},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := retryActionFor(test.err); got != test.want {
				t.Errorf("retryActionFor() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestExecWithRetryStopsWhenContextIsCanceled(t *testing.T) {
	conn, _ := newRetryTestConnection(5)
	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	_, err := conn.execWithRetry(ctx, "test-query", func(context.Context) (*r.Cursor, error) {
		attempts++
		cancel()
		return nil, opFailedTestError(primaryReplicaUnavailableMessage)
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("execWithRetry() error = %v, want context.Canceled", err)
	}
	if attempts != 1 {
		t.Errorf("query attempts = %d, want 1", attempts)
	}
}

func TestWaitHonorsCanceledContext(t *testing.T) {
	conn, mock := newRetryTestConnection(5)
	waitQuery := expectWaitForReads(mock, 1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := conn.wait(ctx)

	if err == nil {
		t.Fatal("wait() error = nil, want cancellation error")
	}
	mock.AssertNumberOfExecutions(t, waitQuery, 1)
	mock.AssertExpectations(t)
}
