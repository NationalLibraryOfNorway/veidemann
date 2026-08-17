/*
 * Copyright 2026 National Library of Norway.
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
package no.nb.nna.veidemann.db.initializer;

import static com.rethinkdb.RethinkDB.r;

import java.util.Map;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.commons.db.DbConnectionException;
import no.nb.nna.veidemann.commons.db.DbQueryException;
import no.nb.nna.veidemann.db.RethinkDbConnection;
import no.nb.nna.veidemann.db.Tables;

/**
 * Repairs the crawl execution invariant that desiredState is unset once the
 * desired state has been reached.
 */
final class CrawlExecutionDesiredStateMigration implements Runnable {
    private static final Logger LOG = LoggerFactory.getLogger(CrawlExecutionDesiredStateMigration.class);
    private static final String MIGRATION_ID = "migration_clear_stale_crawl_execution_desired_state_v1";

    private final RethinkDbConnection conn;

    CrawlExecutionDesiredStateMigration(RethinkDbConnection conn) {
        this.conn = conn;
    }

    @Override
    public void run() {
        try {
            Object completed = conn.exec(r.table(Tables.SYSTEM.name).get(MIGRATION_ID));
            if (completed != null) {
                return;
            }

            // Use the existing state index rather than building a desiredState index over
            // the entire executions table. Only terminal abort rows can violate this
            // historical invariant.
            conn.exec(r.table(Tables.EXECUTIONS.name).indexWait("state"));

            long replaced = 0;
            for (CrawlExecutionStatus.State state : new CrawlExecutionStatus.State[] {
                    CrawlExecutionStatus.State.ABORTED_MANUAL,
                    CrawlExecutionStatus.State.ABORTED_TIMEOUT,
                    CrawlExecutionStatus.State.ABORTED_SIZE }) {
                Map<String, Object> result = conn.exec(
                        r.table(Tables.EXECUTIONS.name)
                                .getAll(state.name()).optArg("index", "state")
                                .filter(row -> row.g("desiredState").default_("UNDEFINED").eq(state.name()))
                                // update() would merge this document back into the old one,
                                // retaining the omitted field. replace() makes without()
                                // structurally remove it.
                                .replace(row -> row.without("desiredState"))
                                .optArg("durability", "hard"));
                replaced += ((Number) result.getOrDefault("replaced", 0L)).longValue();
            }

            long remaining = 0;
            for (CrawlExecutionStatus.State state : new CrawlExecutionStatus.State[] {
                    CrawlExecutionStatus.State.ABORTED_MANUAL,
                    CrawlExecutionStatus.State.ABORTED_TIMEOUT,
                    CrawlExecutionStatus.State.ABORTED_SIZE }) {
                Number count = conn.exec(
                        r.table(Tables.EXECUTIONS.name)
                                .getAll(state.name()).optArg("index", "state")
                                .filter(row -> row.g("desiredState").default_("UNDEFINED").eq(state.name()))
                                .count());
                remaining += count.longValue();
            }
            if (remaining != 0) {
                throw new IllegalStateException(
                        "Refusing to mark desiredState migration complete; " + remaining + " stale rows remain");
            }

            conn.exec(r.table(Tables.SYSTEM.name).insert(
                    r.hashMap("id", MIGRATION_ID).with("completedTime", r.now()))
                    .optArg("conflict", "replace")
                    .optArg("durability", "hard"));

            LOG.info("Crawl execution desiredState migration completed; repaired {} rows", replaced);
        } catch (DbQueryException | DbConnectionException | IllegalStateException e) {
            throw new RuntimeException("Could not repair stale crawl execution desiredState values", e);
        }
    }
}
