/*
 * Copyright 2026 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 */
package no.nb.nna.veidemann.frontier.worker;

import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.CrawlHostGroup;
import no.nb.nna.veidemann.commons.ExtraStatusCodes;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.frontier.db.QueueLease;
import no.nb.nna.veidemann.frontier.worker.Preconditions.PreconditionOutcome;

/** Owns the ordering between durable URI effects and queue-lease disposition. */
final class UriWorkFinalizer {
    private UriWorkFinalizer() {
    }

    static void finish(
            Frontier frontier,
            StatusWrapper status,
            QueuedUriWrapper uri,
            QueueLease lease,
            PreconditionOutcome outcome,
            CrawlHostGroup sourceGroup,
            long delayMs,
            boolean timeoutRelease) throws DbException {
        try {
            if (outcome == PreconditionOutcome.COMPLETE
                    && uri.hasError()
                    && uri.getDiscoveryPath().isEmpty()
                    && !CrawlExecutionHelpers.isTerminalState(status.getState())) {
                CrawlExecutionStatus.State state = uri.getError().getCode()
                        == ExtraStatusCodes.PRECLUDED_BY_ROBOTS.getCode()
                                ? CrawlExecutionStatus.State.FINISHED
                                : CrawlExecutionStatus.State.FAILED;
                status.setEndState(state).setError(uri.getError());
                frontier.getCrawlQueueManager().removeCrawlExecutionFromTimeoutSchedule(status.getId());
            }

            // The Redis lease remains visible until every accumulated effect has
            // been acknowledged by RethinkDB.
            status.saveStatus();

            switch (outcome) {
                case COMPLETE:
                    frontier.getCrawlQueueManager().completeLease(lease);
                    break;
                case RETRY:
                    uri.save();
                    break;
                case MOVE:
                    uri.save();
                    frontier.getCrawlQueueManager().moveLease(lease, uri.getQueuedUri());
                    break;
                case FETCH:
                    throw new IllegalArgumentException("Cannot finalize a URI with FETCH outcome");
            }

            CrawlExecutionStatus current = frontier.getExecutionsAdapter()
                    .getCrawlExecutionStatus(status.getId());
            if (current == null
                    || CrawlExecutionHelpers.isTerminalState(current.getState())
                    || CrawlExecutionHelpers.isAbortState(current.getDesiredState())
                    || frontier.getCrawlQueueManager().countByCrawlExecution(status.getId()) <= 0) {
                frontier.getCrawlQueueManager().scheduleExecutionFinalization(status.getId());
            } else {
                StatusWrapper.getStatusWrapper(frontier, current)
                        .setState(CrawlExecutionStatus.State.SLEEPING)
                        .saveStatus();
            }
        } finally {
            frontier.getCrawlQueueManager().releaseCrawlHostGroup(sourceGroup, delayMs, timeoutRelease);
        }
    }
}
