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

import java.util.function.Consumer;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import com.google.common.util.concurrent.FutureCallback;
import com.google.common.util.concurrent.Futures;
import com.google.common.util.concurrent.ListenableFuture;
import com.google.common.util.concurrent.MoreExecutors;
import com.google.common.util.concurrent.SettableFuture;

import io.opentracing.Scope;
import io.opentracing.Span;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.CrawlLimitsConfig;
import no.nb.nna.veidemann.api.config.v1.PolitenessConfig.RobotsPolicy;
import no.nb.nna.veidemann.commons.ExtraStatusCodes;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.frontier.worker.DnsServiceClient.Resolution;

/**
 * Preconditions for fetching a URI: crawl execution state, limits, DNS,
 * robots.txt, scope, etc.
 */
public class Preconditions {

    private static final Logger LOG = LoggerFactory.getLogger(Preconditions.class);

    public enum PreconditionOutcome {
        FETCH,
        COMPLETE,
        RETRY,
        MOVE,
    }

    private Preconditions() {
    }

    /**
     * Check whether the crawl execution is still valid and limits are not reached.
     */
    public static boolean crawlExecutionOk(Frontier frontier, StatusWrapper status) throws DbException {
        if (!CrawlExecutionHelpers.isExecutionActive(frontier, status.getId())) {
            return false;
        }
        return !isLimitReached(status);
    }

    /**
     * Check preconditions for a queued URI.
     *
     * Synchronous part:
     * - scope / inclusion
     * - possibly immediate COMPLETE/FETCH
     *
     * Asynchronous part:
     * - DNS resolution (if unresolved)
     * - robots.txt checks
     *
     * If async work is needed, returns a future which will eventually be completed
     * with
     * FETCH / COMPLETE / RETRY / MOVE.
     */
    public static ListenableFuture<PreconditionOutcome> checkPreconditions(
            Frontier frontier,
            ConfigObject crawlConfig,
            StatusWrapper status,
            QueuedUriWrapper qUri) throws DbException {

        qUri.clearError();

        // Scope / inclusion check
        if (!qUri.shouldInclude()) {
            LOG.debug("URI '{}' precluded by scope check. Reason: {}",
                    qUri.getUri(),
                    qUri.getExcludedReasonStatusCode());

            switch (qUri.getExcludedReasonStatusCode()) {
                case -5001: // BLOCKED
                case -4001: // TOO_MANY_HOPS
                    // Do not log these
                    break;
                default:
                    frontier.writeLog(qUri);
            }

            status.incrementDocumentsOutOfScope();
            frontier.getOutOfScopeHandlerClient().submitUri(qUri.getQueuedUri());

            return Futures.immediateFuture(PreconditionOutcome.COMPLETE);
        }

        // If unresolved, we must do DNS -> robots -> allowed/denied
        if (qUri.isUnresolved()) {
            SettableFuture<PreconditionOutcome> future = SettableFuture.create();

            LOG.debug("Resolve IP for URI '{}'", qUri.getUri());
            ListenableFuture<Resolution> dnsFuture = frontier.getDnsServiceClient()
                    .resolve(frontier,
                            qUri.getHost(),
                            qUri.getPort(),
                            qUri.getExecutionId(),
                            crawlConfig.getCrawlConfig().getCollectionRef());

            Futures.addCallback(
                    dnsFuture,
                    new ResolveDnsCallback(frontier, qUri, status, crawlConfig, future),
                    MoreExecutors.directExecutor());

            return future;
        } else {
            // Already resolved; caller can go straight to fetch
            return Futures.immediateFuture(PreconditionOutcome.FETCH);
        }
    }

    static boolean isLimitReached(StatusWrapper status) throws DbException {
        CrawlLimitsConfig limits = status.getCrawlJobConfig().getCrawlJob().getLimits();
        return LimitsCheck.isLimitReached(limits, status);
    }

    private static Scope activateSpanSafely(Frontier frontier, Span parentSpan) {
        if (parentSpan == null) {
            return null;
        }
        try {
            if (frontier.getTracer() == null || frontier.getTracer().scopeManager() == null) {
                return null;
            }
            return frontier.getTracer().scopeManager().activate(parentSpan);
        } catch (Throwable t) {
            LOG.warn("Failed to activate tracing span", t);
            return null;
        }
    }

    private static void safeSet(SettableFuture<PreconditionOutcome> future, PreconditionOutcome state) {
        if (!future.isDone() && !future.isCancelled()) {
            future.set(state);
        }
    }

    private static void safeSetException(SettableFuture<PreconditionOutcome> future, Throwable t) {
        if (!future.isDone() && !future.isCancelled()) {
            future.setException(t);
        } else {
            LOG.warn("Dropping exception because future is already completed/cancelled", t);
        }
    }

    /**
     * Callback for asynchronous DNS resolution.
     */
    static class ResolveDnsCallback implements FutureCallback<Resolution> {
        private final Frontier frontier;
        private final QueuedUriWrapper qUri;
        private final StatusWrapper status;
        private final ConfigObject crawlConfig;
        private final SettableFuture<PreconditionOutcome> future;
        private final Span parentSpan;

        ResolveDnsCallback(Frontier frontier,
                QueuedUriWrapper qUri,
                StatusWrapper status,
                ConfigObject crawlConfig,
                SettableFuture<PreconditionOutcome> future) {
            this.frontier = frontier;
            this.qUri = qUri;
            this.status = status;
            this.crawlConfig = crawlConfig;
            this.future = future;

            Span span = null;
            try {
                if (frontier.getTracer() != null && frontier.getTracer().scopeManager() != null) {
                    span = frontier.getTracer().scopeManager().activeSpan();
                }
            } catch (Throwable t) {
                LOG.warn("Failed to capture parent span", t);
            }
            this.parentSpan = span;
        }

        @Override
        public void onSuccess(Resolution result) {
            if (result.hasError()) {
                handleFailure(result.getError());
                return;
            }

            // Side-effects must always run, even if the future got cancelled.
            try (Scope scope = activateSpanSafely(frontier, parentSpan)) {
                ConfigObject politeness = frontier.getConfig(
                        crawlConfig.getCrawlConfig().getPolitenessRef());
                ConfigObject browserConfig = frontier.getConfig(
                        crawlConfig.getCrawlConfig().getBrowserConfigRef());

                String changedCrawlHostGroup = null;
                if (!qUri.getCrawlHostGroupId().isEmpty()
                        && !qUri.getQueuedUri().getId().isEmpty()) {
                    changedCrawlHostGroup = qUri.getCrawlHostGroupId();
                }

                qUri.setIp(result.getAddress().getAddress().getHostAddress());
                qUri.setResolved(politeness);

                IsAllowedFunc isAllowedFunc = new IsAllowedFunc(
                        changedCrawlHostGroup,
                        frontier,
                        qUri,
                        status,
                        future);

                LOG.debug("Check robots for URI '{}'", qUri.getUri());
                if (politeness.getPolitenessConfig().getRobotsPolicy() == RobotsPolicy.IGNORE_ROBOTS) {
                    isAllowedFunc.accept(true);
                } else {
                    ListenableFuture<Boolean> robotsFuture = frontier.getRobotsServiceClient()
                            .isAllowed(
                                    frontier,
                                    qUri.getQueuedUri(),
                                    browserConfig.getBrowserConfig().getUserAgent(),
                                    politeness,
                                    crawlConfig.getCrawlConfig().getCollectionRef());

                    Futures.addCallback(
                            robotsFuture,
                            new CheckRobotsCallback(isAllowedFunc),
                            MoreExecutors.directExecutor());
                }
            } catch (DbException e) {
                safeSetException(future, e);
            }
        }

        @Override
        public void onFailure(Throwable t) {
            handleFailure(ExtraStatusCodes.FAILED_DNS.toFetchError(t.toString()));
        }

        private void handleFailure(no.nb.nna.veidemann.api.commons.v1.Error error) {
            LOG.info("Failed IP resolution for URI '{}' by extracting host '{}' and port '{}'.",
                    qUri.getUri(),
                    qUri.getHost(),
                    qUri.getPort());

            try (Scope scope = activateSpanSafely(frontier, parentSpan)) {
                qUri.setError(error);

                ExtraStatusCodes statusCode = ExtraStatusCodes.fromFetchError(error);
                boolean initialSeed = qUri.getDiscoveryPath().isEmpty()
                        && qUri.getQueuedUri().getId().isEmpty();
                boolean updateCounters = statusCode.isTemporary() || !initialSeed;

                PreconditionOutcome state = ErrorHandler.fetchFailure(
                        frontier, status, qUri, qUri.getError(), updateCounters);

                if (state == PreconditionOutcome.COMPLETE && !statusCode.isTemporary()) {
                    frontier.writeLog(qUri);
                }

                safeSet(future, state);
            } catch (DbException e) {
                safeSetException(future, e);
            }
        }
    }

    /**
     * Callback for robots.txt checks.
     */
    static class CheckRobotsCallback implements FutureCallback<Boolean> {
        private final IsAllowedFunc isAllowedFunc;
        private final Span parentSpan;

        CheckRobotsCallback(IsAllowedFunc isAllowedFunc) {
            this.isAllowedFunc = isAllowedFunc;

            Span span = null;
            try {
                if (isAllowedFunc.frontier.getTracer() != null
                        && isAllowedFunc.frontier.getTracer().scopeManager() != null) {
                    span = isAllowedFunc.frontier.getTracer().scopeManager().activeSpan();
                }
            } catch (Throwable t) {
                LOG.warn("Failed to capture parent span (robots)", t);
            }
            this.parentSpan = span;
        }

        @Override
        public void onSuccess(Boolean result) {
            // Side-effects must run regardless of cancellation.
            try (Scope scope = activateSpanSafely(isAllowedFunc.frontier, parentSpan)) {
                isAllowedFunc.accept(result);
            }
        }

        @Override
        public void onFailure(Throwable t) {
            try (Scope scope = activateSpanSafely(isAllowedFunc.frontier, parentSpan)) {
                LOG.info("Failed checking robots.txt for URI '{}', will allow harvest. Cause: {}",
                        isAllowedFunc.qUri.getUri(),
                        t.toString());

                // Failure talking to robots service -> fail-open (allow)
                isAllowedFunc.accept(true);
            }
        }
    }

    /**
     * Handles the decision based on robots.txt result and updates Redis/DB
     * accordingly.
     */
    static class IsAllowedFunc implements Consumer<Boolean> {
        private final String changedCrawlHostGroup;
        private final Frontier frontier;
        private final QueuedUriWrapper qUri;
        private final StatusWrapper status;
        private final SettableFuture<PreconditionOutcome> future;

        IsAllowedFunc(String changedCrawlHostGroup,
                Frontier frontier,
                QueuedUriWrapper qUri,
                StatusWrapper status,
                SettableFuture<PreconditionOutcome> future) {
            this.changedCrawlHostGroup = changedCrawlHostGroup;
            this.frontier = frontier;
            this.qUri = qUri;
            this.status = status;
            this.future = future;
        }

        @Override
        public void accept(Boolean isAllowed) {
            PreconditionOutcome state;

            if (Boolean.TRUE.equals(isAllowed)) {
                if (changedCrawlHostGroup != null) {
                    state = PreconditionOutcome.MOVE;
                } else {
                    state = PreconditionOutcome.FETCH;
                }
            } else {
                LOG.info("URI '{}' precluded by robots.txt", qUri.getUri());
                qUri.setError(ExtraStatusCodes.PRECLUDED_BY_ROBOTS.toFetchError());
                status.incrementDocumentsDenied(1L);
                frontier.writeLog(qUri);

                state = PreconditionOutcome.COMPLETE;
            }

            safeSet(future, state);
        }
    }
}
