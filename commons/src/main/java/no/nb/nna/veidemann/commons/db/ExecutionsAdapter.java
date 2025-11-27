/*
 * Copyright 2018 National Library of Norway.
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
package no.nb.nna.veidemann.commons.db;

import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus;
import no.nb.nna.veidemann.api.report.v1.CrawlExecutionsListRequest;
import no.nb.nna.veidemann.api.report.v1.JobExecutionsListRequest;

public interface ExecutionsAdapter {
    JobExecutionStatus createJobExecutionStatus(String jobId) throws DbException;

    JobExecutionStatus getJobExecutionStatus(String jobExecutionId) throws DbException;

    ChangeFeed<JobExecutionStatus> listJobExecutionStatus(JobExecutionsListRequest request) throws DbException;

    /**
     * Update the state for a Job Execution to ABORTED_MANUAL.
     *
     * @param jobExecutionId id of the job execution to update
     */
    JobExecutionStatus setJobExecutionStateAborted(String jobExecutionId) throws DbException;

    /**
     * Update the state for all CrawlExecutions of a Job Execution to ABORTED_TIMEOUT.
     *
     * @param jobExecutionId id of the job execution to update
     */
    void setJobExecutionStateAbortedTimeout(String jobExecutionId) throws DbException;

    CrawlExecutionStatus getCrawlExecutionStatus(String crawlExecutionId) throws DbException;

    ChangeFeed<CrawlExecutionStatus> listCrawlExecutionStatus(CrawlExecutionsListRequest request) throws DbException;

    /**
     * Update the state for a Crawl Execution to the submitted abortion state.
     * <p>
     * Only ABORTED_MANUAL, ABORTED_SIZE or ABORTED_TIMEOUT are allowed.
     * The frontier should detect this and abort the crawl.
     *
     * @param crawlExecutionId id of the execution to update
     * @param state            the state to set. Must be one of ABORTED_MANUAL, ABORTED_SIZE, ABORTED_TIMEOUT
     * @throws IllegalArgumentException if an illegal state is submitted
     */
    CrawlExecutionStatus setCrawlExecutionStateAborted(String crawlExecutionId, CrawlExecutionStatus.State state) throws DbException;

    /**
     * Set the desired pause state for Veidemann
     *
     * @param value true if Veidemann should pause
     * @return the old state
     * @throws DbException
     */
    boolean setDesiredPausedState(boolean value) throws DbException;

    /**
     * Get the desired pause state for Veidemann
     *
     * @return true if Veidemann should pause
     * @throws DbException
     */
    boolean getDesiredPausedState() throws DbException;
}
