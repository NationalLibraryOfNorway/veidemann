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
package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_BUSY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_PREFIX;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_READY_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_TIMEOUT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_WAIT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.SESSION_TO_CHG_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.UCHG;

import java.util.List;

import com.google.common.collect.ImmutableList;

/** Atomically removes an empty crawl-host group from every Redis schedule. */
public class ChgCleanupIfEmptyScript extends RedisJob<Boolean> {
    private final LuaScript script = new LuaScript("chg_cleanup_if_empty.lua");

    public ChgCleanupIfEmptyScript() {
        super("chgCleanupIfEmptyScript");
    }

    public boolean run(RedisContext ctx, String chgId) {
        return execute(ctx, jedis -> {
            List<String> keys = ImmutableList.of(
                    CHG_BUSY_KEY,
                    CHG_WAIT_KEY,
                    CHG_READY_KEY,
                    CHG_TIMEOUT_KEY,
                    CHG_PREFIX + chgId,
                    SESSION_TO_CHG_KEY,
                    UCHG + chgId);
            Long result = (Long) script.runString(jedis, keys, ImmutableList.of(chgId));
            return result != null && result == 1L;
        });
    }
}
