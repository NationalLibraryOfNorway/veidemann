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
	"fmt"
	"slices"
	"time"

	frontierV1 "github.com/NationalLibraryOfNorway/veidemann/api/frontier/v1"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/frontier"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/metrics"
	"github.com/NationalLibraryOfNorway/veidemann/browser-controller/session"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Frontier interface {
	GetNextPage(context.Context) (*frontierV1.PageHarvestSpec, error)
	PageCompleted(context.Context, *frontierV1.PageHarvestSpec, *frontier.RenderResult, error) error
}

type BrowserController struct {
	tokens                      []string
	options                     []session.Option
	frontier                    Frontier
	fetchTimeout, reportTimeout time.Duration
}

func New(tokens []string, frontier Frontier, fetchTimeout, reportTimeout time.Duration, options ...session.Option) BrowserController {
	return BrowserController{tokens: tokens, options: options, frontier: frontier, fetchTimeout: fetchTimeout, reportTimeout: reportTimeout}
}

// Each worker exclusively owns its recorder slot. There is no session registry:
// callback lookup and cross-session completion no longer exist.
func (bc BrowserController) Run(ctx context.Context) error {
	workers, ctx := errgroup.WithContext(ctx)
	metrics.BrowserSessions.Set(float64(len(bc.tokens)))
	for slot := 1; slot <= len(bc.tokens); slot++ {
		workers.Go(func() error { return bc.runWorker(ctx, slot) })
	}
	return workers.Wait()
}

func (bc BrowserController) runWorker(ctx context.Context, slot int) error {
	token := bc.tokens[slot-1]
	options := slices.Clip(bc.options)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		phs, err := bc.frontier.GetNextPage(ctx)
		if status.Code(err) == codes.NotFound {
			timer := time.NewTimer(10 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("get next page: %w", err)
		}
		sess := session.New(slot, append(options, session.WithAdmissionToken(token))...)
		bc.RunFetch(ctx, sess, phs)
		token = sess.NextToken()
		if !sess.Reusable() {
			return fmt.Errorf("slot %d could not be isolated; restart required", slot)
		}
	}
}

func (bc BrowserController) RunFetch(ctx context.Context, sess *session.Session, phs *frontierV1.PageHarvestSpec) {
	metrics.ActiveBrowserSessions.Inc()
	defer metrics.ActiveBrowserSessions.Dec()
	// Let already-started harvests finalize and report when another worker stops.
	ctx = context.WithoutCancel(ctx)
	fetchCtx, cancelFetch := context.WithTimeout(ctx, bc.fetchTimeout)
	defer cancelFetch()
	result, fetchErr := sess.Fetch(fetchCtx, phs)
	reportCtx, cancelReport := context.WithTimeout(ctx, bc.reportTimeout)
	defer cancelReport()
	if err := bc.frontier.PageCompleted(reportCtx, phs, result, fetchErr); err != nil {
		sess.Logger().Error("Harvest report failed", "phase", "frontier_report", "error", err)
	}
}
