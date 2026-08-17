/*
 * Copyright 2026 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 */
package no.nb.nna.veidemann.frontier.db;

import no.nb.nna.veidemann.api.frontier.v1.QueuedUri;

/**
 * Immutable identity of a queued URI's Redis membership.
 *
 * The queued URI may be mutated while resolving DNS or scheduling a retry. This
 * handle deliberately retains the fields used by the original sorted-set member
 * so completion always removes or moves the entry which was actually leased.
 */
public record QueueLease(
        String uriId,
        String executionId,
        String jobExecutionId,
        String crawlHostGroupId,
        long sequence,
        long fetchTime) {

    public static QueueLease from(QueuedUri uri) {
        if (uri.getId().isEmpty()) {
            throw new IllegalArgumentException("Cannot lease a queued URI without an id");
        }
        return new QueueLease(
                uri.getId(),
                uri.getExecutionId(),
                uri.getJobExecutionId(),
                uri.getCrawlHostGroupId(),
                uri.getSequence(),
                uri.getEarliestFetchTimeStamp().getSeconds());
    }
}
