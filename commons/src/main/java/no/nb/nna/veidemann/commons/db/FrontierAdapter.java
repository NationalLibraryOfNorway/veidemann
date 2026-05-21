package no.nb.nna.veidemann.commons.db;

import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatusChange;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.QueuedUri;

public interface FrontierAdapter {
    String getJobExecutionState(String jobExecutionId) throws DbException;

    CrawlExecutionStatus createCrawlExecutionStatus(String jobId, String jobExecutionId, String seedId) throws DbException;

    CrawlExecutionStatusUpdate updateCrawlExecutionStatus(CrawlExecutionStatusChange change) throws DbException;

    JobExecutionStatus getJobExecutionStatus(String jobExecutionId) throws DbException;

    void saveJobExecutionStatus(JobExecutionStatus status) throws DbException;

    QueuedUri saveQueuedUri(QueuedUri queuedUri) throws DbException;

    QueuedUri updateQueuedUri(QueuedUri queuedUri) throws DbException;

    QueuedUri getQueuedUri(String uriId) throws DbException;
}