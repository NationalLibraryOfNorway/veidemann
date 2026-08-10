package no.nb.nna.veidemann.db;

import com.rethinkdb.net.Result;
import no.nb.nna.veidemann.commons.db.DbResultSet;
import org.junit.jupiter.api.Test;

import java.util.NoSuchElementException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.when;

class RethinkDbResultSetTest {

    @Test
    void treatsDriverExhaustionAfterHasNextAsNormalCompletion() throws TimeoutException {
        Result<String> result = mockResult();
        when(result.hasNext()).thenReturn(true);
        when(result.next(anyLong(), any(TimeUnit.class)))
                .thenThrow(new NoSuchElementException("No more elements."));

        try (DbResultSet<String> resultSet = RethinkDbResultSet.fromResult(result)) {
            assertThat(resultSet.stream()).isEmpty();
        }
    }

    @Test
    void retriesTimedReads() throws TimeoutException {
        Result<String> result = mockResult();
        when(result.hasNext()).thenReturn(true, true, false);
        when(result.next(anyLong(), any(TimeUnit.class)))
                .thenThrow(new TimeoutException("not ready"))
                .thenReturn("value");

        try (DbResultSet<String> resultSet = RethinkDbResultSet.fromResult(result)) {
            assertThat(resultSet.stream()).containsExactly("value");
        }
    }

    @Test
    void doesNotSwallowConsumerExceptions() throws TimeoutException {
        Result<String> result = mockResult();
        when(result.hasNext()).thenReturn(true);
        when(result.next(anyLong(), any(TimeUnit.class))).thenReturn("value");

        try (DbResultSet<String> resultSet = RethinkDbResultSet.fromResult(result)) {
            assertThatThrownBy(() -> resultSet.stream().forEach(value -> {
                throw new NoSuchElementException("consumer failed");
            })).isInstanceOf(NoSuchElementException.class)
                    .hasMessage("consumer failed");
        }
    }

    @Test
    void closingTheResultSetCancelsTheDriverResult() {
        Result<String> result = mockResult();
        DbResultSet<String> resultSet = RethinkDbResultSet.fromResult(result);

        resultSet.close();

        verify(result).close();
    }

    @SuppressWarnings("unchecked")
    private static Result<String> mockResult() {
        return mock(Result.class);
    }
}
