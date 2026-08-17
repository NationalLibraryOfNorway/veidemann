/*
 * Copyright 2017 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package no.nb.nna.veidemann.frontier.worker;

import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus.State;
import no.nb.nna.veidemann.commons.db.DbException;

/**
 * Helpers for handling crawl execution lifecycle and finalization.
 */
public class CrawlExecutionHelpers {

    public static boolean isAborted(Frontier frontier,
            StatusWrapper status) throws DbException {
        CrawlExecutionStatus current = frontier.getExecutionsAdapter()
                .getCrawlExecutionStatus(status.getId());
        if (current == null) {
            return false;
        }
        if (isAbortState(current.getDesiredState())) {
            status.saveStatus();
            frontier.getCrawlQueueManager().scheduleExecutionFinalization(status.getId());
            return true;
        }
        return isAbortState(current.getState());
    }

    /**
     * Return whether new work may be queued or issued for an execution.
     *
     * The status is read afresh because desiredState may have changed while DNS,
     * robots, scope, or prefetch work was in progress.
     */
    public static boolean isExecutionActive(Frontier frontier, String executionId) throws DbException {
        CrawlExecutionStatus current = frontier.getExecutionsAdapter()
                .getCrawlExecutionStatus(executionId);
        if (current == null || isTerminalState(current.getState())) {
            return false;
        }
        if (isAbortState(current.getDesiredState())) {
            frontier.getCrawlQueueManager().scheduleExecutionFinalization(executionId);
            return false;
        }
        return true;
    }

    public static boolean isAbortState(State state) {
        return state == State.ABORTED_MANUAL
                || state == State.ABORTED_TIMEOUT
                || state == State.ABORTED_SIZE;
    }

    public static boolean isTerminalState(State state) {
        return state == State.FINISHED
                || isAbortState(state)
                || state == State.FAILED
                || state == State.DIED;
    }
}
