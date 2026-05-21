/*
 * Copyright 2018 National Library of Norway.
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
package no.nb.nna.veidemann.db;

import com.google.protobuf.Message;
import com.rethinkdb.RethinkDB;
import com.rethinkdb.ast.ReqlAst;
import com.rethinkdb.gen.ast.Get;
import com.rethinkdb.gen.ast.Insert;
import com.rethinkdb.gen.ast.ReqlExpr;
import com.rethinkdb.gen.ast.Update;
import com.rethinkdb.gen.exc.ReqlDriverError;
import com.rethinkdb.gen.exc.ReqlError;
import com.rethinkdb.gen.exc.ReqlOpFailedError;
import com.rethinkdb.model.OptArgs;
import com.rethinkdb.net.Connection;
import com.rethinkdb.net.Result;
import no.nb.nna.veidemann.commons.db.ConfigAdapter;
import no.nb.nna.veidemann.commons.db.DbConnectionException;
import no.nb.nna.veidemann.commons.db.DbException;
import no.nb.nna.veidemann.commons.db.DbInitializer;
import no.nb.nna.veidemann.commons.db.DbQueryAdapter;
import no.nb.nna.veidemann.commons.db.DbQueryException;
import no.nb.nna.veidemann.commons.db.DbResultSet;
import no.nb.nna.veidemann.commons.db.DbServiceSPI;
import no.nb.nna.veidemann.commons.db.ExecutionsAdapter;
import no.nb.nna.veidemann.commons.db.FrontierAdapter;
import no.nb.nna.veidemann.commons.settings.CommonSettings;
import no.nb.nna.veidemann.db.initializer.RethinkDbInitializer;
import no.nb.nna.veidemann.db.opentracing.ConnectionTracingInterceptor;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.lang.reflect.Array;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.Iterator;
import java.util.List;
import java.util.Map;

public class RethinkDbConnection implements DbServiceSPI {
    private static final Logger LOG = LoggerFactory.getLogger(RethinkDbConnection.class);

    private static final long MAX_WAIT_FOR_DB_MILLIS = 1000 * 60 * 30; // Half an hour

    static final String RETHINK_ARRAY_LIMIT_KEY = "RETHINK_ARRAY_LIMIT";

    static final RethinkDB r = RethinkDB.r;

    private Connection conn;

    private RethinkDbConfigAdapter configAdapter;

    private RethinkDbExecutionsAdapter executionsAdapter;

    private RethinkDbQueryAdapter dbQueryAdapter;

    private RethinkDbFrontierAdapter frontierAdapter;

    private RethinkDbInitializer dbInitializer;

    public <T> T exec(ReqlAst qry) throws DbConnectionException, DbQueryException {
        return exec("db-query", qry);
    }

    @SuppressWarnings("unchecked")
    public <T> T exec(String operationName, ReqlAst qry) throws DbConnectionException, DbQueryException {
        return (T) toUntypedResult(runQuery(operationName, qry));
    }

    public <T> T executeAtom(String operationName, ReqlAst qry, Class<T> type) throws DbConnectionException, DbQueryException {
        try (Result<Object> result = runQuery(operationName, qry)) {
            return type.cast(readRequiredAtom(operationName, result));
        }
    }

    @SuppressWarnings("unchecked")
    public Map<String, Object> executeObject(String operationName, ReqlAst qry) throws DbConnectionException, DbQueryException {
        return (Map<String, Object>) executeAtom(operationName, qry, Object.class);
    }

    public <T> DbResultSet<T> executeSequence(String operationName, ReqlAst qry) throws DbConnectionException, DbQueryException {
        return toSequenceResult(operationName, runQuery(operationName, qry));
    }

    public DbResultSet<Object> executeQuery(String operationName, ReqlAst qry) throws DbConnectionException, DbQueryException {
        return executeSequence(operationName, qry);
    }

    private Result<Object> runQuery(String operationName, ReqlAst qry) throws DbConnectionException, DbQueryException {
        synchronized (this) {
            if (!conn.isOpen()) {
                try {
                    conn.connect();
                } catch (RuntimeException ex) {
                    LOG.debug(ex.toString(), ex);
                    throw new DbConnectionException("Failed connecting to RethinkDB", ex);
                }
            }
        }

        int retries = 0;
        long startTime = System.currentTimeMillis();

        OptArgs globalOpts = OptArgs.of(ConnectionTracingInterceptor.OPERATION_NAME_KEY, operationName);
        int arrayLimit = getArrayLimit();
        if (arrayLimit > 0) {
            globalOpts = globalOpts.with("array_limit", arrayLimit);
        }

        while (true) {
            try {
                return qry.run(conn, globalOpts);
            } catch (ReqlOpFailedError e) {
                if (System.currentTimeMillis() < startTime + MAX_WAIT_FOR_DB_MILLIS && (
                        e.getMessage().contains("primary replica")
                                || e.getMessage().contains("java.net.SocketException: Operation timed out (Read failed)")
                )) {
                    LOG.error("DB not available at attempt #{}, waiting. Cause: {}", retries, e.toString(), e);
                    try {
                        String dbName = conn.db();
                        if (dbName == null || dbName.isEmpty()) {
                            throw new IllegalStateException("Configured RethinkDB connection is missing a default database");
                        }
                        try (Result<Object> ignored = r.db(dbName)
                                .wait_()
                                .optArg("wait_for", "ready_for_writes")
                                .run(conn)) {
                            // WAIT_COMPLETE returns no value; closing is enough.
                        }
                    } catch (Exception ex) {
                        LOG.warn("Failed waiting for db to have state ready_for_writes. Sleeping for 5 seconds before retry", ex);
                        try {
                            Thread.sleep(5000);
                        } catch (InterruptedException e1) {
                            throw new RuntimeException(e1);
                        }
                    }
                    retries++;
                } else {
                    LOG.warn(e.toString(), e);
                    throw new DbQueryException(e.getMessage(), e);
                }
            } catch (ReqlError e) {
                LOG.warn(e.toString(), e);
                throw new DbQueryException(e.getMessage(), e);
            }
        }
    }

    private Object toUntypedResult(Result<Object> result) throws DbQueryException {
        return switch (result.responseType()) {
            case SUCCESS_ATOM -> readAtom(result);
            case WAIT_COMPLETE -> {
                result.close();
                yield null;
            }
            default -> result;
        };
    }

    public <T extends Message> T executeInsert(String operationName, Insert qry, Class<T> type) throws DbException {
        return executeInsertOrUpdate(operationName, qry, type);
    }

    public <T extends Message> T executeUpdate(String operationName, Update qry, Class<T> type) throws DbException {
        return executeInsertOrUpdate(operationName, qry, type);
    }

    private <T extends Message> T executeInsertOrUpdate(String operationName, ReqlExpr qry, Class<T> type) throws DbException {
        if (qry instanceof Insert) {
            qry = ((Insert) qry).optArg("return_changes", "always");
        } else if (qry instanceof Update) {
            qry = ((Update) qry).optArg("return_changes", "always");
        }

        Map<String, Object> response = executeObject(operationName, qry);
        @SuppressWarnings("unchecked")
        List<Map<String, Map<String, Object>>> changes = (List<Map<String, Map<String, Object>>>) response.get("changes");

        Map<String, Object> newDoc = changes.get(0).get("new_val");
        return ProtoUtils.rethinkToProto(newDoc, type);
    }

    public <T extends Message> T executeGet(String operationName, Get qry, Class<T> type) throws DbException {
        Map<String, Object> response = executeObject(operationName, qry);

        if (response == null) {
            return null;
        }

        return ProtoUtils.rethinkToProto(response, type);
    }

    @Override
    public void close() {
        conn.close();
    }

    public Connection getConnection() {
        return conn;
    }

    @Override
    public ConfigAdapter getConfigAdapter() {
        return configAdapter;
    }

    @Override
    public ExecutionsAdapter getExecutionsAdapter() {
        return executionsAdapter;
    }

    @Override
    public DbQueryAdapter getDbQueryAdapter() {
        return dbQueryAdapter;
    }

    @Override
    public FrontierAdapter getFrontierAdapter() {
        return frontierAdapter;
    }

    @Override
    public DbInitializer getDbInitializer() {
        return dbInitializer;
    }

    @Override
    public void connect(CommonSettings settings) throws DbConnectionException {
        conn = connect(settings.getDbHost(), settings.getDbPort(), settings.getDbName(), settings.getDbUser(),
                settings.getDbPassword(), 30);

        configAdapter = new RethinkDbConfigAdapter(this);
        executionsAdapter = new RethinkDbExecutionsAdapter(this);
        dbQueryAdapter = new RethinkDbQueryAdapter(this);
        frontierAdapter = new RethinkDbFrontierAdapter(this);
        dbInitializer = new RethinkDbInitializer(this);
    }

    private Connection connect(String dbHost, int dbPort, String dbName, String dbUser, String dbPassword,
                               int reConnectAttempts) throws DbConnectionException {
        Connection c = null;
        int attempts = 0;
        while (c == null) {
            attempts++;
            try {
                c = r.connection()
                        .hostname(dbHost)
                        .port(dbPort)
                        .db(dbName)
                        .user(dbUser, dbPassword)
                        .connect();
            } catch (ReqlDriverError e) {
                LOG.warn(e.getMessage());
                if (attempts < reConnectAttempts) {
                    try {
                        Thread.sleep(1000);
                    } catch (InterruptedException ex) {
                        throw new RuntimeException(ex);
                    }
                } else {
                    LOG.error("Too many connection attempts, giving up");
                    throw new DbConnectionException("Too many connection attempts", e);
                }
            }
        }
        return new ConnectionTracingInterceptor(c, true);
    }

    private int getArrayLimit() {
        return Integer.parseInt(System.getProperty(RETHINK_ARRAY_LIMIT_KEY, "0"));
    }

    @SuppressWarnings("unchecked")
    private <T> DbResultSet<T> toSequenceResult(String operationName, Result<Object> result) throws DbQueryException {
        return switch (result.responseType()) {
            case SUCCESS_ATOM -> {
                Object atomResult = readAtom(result);
                result.close();
                yield RethinkDbResultSet.fromIterator((Iterator<T>) toIterator(atomResult));
            }
            case WAIT_COMPLETE -> {
                result.close();
                yield RethinkDbResultSet.fromIterator(Collections.emptyIterator());
            }
            default -> RethinkDbResultSet.fromResult((Result<T>) result);
        };
    }

    private Object readRequiredAtom(String operationName, Result<Object> result) throws DbQueryException {
        return switch (result.responseType()) {
            case SUCCESS_ATOM -> readAtom(result);
            case WAIT_COMPLETE -> null;
            default -> throw new DbQueryException(
                    "Expected atom result for operation '" + operationName + "' but got " + result.responseType());
        };
    }

    private Iterator<?> toIterator(Object value) {
        if (value instanceof Iterable<?> iterable) {
            return iterable.iterator();
        }
        if (value != null && value.getClass().isArray()) {
            if (value instanceof Object[] objectArray) {
                return Arrays.asList(objectArray).iterator();
            }
            int length = Array.getLength(value);
            List<Object> values = new ArrayList<>(length);
            for (int index = 0; index < length; index++) {
                values.add(Array.get(value, index));
            }
            return values.iterator();
        }
        return Collections.singleton(value).iterator();
    }

    private Object readAtom(Result<Object> result) throws DbQueryException {
        Object atomResult = result.first();
        if (atomResult instanceof Map<?, ?> atomResultMap
                && atomResultMap.containsKey("errors")
                && !atomResultMap.get("errors").equals(0L)) {
            DbQueryException ex = new DbQueryException((String) atomResultMap.get("first_error"));
            LOG.error(ex.toString(), ex);
            throw ex;
        }
        return atomResult;
    }

}
