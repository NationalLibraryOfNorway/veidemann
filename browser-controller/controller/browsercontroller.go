/*
 * Copyright 2020 National Library of Norway.
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

package controller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/metrics"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SessionRegister interface {
	GetNextAvailable(ctx context.Context) (*session.Session, error)
	Release(*session.Session)
}

type Frontier interface {
	GetNextPage(context.Context) (*frontierV1.PageHarvestSpec, error)
	PageCompleted(context.Context, *frontierV1.PageHarvestSpec, *frontier.RenderResult, error) error
}

type BrowserController struct {
	sessions      SessionRegister
	frontier      Frontier
	fetchTimeout  time.Duration
	reportTimeout time.Duration
}

func New(session SessionRegister, frontier Frontier, fetchTimeout, reportTimeout time.Duration) BrowserController {
	return BrowserController{
		sessions:      session,
		frontier:      frontier,
		fetchTimeout:  fetchTimeout,
		reportTimeout: reportTimeout,
	}
}

func (bc BrowserController) Run(ctx context.Context) error {
	var wg sync.WaitGroup

	defer func() {
		slog.Info("Waiting for active browser sessions to complete")
		wg.Wait()
		slog.Info("All active browser sessions completed")
	}()

	backoffTimer := time.NewTimer(0)
	backoffTimer.Stop()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		sess, err := bc.sessions.GetNextAvailable(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			return fmt.Errorf("failed to get next session: %w", err)
		}

		phs, err := bc.frontier.GetNextPage(ctx)
		if err != nil {
			bc.sessions.Release(sess)

			if errors.Is(err, context.Canceled) {
				return err
			}

			if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
				d := 10 * time.Second
				slog.Debug("Next page not found, backing off", "durationMs", d.Milliseconds())

				backoffTimer.Reset(d)

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-backoffTimer.C:
					continue
				}
			}

			return fmt.Errorf("failed to get next page: %w", err)
		}

		wg.Go(func() {
			defer bc.sessions.Release(sess)

			bc.RunFetch(ctx, sess, phs)
		})
	}
}

func (bc BrowserController) RunFetch(ctx context.Context, sess *session.Session, phs *frontierV1.PageHarvestSpec) {
	metrics.ActiveBrowserSessions.Inc()
	defer metrics.ActiveBrowserSessions.Dec()

	ctx = context.WithoutCancel(ctx)

	fetchCtx, cancelFetch := context.WithTimeout(ctx, bc.fetchTimeout)
	defer cancelFetch()

	result, fetchErr := sess.Fetch(fetchCtx, phs)

	reportCtx, cancelReport := context.WithTimeout(ctx, bc.reportTimeout)
	defer cancelReport()

	if err := bc.frontier.PageCompleted(reportCtx, phs, result, fetchErr); err != nil {
		slog.Error("failed to report page completed", "error", err)
	}
}
