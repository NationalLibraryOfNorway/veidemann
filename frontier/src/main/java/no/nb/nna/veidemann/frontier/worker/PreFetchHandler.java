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

import java.net.URISyntaxException;
import java.util.Collection;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.MDC;

import no.nb.nna.veidemann.api.config.v1.Annotation;
import no.nb.nna.veidemann.api.config.v1.ConfigObject;
import no.nb.nna.veidemann.api.config.v1.ConfigRef;
import no.nb.nna.veidemann.api.config.v1.Kind;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus.State;
import no.nb.nna.veidemann.api.frontier.v1.PageHarvestSpec;
import no.nb.nna.veidemann.api.frontier.v1.QueuedUri;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.db.ProtoUtils;
import no.nb.nna.veidemann.frontier.worker.Preconditions.PreconditionState;

public class PreFetchHandler {

    private static final Logger LOG = LoggerFactory.getLogger(PreFetchHandler.class);

    private final StatusWrapper status;
    private final Frontier frontier;
    private final QueuedUriWrapper qUri;

    public PreFetchHandler(QueuedUri uri, Frontier frontier) throws DbException {
        this.frontier = frontier;
        this.status = StatusWrapper.getStatusWrapper(frontier, uri.getExecutionId());

        ConfigObject collectionConfig = frontier.getConfig(
                status.getCrawlConfig().getCrawlConfig().getCollectionRef());
        ConfigObject seed = frontier.getConfig(
                ConfigRef.newBuilder()
                        .setKind(Kind.seed)
                        .setId(status.getCrawlExecutionStatus().getSeedId())
                        .build());

        Collection<Annotation> scriptParameters = frontier.getScriptParameterResolver()
                .GetScriptParameters(seed, status.getCrawlJobConfig());

        try {
            this.qUri = QueuedUriWrapper
                    .getQueuedUriWrapperWithScopeCheck(
                            frontier,
                            uri,
                            collectionConfig.getMeta().getName(),
                            scriptParameters,
                            status.getCrawlJobConfig().getCrawlJob().getScopeScriptRef())
                    .clearError();
        } catch (URISyntaxException ex) {
            throw new RuntimeException(ex);
        }
    }

    public QueuedUriWrapper getQueuedUri() {
        return qUri;
    }

    /**
     * Check if crawl is aborted, run preconditions and do prefetch-side effects.
     *
     * @return true if we should do the fetch (harvester should be given a spec)
     */
    public boolean preFetch() throws DbException {
        MDC.put("eid", qUri.getExecutionId());
        MDC.put("uri", qUri.getUri());
        try {
            if (!qUri.getCrawlHostGroup().getSessionToken().isEmpty()) {
                throw new IllegalStateException("Fetching in progress from another harvester");
            }

            if (!Preconditions.crawlExecutionOk(frontier, status)) {
                LOG.debug("DENIED");
                frontier.getCrawlQueueManager().removeQUri(qUri);
                CrawlExecutionHelpers.postFetchFinally(frontier, status, qUri, 0);
                return false;
            }

            String curCrawlHostGroupId = qUri.getCrawlHostGroupId();
            PreconditionState check = Preconditions
                    .checkPreconditions(frontier, status.getCrawlConfig(), status, qUri)
                    .get();

            switch (check) {
                case DENIED:
                    LOG.debug("DENIED");
                    status.saveStatus();
                    CrawlExecutionHelpers.postFetchFinally(frontier, status, qUri, 0);
                    return false;
                case RETRY:
                    LOG.debug("RETRY");
                    status.saveStatus();
                    frontier.getCrawlQueueManager().releaseCrawlHostGroup(curCrawlHostGroupId, 0);
                    return false;
                case OK:
                    LOG.debug("OK");
                    qUri.setPriorityWeight(
                            status.getCrawlConfig().getCrawlConfig().getPriorityWeight());
                    status.saveStatus();
                    return true;
                default:
                    LOG.error("Unknown precondition state: {}", check);
                    return false;
            }
        } catch (DbException e) {
            LOG.error("Failed communicating with DB in preFetch: {}", e.toString(), e);
            throw e;
        } catch (Throwable t) {
            // Errors should be handled elsewhere. Exception here indicates a bug.
            LOG.error("Possible bug in preFetch: {}", t.toString(), t);
            throw new RuntimeException("Bug in preFetch", t);
        } finally {
            MDC.remove("eid");
            MDC.remove("uri");
        }
    }

    public PageHarvestSpec getHarvestSpec() throws DbException {
        qUri.setFetchStartTimeStamp(ProtoUtils.getNowTs());
        qUri.generateSessionToken();
        frontier.getCrawlQueueManager().updateCrawlHostGroup(qUri.getCrawlHostGroup());
        status.setState(State.FETCHING).saveStatus();

        LOG.debug("Fetching {}", qUri.getUri());
        return PageHarvestSpec.newBuilder()
                .setSessionToken(qUri.getCrawlHostGroup().getSessionToken())
                .setQueuedUri(qUri.getQueuedUri())
                .setCrawlConfig(status.getCrawlConfig())
                .build();
    }
}
