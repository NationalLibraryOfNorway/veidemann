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

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.protobuf.Timestamp;

import no.nb.nna.veidemann.api.commons.v1.Error;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.ConfigRef;
import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatusChange;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus;
import no.nb.nna.veidemann.commons.db.CrawlExecutionStatusUpdate;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbQueryException;
import no.nb.nna.veidemann.db.ProtoUtils;

/**
 * Wrapper around CrawlExecutionStatus that accumulates changes and flushes them
 * atomically to RethinkDB.
 *
 * The general pattern is:
 * statusWrapper
 * .setState(...)
 * .incrementDocumentsCrawled()
 * .saveStatus();
 *
 * NOTE:
 * - State changes set the "dirty" flag; calling getCrawlExecutionStatus() while
 * dirty will throw.
 * - Counter changes are kept only in the "change" object and not visible via
 * getters until saveStatus() runs.
 */
public class StatusWrapper {

    private static final Logger LOG = LoggerFactory.getLogger(StatusWrapper.class);

    private CrawlExecutionStatus.Builder status;
    private CrawlExecutionStatusChange.Builder change;
    private boolean dirty;

    private final Frontier frontier;
    private ConfigObject jobConfig;
    private ConfigObject crawlConfig;

    private StatusWrapper(Frontier frontier, CrawlExecutionStatus status) {
        this.frontier = frontier;
        this.status = status.toBuilder();
    }

    private StatusWrapper(Frontier frontier, CrawlExecutionStatus.Builder status) {
        this.frontier = frontier;
        this.status = status;
    }

    public static StatusWrapper getStatusWrapper(Frontier frontier, String executionId) throws DbException {
        return new StatusWrapper(
                frontier,
            frontier.getExecutionsAdapter()
                        .getCrawlExecutionStatus(executionId));
    }

    public static StatusWrapper getStatusWrapper(Frontier frontier, CrawlExecutionStatus status) {
        return new StatusWrapper(frontier, status);
    }

    public static StatusWrapper getStatusWrapper(Frontier frontier, CrawlExecutionStatus.Builder status) {
        return new StatusWrapper(frontier, status);
    }

    /**
     * Persist accumulated changes to RethinkDB and update the internal status
     * snapshot.
     *
     * Also updates JobExecutionStatus aggregates when this was the last running
     * execution.
     */
    public synchronized StatusWrapper saveStatus() throws DbException {
        if (change == null) {
            // Nothing to flush
            dirty = false;
            return this;
        }

        change.setId(status.getId());

        CrawlExecutionStatusUpdate update = frontier.getFrontierAdapter().updateCrawlExecutionStatus(change.build());
        CrawlExecutionStatus oldDoc = update.previous();
        CrawlExecutionStatus newDoc = update.current();
        boolean wasNotEnded = oldDoc == null || !oldDoc.hasEndTime();

        CrawlExecutionStatus.State newState = newDoc.getState();

        boolean terminal = newState == CrawlExecutionStatus.State.FINISHED ||
                newState == CrawlExecutionStatus.State.ABORTED_MANUAL ||
                newState == CrawlExecutionStatus.State.ABORTED_TIMEOUT ||
                newState == CrawlExecutionStatus.State.ABORTED_SIZE ||
                newState == CrawlExecutionStatus.State.FAILED ||
                newState == CrawlExecutionStatus.State.DIED;

        // If this execution is in a terminal state but 'change' did not set a terminal
        // transition, we must force a proper cleanup transition to Redis.
        if (terminal && (change == null || change.getState() == CrawlExecutionStatus.State.UNDEFINED)) {
            LOG.debug("Patching missing terminal transition for execution {} -> {}", getId(), newState);
            getChange().setState(newState);
        }

        // Propagate status change to JobExecutionStatus
        Boolean hasRunningExecutions = frontier.getCrawlQueueManager()
                .updateJobExecutionStatus(
                        newDoc.getJobExecutionId(),
                        oldDoc == null ? status.getState() : oldDoc.getState(),
                        newDoc.getState(),
                        change);

        boolean noRunningExecutions = (hasRunningExecutions == null || !hasRunningExecutions);
        if (noRunningExecutions && wasNotEnded && newDoc.hasEndTime()) {
            updateJobExecution(newDoc.getJobExecutionId());
        }

        status = newDoc.toBuilder();
        change = null;
        dirty = false;

        return this;
    }

    private boolean updateJobExecution(String jobExecutionId) throws DbException {
        return updateJobExecution(jobExecutionId, false);
    }

    private boolean updateJobExecution(String jobExecutionId, boolean reconstruct) throws DbException {
        JobExecutionStatus tjes = reconstruct
                ? null
                : frontier.getCrawlQueueManager().getTempJobExecutionStatus(jobExecutionId);
        if (tjes == null) {
            LOG.info("Reconstructing JobExecution '{}' aggregate from RethinkDB", jobExecutionId);
            tjes = frontier.getFrontierAdapter().getJobExecutionAggregate(jobExecutionId);
            int running = tjes.getExecutionsStateOrDefault(CrawlExecutionStatus.State.UNDEFINED.name(), 0)
                    + tjes.getExecutionsStateOrDefault(CrawlExecutionStatus.State.CREATED.name(), 0)
                    + tjes.getExecutionsStateOrDefault(CrawlExecutionStatus.State.FETCHING.name(), 0)
                    + tjes.getExecutionsStateOrDefault(CrawlExecutionStatus.State.SLEEPING.name(), 0);
            int total = tjes.getExecutionsStateMap().values().stream().mapToInt(value -> value).sum();
            if (total == 0 || running > 0) {
                return false;
            }
        }

        LOG.debug("JobExecution '{}' finished, saving stats", jobExecutionId);

        JobExecutionStatus jes = frontier.getFrontierAdapter().getJobExecutionStatus(jobExecutionId);

        if (jes == null) {
            throw new IllegalStateException("Can't find JobExecution: " + jobExecutionId);
        }

        // Decide final JobExecution state
        JobExecutionStatus.State state;
        switch (jes.getState()) {
            case FAILED:
            case ABORTED_MANUAL:
                state = jes.getState();
                break;
            default:
                if (jes.getDesiredState() != JobExecutionStatus.State.UNDEFINED) {
                    state = jes.getDesiredState();
                } else {
                    state = JobExecutionStatus.State.FINISHED;
                }
                break;
        }

        // Update aggregated statistics
        JobExecutionStatus.Builder jesBuilder = jes.toBuilder()
                .setState(state)
                .setEndTime(ProtoUtils.getNowTs());
        jesBuilder.mergeFrom(tjes);

        frontier.getFrontierAdapter().saveJobExecutionStatus(jesBuilder.build());

        frontier.getCrawlQueueManager().removeRedisJobExecution(jobExecutionId);
        return true;
    }

    /** Repair a persistent JobExecution even when no crawl transition is in flight. */
    public static boolean repairJobExecution(Frontier frontier, String jobExecutionId) throws DbException {
        StatusWrapper wrapper = new StatusWrapper(frontier, CrawlExecutionStatus.getDefaultInstance());
        // Explicit repair must not trust a Redis aggregate which may itself be the
        // artifact left behind by the interrupted terminal transition.
        return wrapper.updateJobExecution(jobExecutionId, true);
    }

    public String getId() {
        return status.getId();
    }

    public ConfigObject getCrawlJobConfig() throws DbQueryException {
        if (jobConfig == null) {
            jobConfig = frontier.getConfig(
                    ConfigRef.newBuilder()
                            .setKind(Kind.crawlJob)
                            .setId(status.getJobId())
                            .build());
        }
        return jobConfig;
    }

    public ConfigObject getCrawlConfig() throws DbQueryException {
        if (crawlConfig == null) {
            crawlConfig = frontier.getConfig(
                    getCrawlJobConfig().getCrawlJob().getCrawlConfigRef());
        }
        return crawlConfig;
    }

    public String getJobExecutionId() {
        return status.getJobExecutionId();
    }

    public Timestamp getStartTime() {
        return getCrawlExecutionStatus().getStartTime();
    }

    public Timestamp getCreatedTime() {
        return getCrawlExecutionStatus().getCreatedTime();
    }

    public Timestamp getEndTime() {
        return getCrawlExecutionStatus().getEndTime();
    }

    public boolean isEnded() {
        return getCrawlExecutionStatus().hasEndTime();
    }

    public CrawlExecutionStatus.State getState() {
        return getCrawlExecutionStatus().getState();
    }

    public CrawlExecutionStatus.State getDesiredState() {
        return status.getDesiredState();
    }

    public StatusWrapper setState(CrawlExecutionStatus.State state) {
        dirty = true;
        getChange().setState(state);
        return this;
    }

    public StatusWrapper setEndState(CrawlExecutionStatus.State state) {
        LOG.debug("Reached end of crawl '{}' with state: {}", getId(), state);
        dirty = true;
        getChange()
                .setState(state)
                .setEndTime(ProtoUtils.getNowTs());
        return this;
    }

    public long getDocumentsCrawled() {
        return getCrawlExecutionStatus().getDocumentsCrawled();
    }

    public synchronized StatusWrapper incrementDocumentsCrawled() {
        getChange().setAddDocumentsCrawled(
                getChange().getAddDocumentsCrawled() + 1);
        return this;
    }

    public long getBytesCrawled() {
        return getCrawlExecutionStatus().getBytesCrawled();
    }

    public synchronized StatusWrapper incrementBytesCrawled(long val) {
        getChange().setAddBytesCrawled(
                getChange().getAddBytesCrawled() + val);
        return this;
    }

    public long getUrisCrawled() {
        return getCrawlExecutionStatus().getUrisCrawled();
    }

    public synchronized StatusWrapper incrementUrisCrawled(long val) {
        // BUGFIX: previously used getAddDocumentsCrawled() – wrong field.
        getChange().setAddUrisCrawled(
                getChange().getAddUrisCrawled() + val);
        return this;
    }

    public long getDocumentsFailed() {
        return getCrawlExecutionStatus().getDocumentsFailed();
    }

    public synchronized StatusWrapper incrementDocumentsFailed() {
        getChange().setAddDocumentsFailed(
                getChange().getAddDocumentsFailed() + 1);
        return this;
    }

    public long getDocumentsOutOfScope() {
        return getCrawlExecutionStatus().getDocumentsOutOfScope();
    }

    public synchronized StatusWrapper incrementDocumentsOutOfScope() {
        getChange().setAddDocumentsOutOfScope(
                getChange().getAddDocumentsOutOfScope() + 1);
        return this;
    }

    public long getDocumentsRetried() {
        return getCrawlExecutionStatus().getDocumentsRetried();
    }

    public synchronized StatusWrapper incrementDocumentsRetried() {
        getChange().setAddDocumentsRetried(
                getChange().getAddDocumentsRetried() + 1);
        return this;
    }

    public long getDocumentsDenied() {
        return getCrawlExecutionStatus().getDocumentsDenied();
    }

    public synchronized StatusWrapper incrementDocumentsDenied(long val) {
        getChange().setAddDocumentsDenied(
                getChange().getAddDocumentsDenied() + val);
        return this;
    }

    /**
     * Get an immutable snapshot of the current CrawlExecutionStatus.
     *
     * Throws if there is an unsaved state change pending (dirty == true), to force
     * callers
     * to call saveStatus() first in that case.
     *
     * Counter-only changes (documents/bytes/etc) are NOT reflected here until
     * saveStatus() has run.
     */
    public CrawlExecutionStatus getCrawlExecutionStatus() {
        if (dirty) {
            throw new IllegalStateException("CrawlExecutionStatus is dirty and must be saved before read: " + change);
        }
        return status.build();
    }

    public StatusWrapper setError(Error error) {
        getChange().setError(error);
        return this;
    }

    public StatusWrapper setError(int code, String message) {
        getChange().setError(
                Error.newBuilder().setCode(code).setMsg(message).build());
        return this;
    }

    private CrawlExecutionStatusChange.Builder getChange() {
        if (change == null) {
            change = CrawlExecutionStatusChange.newBuilder();
        }
        return change;
    }
}
