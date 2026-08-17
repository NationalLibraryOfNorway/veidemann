package no.nb.nna.veidemann.db;

import com.rethinkdb.RethinkDB;
import com.rethinkdb.gen.ast.Insert;
import com.rethinkdb.gen.ast.ReqlFunction1;
import com.rethinkdb.gen.ast.Update;
import com.rethinkdb.model.MapObject;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.CrawlExecutionStatusChange;
import no.nb.nna.veidemann.api.frontier.v1.JobExecutionStatus;
import no.nb.nna.veidemann.api.frontier.v1.QueuedUri;
import no.nb.nna.veidemann.commons.db.CrawlExecutionStatusUpdate;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbResultSet;
import no.nb.nna.veidemann.commons.db.FrontierAdapter;

import java.util.List;
import java.util.Map;

public class RethinkDbFrontierAdapter implements FrontierAdapter {
    private static final RethinkDB r = RethinkDB.r;

    private final RethinkDbConnection conn;

    public RethinkDbFrontierAdapter(RethinkDbConnection conn) {
        this.conn = conn;
    }

    @Override
    public String getJobExecutionState(String jobExecutionId) throws DbException {
        return conn.executeAtom(
                "db-getJobExecutionState",
                r.table(Tables.JOB_EXECUTIONS.name).get(jobExecutionId).g("state"),
                String.class);
    }

    @Override
    public CrawlExecutionStatus createCrawlExecutionStatus(String jobId, String jobExecutionId, String seedId) throws DbException {
        CrawlExecutionStatus status = CrawlExecutionStatus.newBuilder()
                .setJobId(jobId)
                .setJobExecutionId(jobExecutionId)
                .setSeedId(seedId)
                .setState(CrawlExecutionStatus.State.CREATED)
                .build();

        Map<String, Object> rMap = ProtoUtils.protoToRethink(status);
        rMap.put("lastChangeTime", r.now());
        rMap.put("createdTime", r.now());
        rMap.put("desiredState", r.table(Tables.JOB_EXECUTIONS.name).get(jobExecutionId).g("desiredState").default_("")
                .do_(j -> r.branch(j.eq("ABORTED_MANUAL"), "ABORTED_MANUAL", "UNDEFINED")));

        Insert qry = r.table(Tables.EXECUTIONS.name).insert(rMap);
        return conn.executeInsert("db-createExecutionStatus", qry, CrawlExecutionStatus.class);
    }

    @Override
    public CrawlExecutionStatusUpdate updateCrawlExecutionStatus(CrawlExecutionStatusChange change) throws DbException {
        ReqlFunction1 updateFunc = doc -> {
            MapObject<Object, Object> rMap = r.hashMap("lastChangeTime", r.now());

            switch (change.getState()) {
                case UNDEFINED:
                    break;
                case CREATED:
                    throw new IllegalArgumentException("Not allowed to set state back to CREATED");
                default:
                    rMap.with("state", change.getState().name());
            }

            if (change.getAddBytesCrawled() != 0) {
                rMap.with("bytesCrawled",
                        doc.g("bytesCrawled").add(change.getAddBytesCrawled()).default_(change.getAddBytesCrawled()));
            }
            if (change.getAddDocumentsCrawled() != 0) {
                rMap.with("documentsCrawled",
                        doc.g("documentsCrawled").add(change.getAddDocumentsCrawled()).default_(change.getAddDocumentsCrawled()));
            }
            if (change.getAddDocumentsDenied() != 0) {
                rMap.with("documentsDenied",
                        doc.g("documentsDenied").add(change.getAddDocumentsDenied()).default_(change.getAddDocumentsDenied()));
            }
            if (change.getAddDocumentsFailed() != 0) {
                rMap.with("documentsFailed",
                        doc.g("documentsFailed").add(change.getAddDocumentsFailed()).default_(change.getAddDocumentsFailed()));
            }
            if (change.getAddDocumentsOutOfScope() != 0) {
                rMap.with("documentsOutOfScope",
                        doc.g("documentsOutOfScope").add(change.getAddDocumentsOutOfScope()).default_(change.getAddDocumentsOutOfScope()));
            }
            if (change.getAddDocumentsRetried() != 0) {
                rMap.with("documentsRetried",
                        doc.g("documentsRetried").add(change.getAddDocumentsRetried()).default_(change.getAddDocumentsRetried()));
            }
            if (change.getAddUrisCrawled() != 0) {
                rMap.with("urisCrawled",
                        doc.g("urisCrawled").add(change.getAddUrisCrawled()).default_(change.getAddUrisCrawled()));
            }
            if (change.hasEndTime()) {
                rMap.with("endTime", ProtoUtils.tsToOdt(change.getEndTime()));
            }
            if (change.hasError()) {
                rMap.with("error", ProtoUtils.protoToRethink(change.getError()));
            }

            var updated = doc.merge(rMap)
                    .merge(d -> r.branch(
                            doc.g("state").match("FINISHED|ABORTED_TIMEOUT|ABORTED_SIZE|ABORTED_MANUAL|FAILED|DIED"),
                            r.hashMap("state", doc.g("state")).with("endTime",
                                    r.branch(doc.hasFields("endTime"), doc.g("endTime"), d.g("endTime").default_((Object) null))),

                            d.g("state").match("FINISHED|ABORTED_TIMEOUT|ABORTED_SIZE|ABORTED_MANUAL|FAILED|DIED"),
                            r.hashMap("state", d.g("state")).with("startTime",
                                    r.branch(doc.hasFields("startTime"), doc.g("startTime"), d.g("startTime").default_(r.now()))),

                            d.g("state").match("FETCHING|SLEEPING")
                                    .and(doc.g("state").match("CREATED|FETCHING|SLEEPING")),
                            r.hashMap("state", d.g("state")),

                            r.hashMap("state", doc.g("state"))))
                    .merge(d -> r.branch(doc.hasFields("startTime").not().and(d.g("state").match("FETCHING")),
                            r.hashMap("startTime", r.now()),
                            r.hashMap()));

            // Return an update patch, not the complete merged document. In particular,
            // r.literal() must be a direct value in the patch: removing a field from a
            // document and then passing that document to update() only merges the
            // omission back into the old row.
            MapObject<Object, Object> patch = r.hashMap();
            patch.putAll(rMap);
            if (change.getState() != CrawlExecutionStatus.State.UNDEFINED) {
                patch.with("state", updated.g("state"));
                if (change.getState() == CrawlExecutionStatus.State.FETCHING
                        || change.getState().name().matches(
                                "FINISHED|ABORTED_TIMEOUT|ABORTED_SIZE|ABORTED_MANUAL|FAILED|DIED")) {
                    patch.with("startTime", updated.g("startTime"));
                }
            }
            if (change.hasEndTime()) {
                patch.with("endTime", updated.g("endTime"));
            }

            // desiredState is a command, not a second copy of the actual state.
            // Clear it atomically when the guarded transition reaches that state.
            // Abort states are the complete desiredState command vocabulary. The
            // reconciler only requests the value it read from desiredState, so this
            // guarded terminal transition is exactly the point where the command is
            // consumed. RethinkDB requires literal() to be a direct update-object
            // value; placing it under a branch is rejected as a nested literal.
            if (change.getState() == CrawlExecutionStatus.State.ABORTED_MANUAL
                    || change.getState() == CrawlExecutionStatus.State.ABORTED_TIMEOUT
                    || change.getState() == CrawlExecutionStatus.State.ABORTED_SIZE) {
                patch.with("desiredState", r.literal());
            }
            return patch;
        };

        // literal() structurally removes desiredState while retaining update semantics
        // for the frequent counter-only path.
        Update qry = r.table(Tables.EXECUTIONS.name)
                .get(change.getId())
                .update(updateFunc)
                .optArg("durability", "soft")
                .optArg("return_changes", "always");

        Map<String, Object> response = conn.executeObject("db-updateCrawlExecutionStatus", qry);
        List<Map<String, Map<String, Object>>> changes = castChanges(response.get("changes"));
        if (changes == null || changes.isEmpty()) {
            throw new IllegalStateException("No changes returned when updating CrawlExecutionStatus " + change.getId());
        }

        Map<String, Object> oldVal = changes.get(0).get("old_val");
        Map<String, Object> newVal = changes.get(0).get("new_val");

        return new CrawlExecutionStatusUpdate(
                oldVal == null ? null : ProtoUtils.rethinkToProto(oldVal, CrawlExecutionStatus.class),
                ProtoUtils.rethinkToProto(newVal, CrawlExecutionStatus.class));
    }

    @Override
    public JobExecutionStatus getJobExecutionStatus(String jobExecutionId) throws DbException {
        return conn.executeGet(
                "db-getJobExecutionStatus",
                r.table(Tables.JOB_EXECUTIONS.name).get(jobExecutionId),
                JobExecutionStatus.class);
    }

    @Override
    public JobExecutionStatus getJobExecutionAggregate(String jobExecutionId) throws DbException {
        JobExecutionStatus.Builder aggregate = JobExecutionStatus.newBuilder().setId(jobExecutionId);
        try (DbResultSet<Map<String, Object>> rows = conn.executeSequence(
                "db-getJobExecutionAggregate",
                r.table(Tables.EXECUTIONS.name)
                        .between(
                                r.array(jobExecutionId, r.minval()),
                                r.array(jobExecutionId, r.maxval()))
                        .optArg("index", "jobExecutionId_seedId"))) {
            rows.stream()
                    .map(row -> ProtoUtils.rethinkToProto(row, CrawlExecutionStatus.class))
                    .forEach(execution -> {
                        aggregate.putExecutionsState(
                                execution.getState().name(),
                                aggregate.getExecutionsStateOrDefault(execution.getState().name(), 0) + 1);
                        aggregate.setDocumentsCrawled(
                                aggregate.getDocumentsCrawled() + execution.getDocumentsCrawled());
                        aggregate.setBytesCrawled(
                                aggregate.getBytesCrawled() + execution.getBytesCrawled());
                        aggregate.setUrisCrawled(
                                aggregate.getUrisCrawled() + execution.getUrisCrawled());
                        aggregate.setDocumentsFailed(
                                aggregate.getDocumentsFailed() + execution.getDocumentsFailed());
                        aggregate.setDocumentsOutOfScope(
                                aggregate.getDocumentsOutOfScope() + execution.getDocumentsOutOfScope());
                        aggregate.setDocumentsRetried(
                                aggregate.getDocumentsRetried() + execution.getDocumentsRetried());
                        aggregate.setDocumentsDenied(
                                aggregate.getDocumentsDenied() + execution.getDocumentsDenied());
                    });
        }
        return aggregate.build();
    }

    @Override
    public void saveJobExecutionStatus(JobExecutionStatus status) throws DbException {
        Map<String, Object> update = ProtoUtils.protoToRethink(status);
        conn.executeObject(
                "db-saveJobExecutionStatus",
                r.table(Tables.JOB_EXECUTIONS.name)
                        .get(status.getId())
                        .update(doc -> r.branch(
                                doc.hasFields("endTime"),
                                r.expr(update)
                                        .merge(r.hashMap("state", doc.g("state")))
                                        .merge(r.hashMap("endTime", doc.g("endTime"))),
                                update)));
    }

    @Override
    public QueuedUri saveQueuedUri(QueuedUri queuedUri) throws DbException {
        Map<String, Object> rMap = ProtoUtils.protoToRethink(queuedUri);
        Map<String, Object> response = conn.executeObject(
                "db-saveQueuedUri",
                r.table(Tables.URI_QUEUE.name)
                        .insert(rMap)
                        .optArg("durability", "soft")
                        .optArg("conflict", "replace")
                        .optArg("return_changes", "always"));
        return extractQueuedUri(response);
    }

    @Override
    public QueuedUri updateQueuedUri(QueuedUri queuedUri) throws DbException {
        Map<String, Object> rMap = ProtoUtils.protoToRethink(queuedUri);
        Map<String, Object> response = conn.executeObject(
                "db-saveQueuedUri",
                r.table(Tables.URI_QUEUE.name)
                        .get(queuedUri.getId())
                        // The Redis queue lease gives one worker exclusive ownership of
                        // this URI. Replace is intentional: proto3 default values are
                        // absent from rMap, and update() would therefore fail to clear
                        // values such as unresolved=true after DNS succeeds.
                        .replace(rMap)
                        .optArg("durability", "soft")
                        .optArg("return_changes", "always"));
        return extractQueuedUri(response);
    }

    @Override
    public QueuedUri getQueuedUri(String uriId) throws DbException {
        return conn.executeGet(
                "db-getQueuedUri",
                r.table(Tables.URI_QUEUE.name).get(uriId),
                QueuedUri.class);
    }

    private QueuedUri extractQueuedUri(Map<String, Object> response) {
        List<Map<String, Map<String, Object>>> changes = castChanges(response.get("changes"));
        Map<String, Object> newDoc = changes.get(0).get("new_val");
        return ProtoUtils.rethinkToProto(newDoc, QueuedUri.class);
    }

    @SuppressWarnings("unchecked")
    private List<Map<String, Map<String, Object>>> castChanges(Object value) {
        return (List<Map<String, Map<String, Object>>>) value;
    }
}
