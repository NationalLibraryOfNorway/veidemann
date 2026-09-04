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

import java.io.EOFException;
import java.lang.reflect.Array;
import java.net.SocketException;
import java.nio.channels.ClosedChannelException;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.Iterator;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CompletionException;
import java.util.concurrent.atomic.AtomicLong;

public class RethinkDbConnection implements DbServiceSPI {
    private static final Logger LOG = LoggerFactory.getLogger(RethinkDbConnection.class);

    private static final long MAX_WAIT_FOR_DB_MILLIS = 1000 * 60 * 30; // Half an hour

    static final String RETHINK_ARRAY_LIMIT_KEY = "RETHINK_ARRAY_LIMIT";

    static final RethinkDB r = RethinkDB.r;

    private final Object connectionLock = new Object();

    private final AtomicLong connectionGeneration = new AtomicLong();

    private volatile Connection conn;

    private volatile ConnectionFactory connectionFactory;

    private CompletableFuture<Connection> connectionRecovery;

    private boolean closed;

    private RethinkDbConfigAdapter configAdapter;

    private RethinkDbExecutionsAdapter executionsAdapter;

    private RethinkDbQueryAdapter dbQueryAdapter;

    private RethinkDbFrontierAdapter frontierAdapter;

    private RethinkDbInitializer dbInitializer;

    @FunctionalInterface
    interface ConnectionFactory {
        Connection create();
    }

    public RethinkDbConnection() {
    }

    RethinkDbConnection(Connection conn, ConnectionFactory connectionFactory) {
        this.conn = conn;
        this.connectionFactory = connectionFactory;
        connectionGeneration.set(1);
    }

    public <T> T exec(ReqlAst qry) throws DbConnectionException, DbQueryException {
        return exec("db-query", qry);
    }

    @SuppressWarnings("unchecked")
    public <T> T exec(String operationName, ReqlAst qry) throws DbConnectionException, DbQueryException {
        QueryResult queryResult = runQuery(operationName, qry);
        try {
            return (T) toUntypedResult(queryResult.result());
        } catch (ReqlDriverError e) {
            handleConnectionFailure(queryResult.connection(), e);
            throw new DbQueryException(e.getMessage(), e);
        }
    }

    public <T> T executeAtom(String operationName, ReqlAst qry, Class<T> type) throws DbConnectionException, DbQueryException {
        QueryResult queryResult = runQuery(operationName, qry);
        try (Result<Object> result = queryResult.result()) {
            try {
                return type.cast(readRequiredAtom(operationName, result));
            } catch (ReqlDriverError e) {
                handleConnectionFailure(queryResult.connection(), e);
                throw new DbQueryException(e.getMessage(), e);
            }
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

    private QueryResult runQuery(String operationName, ReqlAst qry) throws DbConnectionException, DbQueryException {
        Connection connection = ensureConnection();

        int retries = 0;
        long startTime = System.currentTimeMillis();

        OptArgs globalOpts = OptArgs.of(ConnectionTracingInterceptor.OPERATION_NAME_KEY, operationName);
        int arrayLimit = getArrayLimit();
        if (arrayLimit > 0) {
            globalOpts = globalOpts.with("array_limit", arrayLimit);
        }

        while (true) {
            try {
                return new QueryResult(qry.run(connection, globalOpts), connection);
            } catch (ReqlDriverError e) {
                handleConnectionFailure(connection, e);
                LOG.warn(e.toString(), e);
                throw new DbQueryException(e.getMessage(), e);
            } catch (ReqlOpFailedError e) {
                if (isConnectionFailure(connection, e)) {
                    handleConnectionFailure(connection, e);
                    throw new DbQueryException(e.getMessage(), e);
                }
                if (System.currentTimeMillis() < startTime + MAX_WAIT_FOR_DB_MILLIS
                        && messageContains(e, "primary replica")) {
                    LOG.error("DB not available at attempt #{}, waiting. Cause: {}", retries, e.toString(), e);
                    try {
                        String dbName = connection.db();
                        if (dbName == null || dbName.isEmpty()) {
                            throw new IllegalStateException("Configured RethinkDB connection is missing a default database");
                        }
                        try (Result<Object> ignored = r.db(dbName)
                                .wait_()
                                .optArg("wait_for", "ready_for_writes")
                                .run(connection)) {
                            // WAIT_COMPLETE returns no value; closing is enough.
                        }
                    } catch (Exception ex) {
                        if (isConnectionFailure(connection, ex)) {
                            handleConnectionFailure(connection, ex);
                            throw new DbQueryException(ex.getMessage(), ex);
                        }
                        LOG.warn("Failed waiting for db to have state ready_for_writes. Sleeping for 5 seconds before retry", ex);
                        try {
                            Thread.sleep(5000);
                        } catch (InterruptedException e1) {
                            Thread.currentThread().interrupt();
                            throw new DbQueryException("Interrupted while waiting for RethinkDB", e1);
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

    private Connection ensureConnection() throws DbConnectionException {
        Connection current;
        synchronized (connectionLock) {
            if (closed) {
                throw new DbConnectionException("RethinkDB connection is closed");
            }
            current = conn;
        }
        if (current != null && current.isOpen()) {
            return current;
        }
        return replaceConnection(current, new ReqlDriverError("RethinkDB connection is not open"));
    }

    private void handleConnectionFailure(Connection failedConnection, Throwable failure) throws DbConnectionException {
        if (isConnectionFailure(failedConnection, failure)) {
            replaceConnection(failedConnection, failure);
        }
    }

    private void handleDeferredConnectionFailure(Connection failedConnection, RuntimeException failure) {
        try {
            handleConnectionFailure(failedConnection, failure);
        } catch (DbConnectionException replacementFailure) {
            failure.addSuppressed(replacementFailure);
        }
    }

    private Connection replaceConnection(Connection failedConnection, Throwable failure) throws DbConnectionException {
        CompletableFuture<Connection> recovery;
        boolean recoveryOwner = false;
        long failedGeneration = connectionGeneration.get();

        synchronized (connectionLock) {
            if (closed) {
                throw new DbConnectionException("RethinkDB connection is closed", failure);
            }
            Connection current = conn;
            if (current != failedConnection) {
                if (current != null) {
                    return current;
                }
                if (connectionRecovery == null) {
                    throw new DbConnectionException("RethinkDB connection replacement failed", failure);
                }
            } else if (connectionRecovery == null) {
                conn = null;
                connectionRecovery = new CompletableFuture<>();
                recoveryOwner = true;
            }
            recovery = connectionRecovery;
        }

        if (!recoveryOwner) {
            return awaitRecovery(recovery);
        }

        LOG.warn("RethinkDB connection failed; replacing connection generation {}", failedGeneration, failure);
        closeQuietly(failedConnection);

        Connection replacement;
        try {
            replacement = connectNewConnection(1);
        } catch (DbConnectionException e) {
            synchronized (connectionLock) {
                if (connectionRecovery == recovery) {
                    connectionRecovery = null;
                }
            }
            recovery.completeExceptionally(e);
            LOG.error("RethinkDB connection replacement failed; generation {}", failedGeneration, e);
            throw e;
        }

        long replacementGeneration;
        DbConnectionException publicationFailure = null;
        synchronized (connectionLock) {
            if (closed || connectionRecovery != recovery) {
                if (connectionRecovery == recovery) {
                    connectionRecovery = null;
                }
                publicationFailure = new DbConnectionException(
                        closed
                                ? "RethinkDB connection was closed during replacement"
                                : "RethinkDB connection replacement was superseded");
                replacementGeneration = connectionGeneration.get();
            } else {
                replacementGeneration = connectionGeneration.incrementAndGet();
                conn = replacement;
                connectionRecovery = null;
            }
        }

        if (publicationFailure != null) {
            closeQuietly(replacement);
            recovery.completeExceptionally(publicationFailure);
            LOG.debug("Discarded RethinkDB replacement for generation {}", failedGeneration,
                    publicationFailure);
            throw publicationFailure;
        }

        recovery.complete(replacement);
        LOG.info("RethinkDB connection replaced successfully; generation {}", replacementGeneration);
        return replacement;
    }

    private Connection awaitRecovery(CompletableFuture<Connection> recovery) throws DbConnectionException {
        try {
            return recovery.join();
        } catch (CompletionException e) {
            if (e.getCause() instanceof DbConnectionException dbConnectionException) {
                throw dbConnectionException;
            }
            throw new DbConnectionException("RethinkDB connection replacement failed", e.getCause());
        }
    }

    static boolean isConnectionFailure(Connection connection, Throwable failure) {
        if (connection != null && !connection.isOpen()) {
            return true;
        }

        Throwable cause = failure;
        while (cause != null) {
            if (cause instanceof EOFException
                    || cause instanceof SocketException
                    || cause instanceof ClosedChannelException) {
                return true;
            }
            String message = cause.getMessage();
            String normalized = message == null ? "" : message.toLowerCase(Locale.ROOT);
            if (normalized.contains("java.net.socketexception: operation timed out (read failed)")) {
                return true;
            }
            if (cause instanceof ReqlDriverError) {
                if (!normalized.isEmpty()) {
                    if (normalized.contains("reached the end of the read stream")
                            || normalized.contains("client not connected")
                            || normalized.contains("response pump is not running")
                            || normalized.contains("response pump closed")
                            || normalized.contains("connection is closed")
                            || normalized.contains("connection closed")
                            || normalized.contains("socket is closed")
                            || normalized.contains("socket closed")
                            || normalized.contains("closed socket")
                            || normalized.contains("broken pipe")
                            || normalized.contains("connection reset")
                            || normalized.contains("connection abort")) {
                        return true;
                    }
                }
            }
            cause = cause.getCause();
        }
        return false;
    }

    private static boolean messageContains(Throwable failure, String text) {
        return failure.getMessage() != null && failure.getMessage().contains(text);
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
        Connection current;
        CompletableFuture<Connection> recovery;
        synchronized (connectionLock) {
            if (closed) {
                return;
            }
            closed = true;
            current = conn;
            conn = null;
            recovery = connectionRecovery;
            connectionRecovery = null;
        }

        if (recovery != null) {
            recovery.completeExceptionally(new DbConnectionException("RethinkDB connection is closed"));
        }
        if (current != null) {
            current.close();
        }
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
        synchronized (connectionLock) {
            if (closed) {
                throw new DbConnectionException("RethinkDB connection is closed");
            }
        }

        Connection.Builder connectionBuilder = r.connection()
                .hostname(settings.getDbHost())
                .port(settings.getDbPort())
                .db(settings.getDbName())
                .user(settings.getDbUser(), settings.getDbPassword());
        ConnectionFactory configuredFactory = () -> new ConnectionTracingInterceptor(
                new Connection(r.connection(connectionBuilder)), true);
        synchronized (connectionLock) {
            if (closed) {
                throw new DbConnectionException("RethinkDB connection is closed");
            }
            connectionFactory = configuredFactory;
        }

        Connection initialConnection = connectNewConnection(30);
        boolean closedWhileConnecting;
        synchronized (connectionLock) {
            closedWhileConnecting = closed;
            if (!closedWhileConnecting) {
                conn = initialConnection;
                connectionGeneration.set(1);

                configAdapter = new RethinkDbConfigAdapter(this);
                executionsAdapter = new RethinkDbExecutionsAdapter(this);
                dbQueryAdapter = new RethinkDbQueryAdapter(this);
                frontierAdapter = new RethinkDbFrontierAdapter(this);
                dbInitializer = new RethinkDbInitializer(this);
            }
        }
        if (closedWhileConnecting) {
            closeQuietly(initialConnection);
            throw new DbConnectionException("RethinkDB connection was closed while connecting");
        }
    }

    private Connection connectNewConnection(int connectAttempts) throws DbConnectionException {
        ConnectionFactory factory = connectionFactory;
        if (factory == null) {
            throw new DbConnectionException("RethinkDB connection is not configured");
        }

        int attempts = 0;
        while (attempts < connectAttempts) {
            attempts++;
            Connection candidate = null;
            try {
                candidate = factory.create();
                candidate.connect();
                return candidate;
            } catch (RuntimeException e) {
                closeQuietly(candidate);
                LOG.warn(e.getMessage());
                if (attempts < connectAttempts) {
                    try {
                        Thread.sleep(1000);
                    } catch (InterruptedException ex) {
                        Thread.currentThread().interrupt();
                        throw new DbConnectionException("Interrupted while connecting to RethinkDB", ex);
                    }
                } else {
                    throw new DbConnectionException("Failed connecting to RethinkDB after " + attempts + " attempt(s)", e);
                }
            }
        }
        throw new DbConnectionException("Failed connecting to RethinkDB");
    }

    private void closeQuietly(Connection connection) {
        if (connection == null) {
            return;
        }
        try {
            connection.close(false);
        } catch (RuntimeException closeFailure) {
            LOG.debug("Failed closing obsolete RethinkDB connection", closeFailure);
        }
    }

    private int getArrayLimit() {
        return Integer.parseInt(System.getProperty(RETHINK_ARRAY_LIMIT_KEY, "0"));
    }

    @SuppressWarnings("unchecked")
    private <T> DbResultSet<T> toSequenceResult(String operationName, QueryResult queryResult)
            throws DbConnectionException, DbQueryException {
        Result<Object> result = queryResult.result();
        try {
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
                default -> RethinkDbResultSet.fromResult((Result<T>) result,
                        failure -> handleDeferredConnectionFailure(queryResult.connection(), failure));
            };
        } catch (ReqlDriverError e) {
            handleConnectionFailure(queryResult.connection(), e);
            throw new DbQueryException(e.getMessage(), e);
        }
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

    private record QueryResult(Result<Object> result, Connection connection) {
    }

}
