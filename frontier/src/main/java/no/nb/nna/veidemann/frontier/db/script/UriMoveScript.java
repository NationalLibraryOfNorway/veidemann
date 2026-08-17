/*
 * Copyright 2026 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 */
package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_PREFIX;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.CHG_WAIT_KEY;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.UCHG;
import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.UEID;

import java.util.List;
import java.util.Locale;

import com.google.common.collect.ImmutableList;

import no.nb.nna.veidemann.api.frontier.v1.QueuedUri;
import no.nb.nna.veidemann.frontier.db.QueueLease;

/** Atomically transfers a URI membership between crawl-host groups. */
public class UriMoveScript extends RedisJob<Long> {
    private final LuaScript script = new LuaScript("uri_move.lua");

    public UriMoveScript() {
        super("uriMoveScript");
    }

    public long run(RedisContext ctx, QueueLease source, QueuedUri target) {
        String sourceMember = String.format("%4d:%d:%s",
                source.sequence(), source.fetchTime(), source.uriId());
        String targetMember = String.format("%4d:%d:%s",
                target.getSequence(), target.getEarliestFetchTimeStamp().getSeconds(), target.getId());

        double priorityWeight = target.getPriorityWeight();
        if (target.getDiscoveryPath().isEmpty()) {
            priorityWeight += 100d;
        }

        List<String> keys = ImmutableList.of(
                UEID + source.crawlHostGroupId() + ":" + source.executionId(),
                UCHG + source.crawlHostGroupId(),
                CHG_PREFIX + source.crawlHostGroupId(),
                UEID + target.getCrawlHostGroupId() + ":" + target.getExecutionId(),
                UCHG + target.getCrawlHostGroupId(),
                CHG_PREFIX + target.getCrawlHostGroupId(),
                CHG_WAIT_KEY);
        List<String> args = ImmutableList.of(
                sourceMember,
                targetMember,
                source.executionId(),
                source.crawlHostGroupId(),
                target.getCrawlHostGroupId(),
                String.format(Locale.ENGLISH, "%1.2f", priorityWeight),
                Long.toString(target.getEarliestFetchTimeStamp().getSeconds() * 1000L));

        return execute(ctx, jedis -> (Long) script.runString(jedis, keys, args));
    }
}
