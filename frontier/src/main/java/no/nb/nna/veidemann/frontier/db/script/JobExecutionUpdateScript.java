package no.nb.nna.veidemann.frontier.db.script;

import static no.nb.nna.veidemann.frontier.db.CrawlQueueManager.JOB_EXECUTION_PREFIX;

import java.util.List;

import com.google.common.collect.ImmutableList;

import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus.State;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatusChangeOrBuilder;

public class JobExecutionUpdateScript extends RedisJob<Boolean> {
    private final LuaScript jobExecutionUpdate;

    public JobExecutionUpdateScript() {
        super("jobExecutionUpdate");
        this.jobExecutionUpdate = new LuaScript("jobexecution_update.lua");
    }

    public Boolean run(JedisContext ctx,
            String jobExecutionId,
            State oldState,
            State newState,
            CrawlExecutionStatusChangeOrBuilder change) {
        return execute(ctx, jedis -> {
            String key = JOB_EXECUTION_PREFIX + jobExecutionId;

            String oState = State.UNDEFINED.name();
            String nState = State.UNDEFINED.name();

            if (oldState != newState) {
                if (oldState != null && oldState != State.UNDEFINED) {
                    oState = oldState.name();
                }
                nState = newState.name();
            }

            String documentsCrawled = Long.toString(change.getAddDocumentsCrawled());
            String documentsDenied = Long.toString(change.getAddDocumentsDenied());
            String documentsFailed = Long.toString(change.getAddDocumentsFailed());
            String documentsOutOfScope = Long.toString(change.getAddDocumentsOutOfScope());
            String documentsRetried = Long.toString(change.getAddDocumentsRetried());
            String urisCrawled = Long.toString(change.getAddUrisCrawled());
            String bytesCrawled = Long.toString(change.getAddBytesCrawled());

            List<String> keys = ImmutableList.of(key);
            List<String> args = ImmutableList.of(
                    oState,
                    nState,
                    documentsCrawled,
                    documentsDenied,
                    documentsFailed,
                    documentsOutOfScope,
                    documentsRetried,
                    urisCrawled,
                    bytesCrawled);

            Long runningExecutions = (Long) jobExecutionUpdate.runString(jedis, keys, args);

            if (runningExecutions == null) {
                runningExecutions = 0L;
            }

            return runningExecutions > 0;

        });
    }
}
