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

        @SuppressWarnings("unchecked")
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
            MapObject rMap = r.hashMap("lastChangeTime", r.now());

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

            return doc.merge(rMap)
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
        };

        Update qry = r.table(Tables.EXECUTIONS.name)
                .get(change.getId())
                .update(updateFunc)
                .optArg("durability", "soft")
                .optArg("return_changes", "always");

        Map<String, Object> response = conn.executeObject("db-updateCrawlExecutionStatus", qry);
        @SuppressWarnings("unchecked")
        List<Map<String, Map<String, Object>>> changes = (List<Map<String, Map<String, Object>>>) response.get("changes");
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
    public void saveJobExecutionStatus(JobExecutionStatus status) throws DbException {
        conn.executeObject(
                "db-saveJobExecutionStatus",
                r.table(Tables.JOB_EXECUTIONS.name)
                        .get(status.getId())
                        .update(ProtoUtils.protoToRethink(status)));
    }

    @Override
    public QueuedUri saveQueuedUri(QueuedUri queuedUri) throws DbException {
        @SuppressWarnings("unchecked")
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
        @SuppressWarnings("unchecked")
        Map<String, Object> rMap = ProtoUtils.protoToRethink(queuedUri);
        Map<String, Object> response = conn.executeObject(
                "db-saveQueuedUri",
                r.table(Tables.URI_QUEUE.name)
                        .get(queuedUri.getId())
                        .update(rMap)
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
        @SuppressWarnings("unchecked")
        List<Map<String, Map<String, Object>>> changes = (List<Map<String, Map<String, Object>>>) response.get("changes");
        Map<String, Object> newDoc = changes.get(0).get("new_val");
        return ProtoUtils.rethinkToProto(newDoc, QueuedUri.class);
    }
}