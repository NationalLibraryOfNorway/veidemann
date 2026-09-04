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
package no.nb.nna.veidemann.db;

import com.rethinkdb.ast.ReqlAst;
import com.rethinkdb.gen.exc.ReqlDriverError;
import com.rethinkdb.gen.exc.ReqlOpFailedError;
import com.rethinkdb.gen.exc.ReqlQueryLogicError;
import com.rethinkdb.gen.proto.ResponseType;
import com.rethinkdb.model.OptArgs;
import com.rethinkdb.net.Connection;
import com.rethinkdb.net.Result;
import no.nb.nna.veidemann.commons.db.DbConnectionException;
import no.nb.nna.veidemann.commons.db.DbQueryException;
import no.nb.nna.veidemann.commons.db.DbResultSet;
import no.nb.nna.veidemann.commons.settings.CommonSettings;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.CyclicBarrier;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.assertj.core.api.Assertions.catchThrowable;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyBoolean;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.same;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class RethinkDbConnectionTest {

    @Test
    void replacesBrokenConnectionWithoutRetryingTheQuery() {
        Connection failed = openConnection();
        Connection replacement = connectedReplacement();
        AtomicInteger creations = new AtomicInteger();
        try (RethinkDbConnection db = new RethinkDbConnection(failed, () -> {
            creations.incrementAndGet();
            return replacement;
        })) {
            ReqlAst query = mock(ReqlAst.class);
            when(query.run(same(failed), any(OptArgs.class)))
                    .thenThrow(new ReqlDriverError("Reached the end of the read stream."));

            assertThatThrownBy(() -> db.exec("test", query))
                    .isInstanceOf(DbQueryException.class)
                    .hasMessageContaining("Reached the end of the read stream");

            assertThat(db.getConnection()).isSameAs(replacement);
            assertThat(creations).hasValue(1);
            verify(failed, never()).connect();
            verify(failed).close(false);
            verify(replacement).connect();
            verify(query).run(same(failed), any(OptArgs.class));
            verify(query, never()).run(same(replacement), any(OptArgs.class));
        }
    }

    @Test
    void neverReconnectsAnExistingDriverConnection() throws Exception {
        Connection failed = mock(Connection.class);
        when(failed.isOpen()).thenReturn(false);
        when(failed.connect()).thenThrow(new ReqlDriverError("Client already connected!"));
        Connection replacement = connectedReplacement();
        try (RethinkDbConnection db = new RethinkDbConnection(failed, () -> replacement)) {
            ReqlAst query = queryReturning(waitCompleteResult(), replacement);

            assertThat((Object) db.exec("test", query)).isNull();

            assertThat(db.getConnection()).isSameAs(replacement);
            verify(failed, never()).connect();
            verify(replacement).connect();
        }
    }

    @Test
    void concurrentFailuresOnlyCreateOneReplacement() throws Exception {
        int threadCount = 12;
        Connection failed = openConnection();
        Connection replacement = connectedReplacement();
        AtomicInteger creations = new AtomicInteger();
        try (RethinkDbConnection db = new RethinkDbConnection(failed, () -> {
            creations.incrementAndGet();
            return replacement;
        })) {
            ReqlAst query = mock(ReqlAst.class);
            CyclicBarrier allQueriesStarted = new CyclicBarrier(threadCount);
            when(query.run(same(failed), any(OptArgs.class))).thenAnswer(invocation -> {
                allQueriesStarted.await(5, TimeUnit.SECONDS);
                throw new ReqlDriverError("Reached the end of the read stream.");
            });

            ExecutorService executor = Executors.newFixedThreadPool(threadCount);
            try {
                List<Future<?>> callers = new ArrayList<>();
                for (int i = 0; i < threadCount; i++) {
                    callers.add(executor.submit(() -> {
                        assertThatThrownBy(() -> db.exec("test", query))
                                .isInstanceOf(DbQueryException.class);
                    }));
                }
                for (Future<?> caller : callers) {
                    caller.get(10, TimeUnit.SECONDS);
                }
            } finally {
                executor.shutdownNow();
            }

            assertThat(creations).hasValue(1);
            assertThat(db.getConnection()).isSameAs(replacement);
            verify(failed).close(false);
            verify(replacement).connect();
            verify(replacement, never()).close(anyBoolean());

            ReqlAst followUp = queryReturning(waitCompleteResult(), replacement);
            assertThat((Object) db.exec("follow-up", followUp)).isNull();
            verify(followUp).run(same(replacement), any(OptArgs.class));
        }
    }

    @Test
    void failedReplacementIsNotPublishedAndACallCanTryAgain() throws Exception {
        Connection failed = openConnection();
        Connection rejected = mock(Connection.class);
        when(rejected.connect()).thenThrow(new ReqlDriverError("Connection timed out."));
        Connection replacement = connectedReplacement();
        AtomicInteger creations = new AtomicInteger();
        try (RethinkDbConnection db = new RethinkDbConnection(failed,
                () -> creations.getAndIncrement() == 0 ? rejected : replacement)) {
            ReqlAst failingQuery = mock(ReqlAst.class);
            when(failingQuery.run(same(failed), any(OptArgs.class)))
                    .thenThrow(new ReqlDriverError("Reached the end of the read stream."));

            assertThatThrownBy(() -> db.exec("test", failingQuery))
                    .isInstanceOf(DbConnectionException.class)
                    .hasMessageContaining("Failed connecting to RethinkDB");
            assertThat(db.getConnection()).isNull();
            verify(rejected).close(false);

            ReqlAst followUp = queryReturning(waitCompleteResult(), replacement);
            assertThat((Object) db.exec("follow-up", followUp)).isNull();
            assertThat(db.getConnection()).isSameAs(replacement);
            assertThat(creations).hasValue(2);
        }
    }

    @Test
    void queryErrorsDoNotReplaceAnOpenConnection() {
        Connection connection = openConnection();
        AtomicInteger creations = new AtomicInteger();
        try (RethinkDbConnection db = new RethinkDbConnection(connection, () -> {
            creations.incrementAndGet();
            return connectedReplacement();
        })) {
            ReqlAst query = mock(ReqlAst.class);
            when(query.run(same(connection), any(OptArgs.class)))
                    .thenThrow(new ReqlQueryLogicError("Expected type NUMBER but found STRING."));

            assertThatThrownBy(() -> db.exec("test", query))
                    .isInstanceOf(DbQueryException.class)
                    .hasMessageContaining("Expected type NUMBER");

            assertThat(creations).hasValue(0);
            assertThat(db.getConnection()).isSameAs(connection);
            verify(connection, never()).close(anyBoolean());
        }
    }

    @Test
    void nonTransportDriverErrorsDoNotReplaceAnOpenConnection() {
        Connection connection = openConnection();
        AtomicInteger creations = new AtomicInteger();
        try (RethinkDbConnection db = new RethinkDbConnection(connection, () -> {
            creations.incrementAndGet();
            return connectedReplacement();
        })) {
            ReqlAst query = mock(ReqlAst.class);
            when(query.run(same(connection), any(OptArgs.class)))
                    .thenThrow(new ReqlDriverError("Invalid driver option."));

            assertThatThrownBy(() -> db.exec("test", query))
                    .isInstanceOf(DbQueryException.class)
                    .hasMessageContaining("Invalid driver option");

            assertThat(creations).hasValue(0);
            assertThat(db.getConnection()).isSameAs(connection);
            verify(connection, never()).close(anyBoolean());
        }
    }

    @Test
    void operationTimedOutReadFailureReplacesConnectionWithoutRetryingQuery() {
        Connection failed = openConnection();
        Connection replacement = connectedReplacement();
        AtomicInteger creations = new AtomicInteger();
        try (RethinkDbConnection db = new RethinkDbConnection(failed, () -> {
            creations.incrementAndGet();
            return replacement;
        })) {
            ReqlAst query = mock(ReqlAst.class);
            when(query.run(same(failed), any(OptArgs.class)))
                    .thenThrow(new ReqlOpFailedError(
                            "java.net.SocketException: Operation timed out (Read failed)"));

            assertThatThrownBy(() -> db.exec("test", query))
                    .isInstanceOf(DbQueryException.class)
                    .hasMessageContaining("Operation timed out");

            assertThat(db.getConnection()).isSameAs(replacement);
            assertThat(creations).hasValue(1);
            verify(failed).close(false);
            verify(replacement).connect();
            verify(query).run(same(failed), any(OptArgs.class));
            verify(query, never()).run(same(replacement), any(OptArgs.class));
        }
    }

    @Test
    void otherOperationFailuresDoNotReplaceAnOpenConnection() {
        Connection connection = openConnection();
        AtomicInteger creations = new AtomicInteger();
        try (RethinkDbConnection db = new RethinkDbConnection(connection, () -> {
            creations.incrementAndGet();
            return connectedReplacement();
        })) {
            ReqlAst query = mock(ReqlAst.class);
            when(query.run(same(connection), any(OptArgs.class)))
                    .thenThrow(new ReqlOpFailedError("Table is not available."));

            assertThatThrownBy(() -> db.exec("test", query))
                    .isInstanceOf(DbQueryException.class)
                    .hasMessageContaining("Table is not available");

            assertThat(creations).hasValue(0);
            assertThat(db.getConnection()).isSameAs(connection);
            verify(connection, never()).close(anyBoolean());
        }
    }

    @Test
    void closeDuringRecoveryDiscardsReplacementAndUnblocksWaiters() throws Exception {
        Connection failed = openConnection();
        Connection replacement = openConnection();
        CountDownLatch connectStarted = new CountDownLatch(1);
        CountDownLatch allowConnect = new CountDownLatch(1);
        when(replacement.connect()).thenAnswer(invocation -> {
            connectStarted.countDown();
            if (!allowConnect.await(5, TimeUnit.SECONDS)) {
                throw new AssertionError("Timed out waiting to finish replacement connection");
            }
            return replacement;
        });
        AtomicInteger creations = new AtomicInteger();
        RethinkDbConnection db = new RethinkDbConnection(failed, () -> {
            creations.incrementAndGet();
            return replacement;
        });
        ReqlAst query = mock(ReqlAst.class);
        when(query.run(same(failed), any(OptArgs.class)))
                .thenThrow(new ReqlDriverError("Reached the end of the read stream."));

        ExecutorService executor = Executors.newFixedThreadPool(2);
        try {
            Future<Throwable> recoveryOwner = executor.submit(
                    () -> catchThrowable(() -> db.exec("owner", query)));
            assertThat(connectStarted.await(5, TimeUnit.SECONDS)).isTrue();

            CountDownLatch waiterStarted = new CountDownLatch(1);
            Future<Throwable> recoveryWaiter = executor.submit(() -> {
                waiterStarted.countDown();
                return catchThrowable(() -> db.exec("waiter", query));
            });
            assertThat(waiterStarted.await(5, TimeUnit.SECONDS)).isTrue();

            db.close();

            assertThat(recoveryWaiter.get(5, TimeUnit.SECONDS))
                    .isInstanceOf(DbConnectionException.class)
                    .hasMessageContaining("closed");

            allowConnect.countDown();
            assertThat(recoveryOwner.get(5, TimeUnit.SECONDS))
                    .isInstanceOf(DbConnectionException.class)
                    .hasMessageContaining("closed during replacement");
        } finally {
            allowConnect.countDown();
            executor.shutdownNow();
        }

        assertThat(db.getConnection()).isNull();
        assertThat(creations).hasValue(1);
        verify(failed).close(false);
        verify(replacement).connect();
        verify(replacement).close(false);

        assertThatThrownBy(() -> db.exec("after-close", query))
                .isInstanceOf(DbConnectionException.class)
                .hasMessageContaining("closed");
        assertThat(creations).hasValue(1);

        db.close();
        verify(failed).close(false);
        verify(replacement).close(false);
    }

    @Test
    void closeIsIdempotentAndConnectionCannotBeReopened() {
        Connection connection = openConnection();
        AtomicInteger creations = new AtomicInteger();
        RethinkDbConnection db = new RethinkDbConnection(connection, () -> {
            creations.incrementAndGet();
            return connectedReplacement();
        });

        db.close();
        db.close();

        assertThat(db.getConnection()).isNull();
        verify(connection).close();

        ReqlAst query = mock(ReqlAst.class);
        assertThatThrownBy(() -> db.exec("after-close", query))
                .isInstanceOf(DbConnectionException.class)
                .hasMessageContaining("closed");
        assertThatThrownBy(() -> db.connect(mock(CommonSettings.class)))
                .isInstanceOf(DbConnectionException.class)
                .hasMessageContaining("closed");
        assertThat(creations).hasValue(0);
    }

    @Test
    void changefeedFailureRepairsConnectionBeforeFeedIsRecreated() throws Exception {
        Connection failed = openConnection();
        Connection replacement = connectedReplacement();
        try (RethinkDbConnection db = new RethinkDbConnection(failed, () -> replacement)) {
            Result<Object> failedFeed = mockResult();
            when(failedFeed.responseType()).thenReturn(ResponseType.SUCCESS_PARTIAL);
            when(failedFeed.hasNext()).thenReturn(true);
            when(failedFeed.next(anyLong(), any(TimeUnit.class)))
                    .thenThrow(new ReqlDriverError("Reached the end of the read stream."));
            Result<Object> replacementFeed = mockResult();
            when(replacementFeed.responseType()).thenReturn(ResponseType.SUCCESS_PARTIAL);
            when(replacementFeed.hasNext()).thenReturn(false);
            ReqlAst query = mock(ReqlAst.class);
            when(query.run(same(failed), any(OptArgs.class))).thenReturn(failedFeed);
            when(query.run(same(replacement), any(OptArgs.class))).thenReturn(replacementFeed);

            try (DbResultSet<Object> firstFeed = db.executeSequence("changefeed", query)) {
                assertThatThrownBy(() -> firstFeed.stream().iterator().next())
                        .isInstanceOf(ReqlDriverError.class)
                        .hasMessageContaining("Reached the end of the read stream");
            }

            assertThat(db.getConnection()).isSameAs(replacement);
            try (DbResultSet<Object> secondFeed = db.executeSequence("changefeed", query)) {
                assertThat(secondFeed.stream()).isEmpty();
            }
            verify(query).run(same(failed), any(OptArgs.class));
            verify(query).run(same(replacement), any(OptArgs.class));
        }
    }

    private static Connection openConnection() {
        Connection connection = mock(Connection.class);
        when(connection.isOpen()).thenReturn(true);
        return connection;
    }

    private static Connection connectedReplacement() {
        Connection connection = openConnection();
        when(connection.connect()).thenReturn(connection);
        return connection;
    }

    private static ReqlAst queryReturning(Result<Object> result, Connection connection) {
        ReqlAst query = mock(ReqlAst.class);
        when(query.run(same(connection), any(OptArgs.class))).thenReturn(result);
        return query;
    }

    private static Result<Object> waitCompleteResult() {
        Result<Object> result = mockResult();
        when(result.responseType()).thenReturn(ResponseType.WAIT_COMPLETE);
        return result;
    }

    @SuppressWarnings("unchecked")
    private static Result<Object> mockResult() {
        return mock(Result.class);
    }
}
