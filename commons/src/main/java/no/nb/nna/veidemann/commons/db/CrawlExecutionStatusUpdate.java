package no.nb.nna.veidemann.commons.db;

import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;

public record CrawlExecutionStatusUpdate(
        CrawlExecutionStatus previous,
        CrawlExecutionStatus current) {
}